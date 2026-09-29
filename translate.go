package gogtrans

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// TranslateOptions 免费翻译配置
type TranslateOptions struct {
	Target    string            // 目标语言，默认 zh-CN
	Source    string            // 源语言，默认 auto
	Format    string            // 文本格式 text|html，默认 html
	UserAgent string            // 用户代理
	Domain    string            // 域名，默认 com
	Proxies   map[string]string // 代理
	Timeout   time.Duration     // 超时时间
	TrustEnv  bool              // 是否信任环境变量
}

// Translate 免费翻译客户端
type Translate struct {
	target    string
	source    string
	format    string
	userAgent string
	timeout   time.Duration
	baseURL   string
	detectURL string
	transURL  string
	ttsURL    string
	client    *http.Client
}

// NewTranslate 创建免费翻译客户端
func NewTranslate(opts *TranslateOptions) (*Translate, error) {
	if opts == nil {
		opts = &TranslateOptions{}
	}

	target := opts.Target
	if target == "" {
		target = "zh-CN"
	}
	source := opts.Source
	if source == "" {
		source = "auto"
	}
	format := opts.Format
	if format == "" {
		format = "html"
	}
	domain := opts.Domain
	if domain == "" {
		domain = "com"
	}

	userAgent := opts.UserAgent
	if userAgent == "" {
		userAgent = generateUserAgent()
	}

	transport, err := buildTransport(opts.Proxies)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
	}

	baseURL := "https://translate.google." + domain
	return &Translate{
		target:    target,
		source:    source,
		format:    format,
		userAgent: userAgent,
		timeout:   opts.Timeout,
		baseURL:   baseURL,
		detectURL: baseURL + "/translate_a/single",
		transURL:  baseURL + "/translate_a/t",
		ttsURL:    baseURL + "/translate_tts",
		client:    client,
	}, nil
}

// buildTransport 统一构建 transport，正确处理 http/https/socks5 代理。
// 注意：socks5 只设置 DialContext，http/https 只设置 Proxy，二者不混用。
func buildTransport(proxies map[string]string) (*http.Transport, error) {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	if len(proxies) == 0 {
		return tr, nil
	}
	proxyURL := proxies["https"]
	if proxyURL == "" {
		proxyURL = proxies["http"]
	}
	if proxyURL == "" {
		return tr, nil
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if u.User != nil {
			pw, _ := u.User.Password()
			auth = &proxy.Auth{
				User:     u.User.Username(),
				Password: pw,
			}
		}
		dialer, err := proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("create socks5 dialer: %w", err)
		}
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// proxy.SOCKS5 返回的 dialer 不支持 ctx，忽略即可
			return dialer.Dial(network, addr)
		}
	default:
		tr.Proxy = http.ProxyURL(u)
	}
	return tr, nil
}

func generateUserAgent() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%f", r.Float64())[2:]))
	return fmt.Sprintf(
		"GoogleTranslate/6.%d.0.06.%d (Linux; U; Android %d; %s) ",
		r.Intn(91)+10,
		r.Intn(888888888)+111111111,
		r.Intn(7)+5,
		b64,
	)
}

// Detect 语言检测
func (t *Translate) Detect(ctx context.Context, q string) (*DetectResponse, error) {
	rt, err := t.TranslateOne(ctx, q, "en", "")
	if err != nil {
		return nil, err
	}
	return NewDetectResponse(rt.DetectedSourceLanguage), nil
}

// TranslateOne 翻译单个文本
func (t *Translate) TranslateOne(ctx context.Context, q, target, source string) (*TranslateResponse, error) {
	results, err := t.Translate(ctx, []string{q}, target, source)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("empty result")
	}
	return results[0], nil
}

// Translate 翻译文本（批量）
func (t *Translate) Translate(ctx context.Context, q []string, target, source string) ([]*TranslateResponse, error) {
	if len(q) == 0 {
		return nil, nil
	}

	// 过滤空字符串（Google 免费接口对空字符串返回不友好）
	filtered := make([]string, 0, len(q))
	for _, s := range q {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return nil, nil
	}

	if target == "" {
		target = t.target
	}
	if source == "" {
		source = t.source
	}
	format := t.format
	if format == "" {
		format = "html"
	}

	resp, body, err := doWithRetry(ctx, 3, func(ctx context.Context) (*http.Response, error) {
		return t.doTranslate(ctx, filtered, target, source, format, "1.0")
	})
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NewNull(resp, string(body))
	}

	var raw []any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	results := make([]*TranslateResponse, 0, len(raw))
	for _, item := range raw {
		results = append(results, parseTranslateItem(item))
	}
	return results, nil
}

func (t *Translate) doTranslate(ctx context.Context, q []string, target, source, format, v string) (*http.Response, error) {
	form := url.Values{}
	for _, s := range q {
		form.Add("q", s)
	}

	params := url.Values{}
	params.Set("tl", target)
	params.Set("sl", source)
	params.Set("ie", "UTF-8")
	params.Set("oe", "UTF-8")
	params.Set("client", "at")
	params.Set("dj", "1")
	params.Set("format", format)
	params.Set("v", v)

	fullURL := t.transURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", t.userAgent)

	return t.client.Do(req)
}

func parseTranslateItem(item any) *TranslateResponse {
	switch v := item.(type) {
	case string:
		return &TranslateResponse{TranslatedText: v}
	case []any:
		tr := &TranslateResponse{}
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				tr.TranslatedText = s
			}
		}
		if len(v) > 1 {
			if s, ok := v[1].(string); ok {
				tr.DetectedSourceLanguage = s
			}
		}
		if len(v) > 2 {
			if s, ok := v[2].(string); ok {
				tr.Model = s
			}
		}
		return tr
	default:
		return &TranslateResponse{TranslatedText: fmt.Sprintf("%v", v)}
	}
}

// TTS 文本转语音
func (t *Translate) TTS(ctx context.Context, q, target string) ([]byte, error) {
	if target == "" {
		target = t.target
	}

	params := url.Values{}
	params.Set("ie", "UTF-8")
	params.Set("client", "at")
	params.Set("tl", target)
	params.Set("q", q)

	fullURL := t.ttsURL + "?" + params.Encode()

	resp, body, err := doWithRetry(ctx, 3, func(ctx context.Context) (*http.Response, error) {
		req, e := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if e != nil {
			return nil, e
		}
		req.Header.Set("User-Agent", t.userAgent)
		return t.client.Do(req)
	})
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return body, nil
	}
	return nil, NewNull(resp, string(body))
}

// doWithRetry 通用重试逻辑，尊重 ctx 取消
// 传入的 fn 必须返回未读取 body 的 resp，由本函数负责读取并关闭。
func doWithRetry(ctx context.Context, maxRetries int, fn func(context.Context) (*http.Response, error)) (*http.Response, []byte, error) {
	if maxRetries < 1 {
		maxRetries = 1
	}
	var lastResp *http.Response
	for i := 1; i <= maxRetries; i++ {
		resp, err := fn(ctx)
		if err != nil {
			return nil, nil, err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return resp, nil, fmt.Errorf("read response body: %w", readErr)
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, body, nil
		}
		lastResp = resp
		if i < maxRetries {
			select {
			case <-ctx.Done():
				return lastResp, body, ctx.Err()
			case <-time.After(time.Duration(5*i) * time.Second):
			}
		}
	}
	return lastResp, nil, fmt.Errorf("max retries (%d) exceeded, last status: %d", maxRetries, lastResp.StatusCode)
}

package gogtrans

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	apiBaseURL     = "https://translation.googleapis.com/language/translate/v2"
	apiLanguageURL = apiBaseURL + "/languages"
	apiDetectURL   = apiBaseURL + "/detect"
	apiLimitSize   = 102400
)

// ApiKeyTranslateOptions APIKEY 翻译配置
type ApiKeyTranslateOptions struct {
	APIKey   string
	Target   string
	Source   string
	Format   string
	Model    string
	Proxies  map[string]string
	Timeout  time.Duration
	TrustEnv bool
}

// ApiKeyTranslate APIKEY 翻译客户端
type ApiKeyTranslate struct {
	apiKey  string
	target  string
	source  string
	format  string
	model   string
	timeout time.Duration
	client  *http.Client
}

// NewApiKeyTranslate 创建 APIKEY 翻译客户端
func NewApiKeyTranslate(opts *ApiKeyTranslateOptions) (*ApiKeyTranslate, error) {
	if opts == nil {
		opts = &ApiKeyTranslateOptions{}
	}

	target := opts.Target
	if target == "" {
		target = "zh-CN"
	}
	format := opts.Format
	if format == "" {
		format = "html"
	}
	model := opts.Model
	if model == "" {
		model = "nmt"
	}

	transport, err := buildTransport(opts.Proxies)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
	}

	return &ApiKeyTranslate{
		apiKey:  opts.APIKey,
		target:  target,
		source:  opts.Source,
		format:  format,
		model:   model,
		timeout: opts.Timeout,
		client:  client,
	}, nil
}

// Languages 语言支持列表
func (a *ApiKeyTranslate) Languages(ctx context.Context, target, model string) ([]*LanguageResponse, error) {
	if target == "" {
		target = a.target
	}
	if model == "" {
		model = a.model
	}

	params := url.Values{}
	params.Set("key", a.apiKey)
	params.Set("target", target)
	params.Set("model", model)

	fullURL := apiLanguageURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NewNull(resp, string(body))
	}

	var result struct {
		Data struct {
			Languages []struct {
				Language string `json:"language"`
				Name     string `json:"name"`
			} `json:"languages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	langs := make([]*LanguageResponse, 0, len(result.Data.Languages))
	for _, l := range result.Data.Languages {
		langs = append(langs, &LanguageResponse{Language: l.Language, Name: l.Name})
	}
	return langs, nil
}

// Detect 语言检测（支持批量）
func (a *ApiKeyTranslate) Detect(ctx context.Context, q []string) ([]*DetectResponse, error) {
	if len(q) == 0 {
		return nil, nil
	}

	var results []*DetectResponse

	for _, ql := range splitList(q) {
		for _, qli := range splitListByContentSize(ql) {
			if len(qli) == 0 {
				continue
			}
			form := url.Values{}
			for _, s := range qli {
				form.Add("q", s)
			}

			fullURL := apiDetectURL + "?key=" + url.QueryEscape(a.apiKey)

			resp, body, err := doWithRetry(ctx, 3, func(ctx context.Context) (*http.Response, error) {
				req, e := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, strings.NewReader(form.Encode()))
				if e != nil {
					return nil, e
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return a.client.Do(req)
			})
			if err != nil {
				// 保留最后一次响应信息（若有）
				if resp != nil {
					return nil, NewNull(resp, string(body))
				}
				return nil, err
			}

			if resp.StatusCode != http.StatusOK {
				return nil, NewNull(resp, string(body))
			}

			var result struct {
				Data struct {
					Detections [][]struct {
						Language   string  `json:"language"`
						IsReliable bool    `json:"isReliable"`
						Confidence float64 `json:"confidence"`
					} `json:"detections"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, err
			}
			for _, d := range result.Data.Detections {
				if len(d) > 0 {
					results = append(results, &DetectResponse{
						Language:   d[0].Language,
						IsReliable: d[0].IsReliable,
						Confidence: d[0].Confidence,
					})
				}
			}
		}
	}

	return results, nil
}

// Translate 文本翻译（支持批量）
//
// source 参数语义：
//   - ""      使用默认 source（a.source）
//   - "auto"  显式要求 API 自动检测（不传 source 参数）
//   - 其他    直接使用指定语言代码
func (a *ApiKeyTranslate) Translate(ctx context.Context, q []string, target, source, format, model string) ([]*TranslateResponse, error) {
	if len(q) == 0 {
		return nil, nil
	}

	if target == "" {
		target = a.target
	}
	// 先回退默认，再处理 auto
	if source == "" {
		source = a.source
	}
	if source == "auto" {
		source = ""
	}
	if format == "" {
		format = a.format
	}
	if model == "" {
		model = a.model
	}

	var results []*TranslateResponse

	for _, ql := range splitList(q) {
		for _, qli := range splitListByContentSize(ql) {
			if len(qli) == 0 {
				continue
			}
			form := url.Values{}
			for _, s := range qli {
				form.Add("q", s)
			}

			params := url.Values{}
			params.Set("key", a.apiKey)
			params.Set("target", target)
			if source != "" {
				params.Set("source", source)
			}
			params.Set("format", format)
			params.Set("model", model)

			fullURL := apiBaseURL + "?" + params.Encode()

			resp, body, err := doWithRetry(ctx, 3, func(ctx context.Context) (*http.Response, error) {
				req, e := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, strings.NewReader(form.Encode()))
				if e != nil {
					return nil, e
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return a.client.Do(req)
			})
			if err != nil {
				if resp != nil {
					return nil, NewNull(resp, string(body))
				}
				return nil, err
			}

			if resp.StatusCode != http.StatusOK {
				return nil, NewNull(resp, string(body))
			}

			var result struct {
				Data struct {
					Translations []struct {
						TranslatedText         string `json:"translatedText"`
						DetectedSourceLanguage string `json:"detectedSourceLanguage"`
						Model                  string `json:"model"`
					} `json:"translations"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, err
			}
			for _, t := range result.Data.Translations {
				results = append(results, &TranslateResponse{
					TranslatedText:         t.TranslatedText,
					DetectedSourceLanguage: t.DetectedSourceLanguage,
					Model:                  t.Model,
				})
			}
		}
	}

	return results, nil
}

// 工具函数

// splitList 按数量切分列表，subSize 可选，默认 128
func splitList(list []string, subSize ...int) [][]string {
	size := 128
	if len(subSize) > 0 && subSize[0] > 0 {
		size = subSize[0]
	}
	if len(list) == 0 {
		return nil
	}
	var result [][]string
	for i := 0; i < len(list); i += size {
		end := min(i+size, len(list))
		result = append(result, list[i:end])
	}
	return result
}

// splitListByContentSize 按内容大小切分列表，contentSize 可选，默认 apiLimitSize
func splitListByContentSize(list []string, contentSize ...int) [][]string {
	size := apiLimitSize
	if len(contentSize) > 0 && contentSize[0] > 0 {
		size = contentSize[0]
	}
	if len(list) == 0 {
		return nil
	}
	if len(list) == 1 || len(strings.Join(list, "")) <= size {
		return [][]string{list}
	}

	mid := int(math.Ceil(float64(len(list)) / 2))
	var result [][]string
	result = append(result, splitListByContentSize(list[:mid], size)...)
	result = append(result, splitListByContentSize(list[mid:], size)...)
	return result
}

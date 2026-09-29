# gogtrans

谷歌翻译 Go 语言实现，支持 **APIKEY**。

> 这是 [pygtrans](https://github.com/foyoux/pygtrans) 的 Go 语言复刻版本。

[![Go Reference](https://pkg.go.dev/badge/github.com/axiaoxin-com/gogtrans.svg)](https://pkg.go.dev/github.com/axiaoxin-com/gogtrans)

## 特性

- **完全免费** 的 `Translate` 客户端（Google 网页翻译接口）
- **APIKEY** 版本的 `ApiKeyTranslate` 客户端（Google Cloud Translation API）
- 支持 **批量翻译**、**语言检测**、**文本转语音（TTS）**
- 支持 **HTTP / HTTPS / SOCKS5** 代理
- 内建 429 重试、`context.Context` 取消与超时
- 内置 240+ 种源语言、240+ 种目标语言
- 与 pygtrans 接口语义一一对应

## 安装

```shell
go get github.com/axiaoxin-com/gogtrans
```

依赖 Go 1.26+。

## 快速入门

### 免费翻译

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/axiaoxin-com/gogtrans"
)

func main() {
	ctx := context.Background()

	client, err := gogtrans.NewTranslate(&gogtrans.TranslateOptions{
		Target: "zh-CN", // 默认目标语言
		Source: "auto",  // 默认源语言
		Proxies: map[string]string{
			"https": "http://localhost:8118", // 代理地址
		},
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 语言检测
	d, err := client.Detect(ctx, "やめて")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(d.Language) // ja

	// 单句翻译
	text, err := client.TranslateOne(ctx, "Look at these pictures and answer the questions.", "", "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(text.TranslatedText) // 看这些图片，回答问题。

	// 批量翻译
	texts, err := client.Translate(ctx, []string{
		"Good morning. What can I do for you?",
		"Read aloud and underline the sentences about booking a flight.",
		"May I have your name and telephone number?",
	}, "", "")
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range texts {
		fmt.Println(t.TranslatedText)
	}

	// 翻译到日语
	ja, _ := client.TranslateOne(ctx, "请多多指教", "ja", "zh-CN")
	fmt.Println(ja.TranslatedText) // お知らせ下さい

	// 翻译到韩语
	ko, _ := client.TranslateOne(ctx, "请多多指教", "ko", "zh-CN")
	fmt.Println(ko.TranslatedText) // 조언 부탁드립니다
}
```

### 文本转语音（TTS）

```go
tts, err := client.TTS(ctx, "やめて", "ja")
if err != nil {
	log.Fatal(err)
}
if err := os.WriteFile("tts.mp3", tts, 0o644); err != nil {
	log.Fatal(err)
}
```

### APIKEY 翻译

需要有效的谷歌翻译 API KEY，可在 [Google Cloud Translation](https://cloud.google.com/translate/docs/quickstarts) 免费试用。

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/axiaoxin-com/gogtrans"
)

func main() {
	ctx := context.Background()

	client, err := gogtrans.NewApiKeyTranslate(&gogtrans.ApiKeyTranslateOptions{
		APIKey: "YOUR_API_KEY",
		Target: "zh-CN",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 获取支持的语言列表（由 API 实时返回）
	langs, err := client.Languages(ctx, "", "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(langs[0])

	// 单句翻译
	texts, err := client.Translate(ctx, []string{"Google Translate"}, "", "", "", "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(texts[0].TranslatedText) // 谷歌翻译

	// 批量翻译
	texts, err = client.Translate(ctx, []string{"안녕하십니까", "こんにちは"}, "", "", "", "")
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range texts {
		fmt.Println(t.TranslatedText, t.DetectedSourceLanguage)
	}

	// 语言检测（支持批量）
	detects, err := client.Detect(ctx, []string{"Hello", "こんにちは"})
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range detects {
		fmt.Println(d.Language)
	}
}
```

## API 速览

### `NewTranslate(*TranslateOptions) (*Translate, error)`

| 字段        | 类型                | 说明                                      |
| ----------- | ------------------- | ----------------------------------------- |
| `Target`    | `string`            | 目标语言，默认 `zh-CN`                    |
| `Source`    | `string`            | 源语言，默认 `auto`                       |
| `Format`    | `string`            | `text` / `html`，默认 `html`              |
| `UserAgent` | `string`            | 自定义 UA，留空则使用内置随机 UA          |
| `Domain`    | `string`            | `google.com` / `google.cn` 等，默认 `com` |
| `Proxies`   | `map[string]string` | 支持 `http` / `https` / `socks5`          |
| `Timeout`   | `time.Duration`     | HTTP 超时                                 |
| `TrustEnv`  | `bool`              | 是否信任环境变量                          |

主要方法：

- `Detect(ctx, q string) (*DetectResponse, error)`
- `TranslateOne(ctx, q, target, source string) (*TranslateResponse, error)`
- `Translate(ctx, q []string, target, source string) ([]*TranslateResponse, error)`
- `TTS(ctx, q, target string) ([]byte, error)`

### `NewApiKeyTranslate(*ApiKeyTranslateOptions) (*ApiKeyTranslate, error)`

| 字段       | 类型                | 说明                               |
| ---------- | ------------------- | ---------------------------------- |
| `APIKey`   | `string`            | 必填                               |
| `Target`   | `string`            | 目标语言，默认 `zh-CN`             |
| `Source`   | `string`            | 源语言，默认不指定（API 自动检测） |
| `Format`   | `string`            | `text` / `html`，默认 `html`       |
| `Model`    | `string`            | `nmt` / `pbmt`，默认 `nmt`         |
| `Proxies`  | `map[string]string` | 代理                               |
| `Timeout`  | `time.Duration`     | HTTP 超时                          |
| `TrustEnv` | `bool`              | 是否信任环境变量                   |

主要方法：

- `Languages(ctx, target, model string) ([]*LanguageResponse, error)`
- `Detect(ctx, q []string) ([]*DetectResponse, error)`
- `Translate(ctx, q []string, target, source, format, model string) ([]*TranslateResponse, error)`

`Translate` 的 `source` 参数语义：

- `""`：使用客户端默认 `source`
- `"auto"`：显式要求 API 自动检测（不传 `source` 参数）
- 其他：直接作为 `source` 传给 API

## 响应类型

```go
type TranslateResponse struct {
	TranslatedText         string
	DetectedSourceLanguage string
	Model                  string
}

type DetectResponse struct {
	Language   string
	IsReliable bool
	Confidence float64
}

type LanguageResponse struct {
	Language string
	Name     string
}
```

## 支持的语言

```go
import "github.com/axiaoxin-com/gogtrans"

// 支持的源语言（约 240+ 种）
for code, lang := range gogtrans.SourceLanguages {
	fmt.Println(code, lang)
}

// 支持的目标语言（约 240+ 种）
for code, lang := range gogtrans.TargetLanguages {
	fmt.Println(code, lang)
}
```

## 必看说明

1. `gogtrans` 包含两个翻译模块：
   1. **`Translate`**：完全免费，支持批量，从 2021 年 9 月 15 日开始需**科学上网**才能使用。
   2. **`ApiKeyTranslate`**：需要有效的谷歌翻译 **API KEY**，[谷歌提供免费试用](https://cloud.google.com/translate/docs/quickstarts)。

2. **`Translate` 的最佳实践**：
   - HTTP 代理：
     ```go
     Proxies: map[string]string{"https": "http://localhost:8118"}
     ```
   - SOCKS5 代理：
     ```go
     Proxies: map[string]string{"https": "socks5://localhost:8119"}
     ```
   - **重要**：尽量一次性多翻译，减少请求次数（如一次性 2000 / 5000 / 10000，甚至 100000 条）。
   - 如果出现 `429 Too Many Requests`，程序会自动重试；仍失败可尝试切换 `Domain`。

3. **免费接口 `User-Agent`**：默认使用随机生成的移动端 UA，稳定性较高。如遇 429，可尝试模仿默认格式构造，或切换 `Domain`。

4. **并发安全**：`Translate` 和 `ApiKeyTranslate` 结构体本身无状态，可安全用于并发场景；但请注意 `http.Client` 的连接池行为。

## 与 pygtrans 的对应关系

| pygtrans                     | gogtrans                       |
| ---------------------------- | ----------------------------- |
| `Translate` 类               | `Translate` 结构体            |
| `ApiKeyTranslate` 类         | `ApiKeyTranslate` 结构体      |
| `TranslateResponse`          | `TranslateResponse` 结构体    |
| `DetectResponse`             | `DetectResponse` 结构体       |
| `LanguageResponse`           | `LanguageResponse` 结构体     |
| `Null`                       | `Null` 结构体                 |
| `SOURCE_LANGUAGES`           | `SourceLanguages` map         |
| `TARGET_LANGUAGES`           | `TargetLanguages` map         |
| `split_list`                 | `splitList` 函数              |
| `split_list_by_content_size` | `splitListByContentSize` 函数 |

## 设计说明

- **错误处理**：优先使用 `(result, error)` 返回模式；同时保留 `Null` 类型，当请求返回非 200 时作为 `error` 返回。
- **上下文**：所有网络请求方法都接收 `context.Context`，支持取消与超时。
- **重试**：内建 `doWithRetry`，遇 429 自动退避重试（最多 3 次），且尊重 `ctx.Done()`。
- **代理**：`buildTransport` 统一处理代理；SOCKS5 只设置 `DialContext`，HTTP/HTTPS 只设置 `Proxy`，避免混用冲突。
- **Body 管理**：所有响应 body 均读取后立即关闭，无泄漏。

## 依赖

- [`golang.org/x/net`](https://pkg.go.dev/golang.org/x/net) —— 用于 SOCKS5 代理支持。

## 构建与运行

```bash
# 拉取依赖
go mod tidy

# 构建
go build ./...

# 运行示例
go run ./_example/main.go
```

## 许可证

GPL-3.0

## 致谢

- 原始 Python 项目：[pygtrans](https://github.com/foyoux/pygtrans) by [@foyoux](https://github.com/foyoux)
- 本项目为其 Go 语言复刻版本，接口与语义尽量保持一致。

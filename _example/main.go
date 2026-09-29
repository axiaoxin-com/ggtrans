package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/axiaoxin-com/gogtrans"
)

func main() {
	ctx := context.Background()

	// 示例1: 免费翻译
	client, err := gogtrans.NewTranslate(&gogtrans.TranslateOptions{
		Target: "zh-CN",
		Source: "auto",
		Proxies: map[string]string{
			"https": "http://localhost:8118",
			// "https": "socks5://localhost:8119",
			"http": "http://localhost:8118",
		},
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("create translate client: %v", err)
	}

	// 检测语言
	d, err := client.Detect(ctx, "やめて")
	if err != nil {
		log.Printf("detect error: %v", err)
	} else {
		fmt.Printf("Detected language: %s\n", d.Language)
	}

	// 翻译单句
	text, err := client.TranslateOne(ctx, "やめて", "", "")
	if err != nil {
		log.Printf("translate error: %v", err)
	} else {
		fmt.Printf("Translated: %s\n", text.TranslatedText)
	}

	// 批量翻译
	texts, err := client.Translate(ctx, []string{
		"你好，我是<a href='https://blog.axiaoxin.com'>axiaoxin</a>",
		"恭喜发财，身体健康",
		"感谢支持，欢迎Star",
	}, "en", "zh-CN")
	if err != nil {
		log.Printf("batch translate error: %v", err)
	} else {
		for _, t := range texts {
			fmt.Printf("  -> %s\n", t.TranslatedText)
		}
	}

	// 翻译到日语
	ja, err := client.TranslateOne(ctx, "请多多指教", "ja", "zh-CN")
	if err != nil {
		log.Printf("translate to ja error: %v", err)
	} else {
		fmt.Printf("Japanese: %s\n", ja.TranslatedText)
	}

	// 翻译到韩语
	ko, err := client.TranslateOne(ctx, "请多多指教", "ko", "zh-CN")
	if err != nil {
		log.Printf("translate to ko error: %v", err)
	} else {
		fmt.Printf("Korean: %s\n", ko.TranslatedText)
	}

	// 文本到语音
	tts, err := client.TTS(ctx, "やめて", "ja")
	if err != nil {
		log.Printf("tts error: %v", err)
	} else {
		if err := os.WriteFile("tts.mp3", tts, 0o644); err != nil {
			log.Printf("write tts file error: %v", err)
		} else {
			fmt.Println("TTS saved to tts.mp3")
		}
	}

	// 示例2: APIKEY 翻译（需要有效的 API KEY）
	// apiClient, err := gogtrans.NewApiKeyTranslate(&gogtrans.ApiKeyTranslateOptions{
	// 	APIKey: "YOUR_API_KEY",
	// 	Target: "zh-CN",
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// langs, err := apiClient.Languages(ctx, "", "")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("First language: %s\n", langs[0])

	// 打印支持的语言
	fmt.Println("\n支持的源语言数量:", len(gogtrans.SourceLanguages))
	fmt.Println("支持的目标语言数量:", len(gogtrans.TargetLanguages))
}

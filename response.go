package gogtrans

import (
	"fmt"
	"net/http"
)

// TranslateResponse 翻译响应
type TranslateResponse struct {
	TranslatedText         string `json:"translatedText"`
	DetectedSourceLanguage string `json:"detectedSourceLanguage,omitempty"`
	Model                  string `json:"model,omitempty"`
}

func (r *TranslateResponse) String() string {
	return fmt.Sprintf("TranslateResponse{translatedText=%q, detectedSourceLanguage=%q, model=%q}",
		r.TranslatedText, r.DetectedSourceLanguage, r.Model)
}

// DetectResponse 语言检测响应
type DetectResponse struct {
	Language   string  `json:"language"`
	IsReliable bool    `json:"isReliable"`
	Confidence float64 `json:"confidence"`
}

func NewDetectResponse(language string) *DetectResponse {
	return &DetectResponse{
		Language:   language,
		IsReliable: true,
		Confidence: 1.0,
	}
}

func (r *DetectResponse) String() string {
	return fmt.Sprintf("DetectResponse{language=%q, isReliable=%t, confidence=%f}",
		r.Language, r.IsReliable, r.Confidence)
}

// LanguageResponse 语言响应
type LanguageResponse struct {
	Language string `json:"language"`
	Name     string `json:"name,omitempty"`
}

func (r *LanguageResponse) String() string {
	return fmt.Sprintf("LanguageResponse{language=%q, name=%q}", r.Language, r.Name)
}

// Null 请求失败时返回的对象
type Null struct {
	Response *http.Response
	Msg      string
}

func NewNull(resp *http.Response, body string) *Null {
	statusText := ""
	code := 0
	if resp != nil {
		code = resp.StatusCode
		statusText = http.StatusText(resp.StatusCode)
	}
	msg := fmt.Sprintf("%d: %s\n%s", code, statusText, body)
	return &Null{
		Response: resp,
		Msg:      msg,
	}
}

func (n *Null) Error() string {
	return n.Msg
}

func (n *Null) String() string {
	return n.Msg
}

// IsNull 判断是否为 Null
func IsNull(v any) bool {
	_, ok := v.(*Null)
	return ok
}

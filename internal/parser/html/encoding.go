package html

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// convertEncoding 根据配置或自动检测将响应体转换为 UTF-8 字符串。
func convertEncoding(body []byte, headers map[string][]string, configured string) (string, string, error) {
	configured = strings.ToLower(configured)
	switch configured {
	case "utf-8":
		return string(body), "utf-8", nil
	case "gbk", "gb2312":
		s, err := decodeGBK(body)
		return s, configured, err
	}

	// auto：从 HTTP 头或 HTML meta 中检测编码
	enc := detectEncoding(body, headers)
	if enc == "gbk" || enc == "gb2312" {
		s, err := decodeGBK(body)
		return s, enc, err
	}
	return string(body), "utf-8", nil
}

// detectEncoding 从 HTTP 头与 HTML meta 标签中检测字符编码。
func detectEncoding(body []byte, headers map[string][]string) string {
	// 1. HTTP Content-Type
	for _, vals := range headers {
		for _, v := range vals {
			if enc := extractCharset(v); enc != "" {
				return normalizeEncoding(enc)
			}
		}
	}

	// 2. HTML meta 标签（只读取前 4KB 避免大文档）
	sample := body
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	html := string(sample)
	if enc := extractCharsetFromMeta(html); enc != "" {
		return normalizeEncoding(enc)
	}

	return "utf-8"
}

// charsetRe 从 Content-Type 等字符串中提取 charset。
var charsetRe = regexp.MustCompile(`(?i)charset\s*=\s*["']?([^"'\s;]+)`)

// extractCharset 从字符串中提取 charset 值。
func extractCharset(s string) string {
	m := charsetRe.FindStringSubmatch(s)
	if len(m) > 1 {
		return strings.ToLower(strings.TrimSpace(m[1]))
	}
	return ""
}

// metaCharsetRe 从 HTML meta 标签中提取 charset。
var metaCharsetRe = regexp.MustCompile(`(?i)<meta[^>]+charset\s*=\s*["']?([^"'\s>]+)`)

// extractCharsetFromMeta 从 HTML 片段中提取 charset。
func extractCharsetFromMeta(html string) string {
	m := metaCharsetRe.FindStringSubmatch(html)
	if len(m) > 1 {
		return strings.ToLower(strings.TrimSpace(m[1]))
	}
	return ""
}

// normalizeEncoding 统一常见编码名称。
func normalizeEncoding(enc string) string {
	enc = strings.ToLower(enc)
	switch enc {
	case "gbk", "gb2312", "gb18030", "x-gbk":
		return "gbk"
	case "utf-8", "utf8":
		return "utf-8"
	default:
		return enc
	}
}

// decodeGBK 将 GBK/GB2312 字节流解码为 UTF-8 字符串。
func decodeGBK(data []byte) (string, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return "", fmt.Errorf("GBK 解码失败: %w", err)
	}
	return buf.String(), nil
}

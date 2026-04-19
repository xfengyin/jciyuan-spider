// Package utils - 工具函数
package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// ============================================================================
// 字符串处理
// ============================================================================

// CleanText 清理文本
func CleanText(s string) string {
	// 移除HTML标签
	s = RemoveHTMLTags(s)
	
	// 替换空白字符
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.TrimSpace(s)
	
	// 合并多余空格
	spaceRe := regexp.MustCompile(`\s+`)
	s = spaceRe.ReplaceAllString(s, " ")
	
	return s
}

// RemoveHTMLTags 移除HTML标签
func RemoveHTMLTags(s string) string {
	tagRe := regexp.MustCompile(`<[^>]+>`)
	s = tagRe.ReplaceAllString(s, "")
	
	// 解码HTML实体
	s = DecodeHTMLEntities(s)
	
	return s
}

// DecodeHTMLEntities 解码HTML实体
func DecodeHTMLEntities(s string) string {
	entities := map[string]string{
		"&nbsp;":  " ",
		"&amp;":   "&",
		"&lt;":    "<",
		"&gt;":    ">",
		"&quot;":  "\"",
		"&apos;":  "'",
		"&#39;":   "'",
		"&mdash;": "—",
		"&ndash;": "–",
		"&hellip;": "…",
		"&copy;":  "©",
		"&reg;":   "®",
		"&trade;": "™",
	}
	
	for entity, char := range entities {
		s = strings.ReplaceAll(s, entity, char)
	}
	
	// 解码数字实体
	numRe := regexp.MustCompile(`&#(\d+);`)
	s = numRe.ReplaceAllStringFunc(s, func(match string) string {
		var num int
		fmt.Sscanf(match[2:len(match)-1], "%d", &num)
		return string(rune(num))
	})
	
	return s
}

// RemoveExtraSpaces 移除多余空格
func RemoveExtraSpaces(s string) string {
	spaceRe := regexp.MustCompile(`\s+`)
	return spaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
}

// ============================================================================
// 正则提取
// ============================================================================

// ExtractString 正则提取字符串
func ExtractString(pattern, input string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ExtractStrings 正则提取多个字符串
func ExtractStrings(pattern, input string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(input, -1)
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			result = append(result, m[1])
		}
	}
	return result
}

// ExtractInt 正则提取整数
func ExtractInt(pattern, input string) int {
	s := ExtractString(pattern, input)
	var num int
	fmt.Sscanf(s, "%d", &num)
	return num
}

// ============================================================================
// URL处理
// ============================================================================

// BuildURL 构建完整URL
func BuildURL(baseURL, path string) string {
	if strings.HasPrefix(path, "http") {
		return path
	}
	if strings.HasPrefix(path, "//") {
		return "https:" + path
	}
	if strings.HasPrefix(path, "/") {
		return strings.TrimSuffix(baseURL, "/") + path
	}
	return baseURL + "/" + path
}

// IsValidURL 验证URL有效性
func IsValidURL(url string) bool {
	if url == "" {
		return false
	}
	urlRe := regexp.MustCompile(`^https?://[^\s]+$`)
	return urlRe.MatchString(url)
}

// ============================================================================
// 数字处理
// ============================================================================

// ParseInt 安全解析整数
func ParseInt(s string) int {
	var result int
	fmt.Sscanf(strings.TrimSpace(s), "%d", &result)
	return result
}

// ParseInt64 安全解析int64
func ParseInt64(s string) int64 {
	var result int64
	fmt.Sscanf(strings.TrimSpace(s), "%d", &result)
	return result
}

// ============================================================================
// 时间处理
// ============================================================================

// FormatTime 格式化时间
func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// FormatDate 格式化日期
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// ParseDate 解析日期
func ParseDate(s string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02",
		"2006.01.02",
	}
	
	s = strings.TrimSpace(s)
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	
	return time.Now()
}

// ============================================================================
// 随机处理
// ============================================================================

// RandomString 生成随机字符串
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[r.Intn(len(charset))]
	}
	return string(result)
}

// RandomInt 生成随机整数
func RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min) + min
}

// ============================================================================
// MD5处理
// ============================================================================

// MD5 计算MD5
func MD5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// ============================================================================
// 切片处理
// ============================================================================

// UniqueStrings 去重字符串切片
func UniqueStrings(ss []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// ContainsString 检查切片是否包含字符串
func ContainsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// ============================================================================
// 初始化
// ============================================================================

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ============================================================================
// 字符判断
// ============================================================================

// IsChinese 判断是否包含中文
func IsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Scripts["Han"], r) {
			return true
		}
	}
	return false
}

// TruncateString 截断字符串
func TruncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ToString 转换为字符串
func ToString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

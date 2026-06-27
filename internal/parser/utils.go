package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// extractString 正则提取字符串
func extractString(pattern, input string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractInt 正则提取整数
func extractInt(pattern, input string) int {
	s := extractString(pattern, input)
	var num int
	fmt.Sscanf(s, "%d", &num)
	return num
}

// cleanText 清理文本
func cleanText(s string) string {
	s = removeHTMLTags(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.TrimSpace(s)
	spaceRe := regexp.MustCompile(`\s+`)
	s = spaceRe.ReplaceAllString(s, " ")
	return s
}

// removeHTMLTags 移除 HTML 标签
func removeHTMLTags(s string) string {
	tagRe := regexp.MustCompile(`<[^>]+>`)
	s = tagRe.ReplaceAllString(s, "")
	return decodeHTMLEntities(s)
}

// decodeHTMLEntities 解码 HTML 实体
func decodeHTMLEntities(s string) string {
	entities := map[string]string{
		"&nbsp;": " ", "&amp;": "&", "&lt;": "<", "&gt;": ">",
		"&quot;": "\"", "&apos;": "'", "&#39;": "'",
		"&mdash;": "—", "&ndash;": "–", "&hellip;": "…",
	}
	for entity, char := range entities {
		s = strings.ReplaceAll(s, entity, char)
	}
	return s
}

// uniqueStrings 去重字符串切片
func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	result := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}


// Package extractor 提供 Regex/CSS/XPath 三种字段提取器实现。
package extractor

import (
	"context"
	"fmt"
	"regexp"

	"jciyuan-spider-v2/internal/parser"
)

// RegexExtractor 基于正则表达式的字段提取器。
type RegexExtractor struct {
	name    string
	pattern string
	re      *regexp.Regexp
}

// NewRegexExtractor 创建正则提取器。
func NewRegexExtractor(name, pattern string) (*RegexExtractor, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("编译正则表达式失败: %w", err)
	}
	return &RegexExtractor{name: name, pattern: pattern, re: re}, nil
}

// Name 返回提取器名称。
func (e *RegexExtractor) Name() string {
	return e.name
}

// Extract 从 Document 中提取匹配结果，返回 []string。
// 当正则仅含一个捕获组时返回该组内容；否则返回完整匹配。
func (e *RegexExtractor) Extract(ctx context.Context, doc *parser.Document) (interface{}, error) {
	if doc == nil {
		return nil, fmt.Errorf("document 为空")
	}
	matches := e.re.FindAllStringSubmatch(doc.HTML, -1)
	results := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) == 0 {
			continue
		}
		// 仅一个捕获组时返回该组，否则返回完整匹配
		if len(m) == 2 {
			results = append(results, m[1])
		} else {
			results = append(results, m[0])
		}
	}
	return results, nil
}

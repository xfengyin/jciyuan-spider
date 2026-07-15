package processor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// cleanTextProcessor 清理 HTML 标签、HTML 实体与多余空白。
type cleanTextProcessor struct{}

// NewCleanTextProcessor 创建文本清理处理器。
func NewCleanTextProcessor() Processor {
	return &cleanTextProcessor{}
}

// Name 返回处理器名称。
func (p *cleanTextProcessor) Name() string {
	return "clean_text"
}

// Process 清理文本或字符串切片中的每个元素。
func (p *cleanTextProcessor) Process(ctx context.Context, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return cleanText(v), nil
	case []string:
		out := make([]string, len(v))
		for i, s := range v {
			out[i] = cleanText(s)
		}
		return out, nil
	default:
		return value, nil
	}
}

// trimProcessor 去除首尾空白。
type trimProcessor struct{}

// NewTrimProcessor 创建 trim 处理器。
func NewTrimProcessor() Processor {
	return &trimProcessor{}
}

// Name 返回处理器名称。
func (p *trimProcessor) Name() string {
	return "trim"
}

// Process 对字符串或字符串切片执行 TrimSpace。
func (p *trimProcessor) Process(ctx context.Context, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v), nil
	case []string:
		out := make([]string, len(v))
		for i, s := range v {
			out[i] = strings.TrimSpace(s)
		}
		return out, nil
	default:
		return value, nil
	}
}

// splitProcessor 按分隔符分割并取指定索引部分。
type splitProcessor struct {
	separator string
	index     int
}

// NewSplitProcessor 创建 split 处理器。
func NewSplitProcessor(separator string, index int) Processor {
	if separator == "" {
		separator = " "
	}
	if index < 0 {
		index = 0
	}
	return &splitProcessor{separator: separator, index: index}
}

// Name 返回处理器名称。
func (p *splitProcessor) Name() string {
	return "split"
}

// Process 按分隔符分割字符串并返回指定索引部分。
func (p *splitProcessor) Process(ctx context.Context, value interface{}) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return value, nil
	}
	parts := strings.Split(s, p.separator)
	if len(parts) == 0 {
		return "", nil
	}
	if p.index >= len(parts) {
		return parts[len(parts)-1], nil
	}
	return parts[p.index], nil
}

// lowerProcessor 转小写。
type lowerProcessor struct{}

// NewLowerProcessor 创建 lower 处理器。
func NewLowerProcessor() Processor {
	return &lowerProcessor{}
}

// Name 返回处理器名称。
func (p *lowerProcessor) Name() string {
	return "lower"
}

// Process 将字符串或字符串切片转为小写。
func (p *lowerProcessor) Process(ctx context.Context, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return strings.ToLower(v), nil
	case []string:
		out := make([]string, len(v))
		for i, s := range v {
			out[i] = strings.ToLower(s)
		}
		return out, nil
	default:
		return value, nil
	}
}

// upperProcessor 转大写。
type upperProcessor struct{}

// NewUpperProcessor 创建 upper 处理器。
func NewUpperProcessor() Processor {
	return &upperProcessor{}
}

// Name 返回处理器名称。
func (p *upperProcessor) Name() string {
	return "upper"
}

// Process 将字符串或字符串切片转为大写。
func (p *upperProcessor) Process(ctx context.Context, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return strings.ToUpper(v), nil
	case []string:
		out := make([]string, len(v))
		for i, s := range v {
			out[i] = strings.ToUpper(s)
		}
		return out, nil
	default:
		return value, nil
	}
}

// regexReplaceProcessor 正则替换。
type regexReplaceProcessor struct {
	pattern     *regexp.Regexp
	replacement string
}

// NewRegexReplaceProcessor 创建正则替换处理器。
func NewRegexReplaceProcessor(pattern, replacement string) (Processor, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("编译正则失败: %w", err)
	}
	return &regexReplaceProcessor{pattern: re, replacement: replacement}, nil
}

// Name 返回处理器名称。
func (p *regexReplaceProcessor) Name() string {
	return "regex_replace"
}

// Process 对字符串或字符串切片执行正则替换。
func (p *regexReplaceProcessor) Process(ctx context.Context, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return p.pattern.ReplaceAllString(v, p.replacement), nil
	case []string:
		out := make([]string, len(v))
		for i, s := range v {
			out[i] = p.pattern.ReplaceAllString(s, p.replacement)
		}
		return out, nil
	default:
		return value, nil
	}
}

// uniqueProcessor 字符串切片去重。
type uniqueProcessor struct{}

// NewUniqueProcessor 创建去重处理器。
func NewUniqueProcessor() Processor {
	return &uniqueProcessor{}
}

// Name 返回处理器名称。
func (p *uniqueProcessor) Name() string {
	return "unique"
}

// Process 对字符串切片去重，非切片类型直接透传。
func (p *uniqueProcessor) Process(ctx context.Context, value interface{}) (interface{}, error) {
	v, ok := value.([]string)
	if !ok {
		return value, nil
	}
	seen := make(map[string]bool, len(v))
	out := make([]string, 0, len(v))
	for _, s := range v {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out, nil
}

// 正则与工具变量。
var (
	htmlTagRe = regexp.MustCompile(`<[^>]+>`)
	spaceRe   = regexp.MustCompile(`\s+`)
)

// cleanText 清理 HTML 标签、HTML 实体与多余空白。
func cleanText(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = decodeHTMLEntities(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.TrimSpace(s)
	s = spaceRe.ReplaceAllString(s, " ")
	return s
}

// decodeHTMLEntities 解码常见 HTML 实体。
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

package extractor

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"jciyuan-spider/internal/parser"
)

// CSSExtractor 基于 CSS Selector 的字段提取器。
type CSSExtractor struct {
	name  string
	query string
	attr  string
}

// NewCSSExtractor 创建 CSS 提取器。
func NewCSSExtractor(name, query, attr string) *CSSExtractor {
	return &CSSExtractor{name: name, query: query, attr: attr}
}

// Name 返回提取器名称。
func (e *CSSExtractor) Name() string {
	return e.name
}

// Extract 使用 goquery 提取节点文本或属性，返回 []string。
func (e *CSSExtractor) Extract(ctx context.Context, doc *parser.Document) (interface{}, error) {
	if doc == nil {
		return nil, fmt.Errorf("document 为空")
	}
	d, err := goquery.NewDocumentFromReader(strings.NewReader(doc.HTML))
	if err != nil {
		return nil, fmt.Errorf("goquery 解析 HTML 失败: %w", err)
	}

	results := make([]string, 0)
	d.Find(e.query).Each(func(i int, s *goquery.Selection) {
		var v string
		if e.attr != "" {
			v, _ = s.Attr(e.attr)
		} else {
			v = strings.TrimSpace(s.Text())
		}
		if v != "" {
			results = append(results, v)
		}
	})
	return results, nil
}

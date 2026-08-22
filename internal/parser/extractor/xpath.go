package extractor

import (
	"context"
	"fmt"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"

	"jciyuan-spider/internal/parser"
)

// XPathExtractor 基于 XPath 的字段提取器。
type XPathExtractor struct {
	name  string
	query string
	attr  string
}

// NewXPathExtractor 创建 XPath 提取器。
func NewXPathExtractor(name, query, attr string) *XPathExtractor {
	return &XPathExtractor{name: name, query: query, attr: attr}
}

// Name 返回提取器名称。
func (e *XPathExtractor) Name() string {
	return e.name
}

// Extract 使用 htmlquery 提取节点文本或属性，返回 []string。
func (e *XPathExtractor) Extract(ctx context.Context, doc *parser.Document) (interface{}, error) {
	if doc == nil {
		return nil, fmt.Errorf("document 为空")
	}
	root, err := htmlquery.Parse(strings.NewReader(doc.HTML))
	if err != nil {
		return nil, fmt.Errorf("htmlquery 解析 HTML 失败: %w", err)
	}

	nodes, err := htmlquery.QueryAll(root, e.query)
	if err != nil {
		return nil, fmt.Errorf("执行 XPath 失败: %w", err)
	}

	results := make([]string, 0, len(nodes))
	for _, n := range nodes {
		var v string
		if e.attr != "" {
			v = htmlquery.SelectAttr(n, e.attr)
		} else {
			v = strings.TrimSpace(htmlquery.InnerText(n))
		}
		if v != "" {
			results = append(results, v)
		}
	}
	return results, nil
}

// ensure html.Node import 被使用，避免编译器误报。
var _ = (*html.Node)(nil)

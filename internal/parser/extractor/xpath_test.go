package extractor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jciyuan-spider-v2/internal/parser"
)

// TestXPathExtractorText 验证 XPath 提取节点文本。
func TestXPathExtractorText(t *testing.T) {
	e := NewXPathExtractor("title", "//h1/text()", "")
	doc := &parser.Document{HTML: `<html><h1>测试标题</h1></html>`}

	res, err := e.Extract(context.Background(), doc)
	require.NoError(t, err)
	assert.Equal(t, []string{"测试标题"}, res)
}

// TestXPathExtractorAttr 验证 XPath 提取属性值。
func TestXPathExtractorAttr(t *testing.T) {
	e := NewXPathExtractor("cover", "//img/@src", "src")
	doc := &parser.Document{HTML: `<html><img src="https://example.com/cover.jpg"/></html>`}

	res, err := e.Extract(context.Background(), doc)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/cover.jpg"}, res)
}

// TestXPathExtractorMultiple 验证提取多个节点。
func TestXPathExtractorMultiple(t *testing.T) {
	e := NewXPathExtractor("tags", "//span[@class='tag']", "")
	doc := &parser.Document{HTML: `<html><span class="tag">A</span><span class="tag">B</span></html>`}

	res, err := e.Extract(context.Background(), doc)
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B"}, res)
}

// TestXPathExtractorNilDocument 验证空文档返回错误。
func TestXPathExtractorNilDocument(t *testing.T) {
	e := NewXPathExtractor("title", "//h1", "")
	_, err := e.Extract(context.Background(), nil)
	require.Error(t, err)
}

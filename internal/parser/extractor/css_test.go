package extractor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jciyuan-spider-v2/internal/parser"
)

// TestCSSExtractorText 验证 CSS 选择器提取节点文本。
func TestCSSExtractorText(t *testing.T) {
	e := NewCSSExtractor("title", "h1.title", "")
	doc := &parser.Document{HTML: `<html><h1 class="title">  测试标题  </h1></html>`}

	res, err := e.Extract(context.Background(), doc)
	require.NoError(t, err)
	assert.Equal(t, []string{"测试标题"}, res)
}

// TestCSSExtractorAttr 验证 CSS 选择器提取属性值。
func TestCSSExtractorAttr(t *testing.T) {
	e := NewCSSExtractor("cover", "img.poster", "src")
	doc := &parser.Document{HTML: `<html><img class="poster" src="https://example.com/cover.jpg"/></html>`}

	res, err := e.Extract(context.Background(), doc)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com/cover.jpg"}, res)
}

// TestCSSExtractorMultiple 验证提取多个节点。
func TestCSSExtractorMultiple(t *testing.T) {
	e := NewCSSExtractor("tags", "span.tag", "")
	doc := &parser.Document{HTML: `<html><span class="tag">A</span><span class="tag">B</span></html>`}

	res, err := e.Extract(context.Background(), doc)
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B"}, res)
}

// TestCSSExtractorNilDocument 验证空文档返回错误。
func TestCSSExtractorNilDocument(t *testing.T) {
	e := NewCSSExtractor("title", "h1", "")
	_, err := e.Extract(context.Background(), nil)
	require.Error(t, err)
}

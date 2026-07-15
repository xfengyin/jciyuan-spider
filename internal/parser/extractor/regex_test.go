// Package extractor 的单元测试。
package extractor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jciyuan-spider-v2/internal/parser"
)

// TestRegexExtractorSingleGroup 验证仅有一个捕获组时返回捕获内容。
func TestRegexExtractorSingleGroup(t *testing.T) {
	e, err := NewRegexExtractor("title", `<title>([^<]+)</title>`)
	require.NoError(t, err)

	doc := &parser.Document{HTML: `<html><title>测试标题</title></html>`}
	res, err := e.Extract(context.Background(), doc)
	require.NoError(t, err)
	assert.Equal(t, []string{"测试标题"}, res)
}

// TestRegexExtractorMultipleGroups 验证多个捕获组时返回完整匹配。
func TestRegexExtractorMultipleGroups(t *testing.T) {
	e, err := NewRegexExtractor("episode", `/acgplay/(\d+)-(\d+)-(\d+)\.html`)
	require.NoError(t, err)

	doc := &parser.Document{HTML: `<a href="/acgplay/37439-1-5.html">第5集</a>`}
	res, err := e.Extract(context.Background(), doc)
	require.NoError(t, err)
	assert.Equal(t, []string{"/acgplay/37439-1-5.html"}, res)
}

// TestRegexExtractorNoMatch 验证未匹配时返回空切片。
func TestRegexExtractorNoMatch(t *testing.T) {
	e, err := NewRegexExtractor("missing", `不存在`)
	require.NoError(t, err)

	doc := &parser.Document{HTML: `<html></html>`}
	res, err := e.Extract(context.Background(), doc)
	require.NoError(t, err)
	assert.Empty(t, res)
}

// TestRegexExtractorInvalidPattern 验证非法正则会返回错误。
func TestRegexExtractorInvalidPattern(t *testing.T) {
	_, err := NewRegexExtractor("bad", `(?P<invalid`)
	require.Error(t, err)
}

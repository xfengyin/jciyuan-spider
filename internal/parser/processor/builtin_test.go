// Package processor 的单元测试。
package processor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCleanTextProcessor 验证清理 HTML 标签与多余空白。
func TestCleanTextProcessor(t *testing.T) {
	p := NewCleanTextProcessor()
	res, err := p.Process(context.Background(), "<p>  测试  &nbsp; 文本  </p>")
	require.NoError(t, err)
	assert.Equal(t, "测试 文本", res)
}

// TestTrimProcessor 验证去除首尾空白。
func TestTrimProcessor(t *testing.T) {
	p := NewTrimProcessor()
	res, err := p.Process(context.Background(), "  hello  ")
	require.NoError(t, err)
	assert.Equal(t, "hello", res)
}

// TestSplitProcessor 验证按分隔符分割并取指定索引。
func TestSplitProcessor(t *testing.T) {
	p := NewSplitProcessor("_", 1)
	res, err := p.Process(context.Background(), "标题_2024_日本")
	require.NoError(t, err)
	assert.Equal(t, "2024", res)
}

// TestSplitProcessorOutOfRange 验证索引越界时返回最后一部分。
func TestSplitProcessorOutOfRange(t *testing.T) {
	p := NewSplitProcessor("_", 5)
	res, err := p.Process(context.Background(), "a_b")
	require.NoError(t, err)
	assert.Equal(t, "b", res)
}

// TestLowerProcessor 验证转小写。
func TestLowerProcessor(t *testing.T) {
	p := NewLowerProcessor()
	res, err := p.Process(context.Background(), "HELLO")
	require.NoError(t, err)
	assert.Equal(t, "hello", res)
}

// TestUpperProcessor 验证转大写。
func TestUpperProcessor(t *testing.T) {
	p := NewUpperProcessor()
	res, err := p.Process(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "HELLO", res)
}

// TestRegexReplaceProcessor 验证正则替换。
func TestRegexReplaceProcessor(t *testing.T) {
	p, err := NewRegexReplaceProcessor(`\d+`, "[N]")
	require.NoError(t, err)
	res, err := p.Process(context.Background(), "第01集")
	require.NoError(t, err)
	assert.Equal(t, "第[N]集", res)
}

// TestUniqueProcessor 验证切片去重。
func TestUniqueProcessor(t *testing.T) {
	p := NewUniqueProcessor()
	res, err := p.Process(context.Background(), []string{"a", "b", "a", "c"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, res)
}

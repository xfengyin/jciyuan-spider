package extractor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jciyuan-spider-v2/internal/model"
	"jciyuan-spider-v2/internal/parser"
)

// TestFactoryBuildsAllTypes 验证工厂能构造三种选择器。
func TestFactoryBuildsAllTypes(t *testing.T) {
	cases := []struct {
		name string
		cfg  model.SelectorConfig
	}{
		{"regex", model.SelectorConfig{Type: "regex", Value: "test"}},
		{"css", model.SelectorConfig{Type: "css", Value: "h1", Attr: "class"}},
		{"xpath", model.SelectorConfig{Type: "xpath", Value: "//h1", Attr: "id"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ext, err := Build(tc.cfg)
			require.NoError(t, err)
			assert.NotNil(t, ext)
			_, ok := ext.(parser.Extractor)
			assert.True(t, ok)
		})
	}
}

// TestFactoryUnknownType 验证未知选择器类型返回错误。
func TestFactoryUnknownType(t *testing.T) {
	_, err := Build(model.SelectorConfig{Type: "unknown", Value: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未知")
}

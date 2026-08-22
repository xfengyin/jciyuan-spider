package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jciyuan-spider/internal/model"
)

// TestProcessorFactoryBuildsKnownTypes 验证工厂能构造所有内置处理器。
func TestProcessorFactoryBuildsKnownTypes(t *testing.T) {
	cases := []struct {
		name string
		cfg  model.ProcessorConfig
	}{
		{"clean_text", model.ProcessorConfig{Type: "clean_text"}},
		{"trim", model.ProcessorConfig{Type: "trim"}},
		{"split", model.ProcessorConfig{Type: "split", Separator: "_"}},
		{"lower", model.ProcessorConfig{Type: "lower"}},
		{"upper", model.ProcessorConfig{Type: "upper"}},
		{"unique", model.ProcessorConfig{Type: "unique"}},
		{"regex_replace", model.ProcessorConfig{Type: "regex_replace", Params: map[string]string{"pattern": `\d+`, "replacement": "[N]"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := Build(tc.cfg)
			require.NoError(t, err)
			assert.NotNil(t, p)
		})
	}
}

// TestProcessorFactoryUnknownType 验证未知处理器类型返回错误。
func TestProcessorFactoryUnknownType(t *testing.T) {
	_, err := Build(model.ProcessorConfig{Type: "unknown"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未知")
}

// TestProcessorFactoryRegexReplaceMissingPattern 验证缺少 pattern 时返回错误。
func TestProcessorFactoryRegexReplaceMissingPattern(t *testing.T) {
	_, err := Build(model.ProcessorConfig{Type: "regex_replace"})
	require.Error(t, err)
}

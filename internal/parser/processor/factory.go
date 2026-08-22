package processor

import (
	"fmt"
	"strconv"

	"jciyuan-spider/internal/model"
)

// Build 根据 ProcessorConfig 构造对应类型的后处理器。
func Build(cfg model.ProcessorConfig) (Processor, error) {
	switch cfg.Type {
	case "clean_text":
		return NewCleanTextProcessor(), nil
	case "trim":
		return NewTrimProcessor(), nil
	case "split":
		idx := 0
		if v, ok := cfg.Params["index"]; ok {
			if n, err := strconv.Atoi(v); err == nil {
				idx = n
			}
		}
		return NewSplitProcessor(cfg.Separator, idx), nil
	case "lower":
		return NewLowerProcessor(), nil
	case "upper":
		return NewUpperProcessor(), nil
	case "regex_replace":
		pattern := cfg.Params["pattern"]
		replacement := cfg.Params["replacement"]
		if replacement == "" {
			replacement = cfg.Separator
		}
		if pattern == "" {
			return nil, fmt.Errorf("regex_replace 处理器缺少 pattern 参数")
		}
		return NewRegexReplaceProcessor(pattern, replacement)
	case "unique":
		return NewUniqueProcessor(), nil
	default:
		return nil, fmt.Errorf("未知的处理器类型: %s", cfg.Type)
	}
}

package extractor

import (
	"fmt"

	"jciyuan-spider/internal/model"
	"jciyuan-spider/internal/parser"
)

// Build 根据 SelectorConfig 构造对应类型的提取器。
func Build(cfg model.SelectorConfig) (parser.Extractor, error) {
	switch cfg.Type {
	case "regex":
		return NewRegexExtractor(cfg.Type, cfg.Value)
	case "css":
		return NewCSSExtractor(cfg.Type, cfg.Value, cfg.Attr), nil
	case "xpath":
		return NewXPathExtractor(cfg.Type, cfg.Value, cfg.Attr), nil
	default:
		return nil, fmt.Errorf("未知的选择器类型: %s", cfg.Type)
	}
}

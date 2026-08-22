// Package parser 定义解析抽象、字段提取器与 SPI 注册机制，支持 Pipeline 解析。
package parser

import (
	"context"
	"fmt"

	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/model"
)

// ParseResult 解析结果，包含动漫主体、剧集列表与审计用原始 HTML
type ParseResult struct {
	Anime    *model.AnimeInfo
	Episodes []*model.Episode
	RawHTML  []byte // 审计用
}

// Parser 解析器接口，所有 Parser 后端必须实现
type Parser interface {
	// Parse 解析 Fetcher 返回的响应
	Parse(ctx context.Context, resp *fetcher.Response) (*ParseResult, error)
}

// Extractor 字段提取器接口，用于 Pipeline 模型
type Extractor interface {
	// Name 返回提取器名称
	Name() string
	// Extract 从 Document 中提取字段值
	Extract(ctx context.Context, doc *Document) (interface{}, error)
}

// Document 解析上下文，封装原始 HTML 与元数据
type Document struct {
	URL      string
	HTML     string
	Encoding string
	Meta     map[string]interface{}
}

// Builder 构造器函数签名，用于 SPI 注册
type Builder func(cfg model.ParserConfig) (Parser, error)

// registry 存储已注册的 Parser 构造器
var registry = make(map[string]Builder)

// Register 注册 Parser 实现，name 对应配置 parser.type
func Register(name string, b Builder) {
	registry[name] = b
}

// Build 按配置构建 Parser 实例
func Build(cfg model.ParserConfig) (Parser, error) {
	b, ok := registry[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown parser type: %s", cfg.Type)
	}
	return b(cfg)
}

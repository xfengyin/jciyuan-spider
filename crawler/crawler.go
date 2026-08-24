// Package crawler 提供轻量通用 Go 爬虫框架的核心抽象：
// Crawler 接口（Fetch → Parse → Extract）、Engine 调度器与按名称注册的 SPI 机制。
//
// 设计目标
//
//   - 通用：任何站点/数据源只需实现 Crawler 接口即可接入框架，由 Engine 统一调度
//     并发、重试、限速与结果输出，无需关心抓取细节。
//   - 轻量：核心包只依赖标准库；HTTP 客户端、选择器、存储等均可按需替换。
//   - 可扩展：通过 Register/Build 按名称装配 Crawler 实现，支持配置驱动。
//
// 快速上手
//
//	type myCrawler struct{}
//
//	func (c *myCrawler) Name() string { return "my" }
//	func (c *myCrawler) Fetch(ctx context.Context, url string) (*crawler.Page, error) {
//		resp, err := http.Get(url)
//		if err != nil { return nil, err }
//		defer resp.Body.Close()
//		body, _ := io.ReadAll(resp.Body)
//		return &crawler.Page{URL: url, StatusCode: resp.StatusCode, Body: body, Text: string(body)}, nil
//	}
//	func (c *myCrawler) Parse(ctx context.Context, page *crawler.Page) (any, error) { return page.Text, nil }
//	func (c *myCrawler) Extract(ctx context.Context, parsed any) ([]crawler.Item, error) {
//		return []crawler.Item{{"text": parsed}}, nil
//	}
//
//	eng := crawler.NewEngine(&myCrawler{}, crawler.Options{Concurrency: 2, OutputDir: "./out"})
//	result, err := eng.Run(ctx, []string{"https://example.com"})
package crawler

import (
	"context"
	"fmt"
)

// Page 一次抓取得到的原始页面。Body 保留原始字节，Text 为解码后的文本（HTML/JSON 原文）。
type Page struct {
	URL        string
	StatusCode int
	Headers    map[string][]string
	Body       []byte
	Text       string
	Meta       map[string]any
}

// Item 一条结构化抽取结果，字段名到值的映射。
type Item map[string]any

// Crawler 通用爬虫接口：实现该接口的任何站点/数据源都可以被 Engine 调度。
//
// 数据流：Fetch 抓取原始页面 → Parse 解析为中间表示（DOM/文档/对象）→
// Extract 抽取结构化条目。三个方法各自独立，允许按需组合与替换。
type Crawler interface {
	// Name 返回爬虫名称，用于注册、日志与配置识别。
	Name() string
	// Fetch 抓取 url，返回原始页面。失败时返回错误，Engine 会按配置重试。
	Fetch(ctx context.Context, url string) (*Page, error)
	// Parse 将页面解析为中间表示（如 DOM、结构化文档、JSON 对象）。
	Parse(ctx context.Context, page *Page) (any, error)
	// Extract 从中间表示中抽取结构化条目列表，可返回多条。
	Extract(ctx context.Context, parsed any) ([]Item, error)
}

// Builder 构造器签名：根据配置构建一个 Crawler 实例。
type Builder func(cfg map[string]any) (Crawler, error)

// registry 保存已注册的 Crawler 构造器，名称对应配置中的 crawler.type。
var registry = make(map[string]Builder)

// Register 注册 Crawler 实现，name 不可重复。
func Register(name string, b Builder) {
	if _, dup := registry[name]; dup {
		panic("crawler: 重复注册爬虫 " + name)
	}
	registry[name] = b
}

// Build 按名称与配置构建 Crawler 实例。
func Build(name string, cfg map[string]any) (Crawler, error) {
	b, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("crawler: 未注册的爬虫类型 %q（可用的: %v）", name, Names())
	}
	return b(cfg)
}

// Names 返回所有已注册的爬虫名称。
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}

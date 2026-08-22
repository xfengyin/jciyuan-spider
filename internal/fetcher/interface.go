// Package fetcher 定义页面请求抽象与 SPI 注册机制，支持多种后端（http/selenium/playwright）。
package fetcher

import (
	"context"
	"fmt"

	"jciyuan-spider/internal/logger"
	"jciyuan-spider/internal/metrics"
	"jciyuan-spider/internal/model"
)

// Request 请求对象，封装 URL、方法、头、体与透传元数据
type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
	Meta    map[string]interface{} // 透传 traceId、attempt 等
}

// Response 响应对象，封装状态码、响应头、体与耗时
type Response struct {
	URL        string
	StatusCode int
	Headers    map[string][]string
	Body       []byte
	Meta       map[string]interface{}
	Duration   int64 // ms
}

// Fetcher 请求器接口，所有 Fetcher 后端必须实现
type Fetcher interface {
	// Fetch 执行请求并返回响应
	Fetch(ctx context.Context, req *Request) (*Response, error)
	// Close 释放资源
	Close() error
}

// Builder 构造器函数签名，用于 SPI 注册
type Builder func(
	cfg model.FetcherConfig,
	anti model.AnticrawlerConfig,
	spider model.SpiderConfig,
	mws []model.MiddlewareItem,
	m metrics.Metrics,
	l logger.Logger,
) (Fetcher, error)

// registry 存储已注册的 Fetcher 构造器
var registry = make(map[string]Builder)

// Register 注册 Fetcher 实现，name 对应配置 fetcher.type
func Register(name string, b Builder) {
	registry[name] = b
}

// Build 按配置构建 Fetcher 实例
func Build(
	cfg model.FetcherConfig,
	anti model.AnticrawlerConfig,
	spider model.SpiderConfig,
	mws []model.MiddlewareItem,
	m metrics.Metrics,
	l logger.Logger,
) (Fetcher, error) {
	b, ok := registry[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown fetcher type: %s", cfg.Type)
	}
	return b(cfg, anti, spider, mws, m, l)
}

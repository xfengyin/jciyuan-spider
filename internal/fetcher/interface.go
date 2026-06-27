// Package fetcher - HTTP 请求抽象层
package fetcher

import "context"

// 注：单一实现但保留接口以支持 DI 和未来扩展。
// 当前实现：HTTPFetcher

// Fetcher HTTP 请求器接口
type Fetcher interface {
	// Fetch 获取页面内容
	Fetch(ctx context.Context, url string) ([]byte, error)
}

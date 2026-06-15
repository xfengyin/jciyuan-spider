// Package fetcher - HTTP 请求抽象层
package fetcher

import "context"

// Fetcher HTTP 请求器接口
type Fetcher interface {
	// Fetch 获取页面内容
	Fetch(ctx context.Context, url string) ([]byte, error)
}

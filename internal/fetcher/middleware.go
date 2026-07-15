// Package fetcher 的中间件链定义与组合工具。
package fetcher

import "context"

// Handler 是中间件链中处理请求的核心函数签名
type Handler func(ctx context.Context, req *Request) (*Response, error)

// Middleware 是中间件函数签名，负责包装下一个 Handler
// 返回值仍然是 Handler，便于链式组合
type Middleware func(next Handler) Handler

// Compose 将多个中间件组合为一个中间件
func Compose(mws ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

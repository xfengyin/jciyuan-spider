// Package middleware 提供 Fetcher 中间件实现。
package middleware

import (
	"context"
	"sync"
	"time"

	"jciyuan-spider/internal/fetcher"
)

// RateLimitMiddleware 令牌桶限流，按 delayMs 控制相邻请求最小间隔。
// 桶容量固定为 1，令牌以每 delayMs 一个的速率生成，保证请求间隔不低于 delayMs。
func RateLimitMiddleware(delayMs int) fetcher.Middleware {
	interval := time.Duration(delayMs) * time.Millisecond
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}

	var mu sync.Mutex
	var lastRequest time.Time

	return func(next fetcher.Handler) fetcher.Handler {
		return func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
			mu.Lock()
			if elapsed := time.Since(lastRequest); elapsed < interval {
				wait := interval - elapsed
				mu.Unlock()
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			} else {
				mu.Unlock()
			}

			mu.Lock()
			lastRequest = time.Now()
			mu.Unlock()
			return next(ctx, req)
		}
	}
}

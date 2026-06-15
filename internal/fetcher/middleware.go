package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"jciyuan-spider-v2/internal/metrics"
)

// Handler 请求处理函数
type Handler func(ctx context.Context, req *http.Request) ([]byte, error)

// Middleware 中间件函数
type Middleware func(next Handler) Handler

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(delayMs int) Middleware {
	var lastRequest time.Time
	return func(next Handler) Handler {
		return func(ctx context.Context, req *http.Request) ([]byte, error) {
			delay := time.Duration(delayMs) * time.Millisecond
			if elapsed := time.Since(lastRequest); elapsed < delay {
				select {
				case <-time.After(delay - elapsed):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			lastRequest = time.Now()
			return next(ctx, req)
		}
	}
}

// RetryMiddleware 重试中间件
func RetryMiddleware(maxRetry int, m *metrics.Collector) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *http.Request) ([]byte, error) {
			var lastErr error
			for attempt := 0; attempt <= maxRetry; attempt++ {
				if attempt > 0 {
					m.IncrRetry()
					backoff := time.Duration(attempt*500) * time.Millisecond
					select {
					case <-time.After(backoff):
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}

				var body []byte
				body, lastErr = next(ctx, req)
				if lastErr == nil {
					return body, nil
				}

				// 阻止类错误不重试
				if IsBlocked(lastErr) {
					return nil, lastErr
				}
			}
			return nil, fmt.Errorf("重试 %d 次后仍失败: %w", maxRetry, lastErr)
		}
	}
}

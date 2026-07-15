// Package middleware 提供 Fetcher 中间件实现。
package middleware

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	spidererrors "jciyuan-spider-v2/internal/errors"
	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/metrics"
)

// isContextError 判断错误是否由上下文取消/超时产生。
func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// RetryMiddleware 指数退避 + 抖动重试，只对 ErrRetryable 重试。
// BlockedError、CaptchaError 与 ctx.Err() 均不重试。
func RetryMiddleware(maxRetry int, m metrics.Metrics) fetcher.Middleware {
	return func(next fetcher.Handler) fetcher.Handler {
		return func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
			var lastErr error
			for attempt := 0; attempt <= maxRetry; attempt++ {
				if attempt > 0 {
					m.IncrRetry(ctx)
					backoff := calculateBackoff(attempt)
					select {
					case <-time.After(backoff):
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}

				resp, err := next(ctx, req)
				if err == nil {
					return resp, nil
				}
				lastErr = err

				if isContextError(err) || !errors.Is(err, spidererrors.ErrRetryable) {
					return nil, err
				}
			}
			return nil, fmt.Errorf("重试 %d 次后仍失败: %w", maxRetry, lastErr)
		}
	}
}

// calculateBackoff 计算指数退避时间（带抖动）。
func calculateBackoff(attempt int) time.Duration {
	baseDelay := 500 * time.Millisecond
	maxBackoff := 30 * time.Second
	backoff := baseDelay * time.Duration(1<<attempt)
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	// 加入 0~baseDelay 的随机抖动，避免惊群。
	jitter := time.Duration(rand.Int63n(int64(baseDelay)))
	return backoff + jitter
}

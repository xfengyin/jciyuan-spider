// Package middleware 提供 Fetcher 中间件实现。
package middleware

import (
	"context"
	"time"

	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/metrics"
)

// MetricsMiddleware 调用 metrics.Metrics 记录请求数、延迟与状态码。
func MetricsMiddleware(m metrics.Metrics) fetcher.Middleware {
	return func(next fetcher.Handler) fetcher.Handler {
		return func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
			start := time.Now()
			resp, err := next(ctx, req)
			duration := time.Since(start)

			status := "success"
			if err != nil {
				status = "fail"
				m.IncrFail(ctx)
			} else {
				m.IncrSuccess(ctx)
			}
			m.RecordRequestDurationWithStatus(ctx, status, duration)
			return resp, err
		}
	}
}

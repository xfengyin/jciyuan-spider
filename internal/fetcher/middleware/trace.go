// Package middleware 提供 Fetcher 中间件实现。
package middleware

import (
	"context"

	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/logger"
)

// WithTraceID 将 traceId 注入 context，保留旧版入口以兼容现有调用方。
// 内部统一委托给 logger.WithTraceID，确保 context key 一致。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return logger.WithTraceID(ctx, traceID)
}

// TraceIDFromContext 从 context 读取 traceId，供中间件链与业务层使用。
func TraceIDFromContext(ctx context.Context) (string, bool) {
	return logger.TraceIDFromContext(ctx)
}

// TraceMiddleware 从 ctx 读取 traceId 并写入 Request.Meta，同时回写 ctx，保证全链路透传。
func TraceMiddleware() fetcher.Middleware {
	return func(next fetcher.Handler) fetcher.Handler {
		return func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
			if ctx == nil {
				ctx = context.Background()
			}
			if req.Meta == nil {
				req.Meta = make(map[string]interface{})
			}

			traceID, _ := TraceIDFromContext(ctx)
			if traceID == "" {
				if v, ok := req.Meta["traceId"].(string); ok {
					traceID = v
				}
			}
			if traceID != "" {
				req.Meta["traceId"] = traceID
				ctx = WithTraceID(ctx, traceID)
			}
			return next(ctx, req)
		}
	}
}

// Package middleware 提供 Fetcher 中间件实现。
package middleware

import (
	"context"
	"net/url"
	"strings"
	"time"

	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/logger"
)

// LoggingMiddleware 使用 logger.Logger 记录请求 URL、方法、头、状态码、耗时与错误，并自动携带 traceId。
// URL 中的代理认证信息会被脱敏；Cookie、Authorization、Proxy-Authorization 头会被遮蔽。
func LoggingMiddleware(log logger.Logger) fetcher.Middleware {
	return func(next fetcher.Handler) fetcher.Handler {
		return func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
			if ctx == nil {
				ctx = context.Background()
			}
			traceLog := log.WithTrace(ctx)

			start := time.Now()
			resp, err := next(ctx, req)
			duration := time.Since(start).Milliseconds()

			fields := []logger.Field{
				logger.String("url", sanitizeURL(req.URL)),
				logger.String("method", req.Method),
				logger.Any("headers", sanitizeHeaders(req.Headers)),
				logger.Int64("duration_ms", duration),
			}
			if resp != nil {
				fields = append(fields, logger.Int("status_code", resp.StatusCode))
			}
			if err != nil {
				traceLog.Error("fetcher 请求失败", append(fields, logger.Err(err))...)
				return resp, err
			}
			traceLog.Info("fetcher 请求完成", fields...)
			return resp, err
		}
	}
}

// sanitizeURL 移除 URL 中的用户认证信息，避免日志泄露代理账号密码。
func sanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if u.User != nil {
		u.User = nil
	}
	return u.String()
}

// sensitiveHeaders 定义需要脱敏的头部名称（大小写不敏感）。
var sensitiveHeaders = []string{"Cookie", "Authorization", "Proxy-Authorization"}

// sanitizeHeaders 对敏感头部值进行遮蔽处理。
func sanitizeHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		if isSensitiveHeader(k) {
			out[k] = "[REDACTED]"
		} else {
			out[k] = v
		}
	}
	return out
}

// isSensitiveHeader 判断给定头部是否需要脱敏。
func isSensitiveHeader(name string) bool {
	for _, h := range sensitiveHeaders {
		if strings.EqualFold(name, h) {
			return true
		}
	}
	return false
}

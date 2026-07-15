package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/logger"
	"jciyuan-spider-v2/internal/mocks"
)

// stringField 从日志字段中提取字符串值（仅支持 String/Any 类型）。
func stringField(fields []logger.Field, key string) (string, bool) {
	for _, f := range fields {
		if f.Key != key {
			continue
		}
		if f.Type == zapcore.StringType {
			return f.String, true
		}
		if f.Type == zapcore.ReflectType && f.Interface != nil {
			if s, ok := f.Interface.(string); ok {
				return s, true
			}
		}
	}
	return "", false
}

// intField 从日志字段中提取整数值（仅支持 Int64/Int 类型）。
func intField(fields []logger.Field, key string) (int64, bool) {
	for _, f := range fields {
		if f.Key != key {
			continue
		}
		switch f.Type {
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type, zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
			return f.Integer, true
		}
	}
	return 0, false
}

// mapField 从日志字段中提取 map[string]string 值（仅支持 Any 类型）。
func mapField(fields []logger.Field, key string) (map[string]string, bool) {
	for _, f := range fields {
		if f.Key != key {
			continue
		}
		if f.Type == zapcore.ReflectType && f.Interface != nil {
			if m, ok := f.Interface.(map[string]string); ok {
				return m, true
			}
		}
	}
	return nil, false
}

// TestLoggingSanitizesURL 验证 URL 中的认证信息不会被记录。
func TestLoggingSanitizesURL(t *testing.T) {
	log := mocks.NewMockLogger()
	mw := LoggingMiddleware(log)

	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return &fetcher.Response{URL: req.URL, StatusCode: 200}, nil
	})

	_, err := handler(context.Background(), &fetcher.Request{
		URL: "http://user:secret@example.com/path",
	})
	require.NoError(t, err)

	entries := log.Entries()
	require.Len(t, entries, 1)
	urlVal, ok := stringField(entries[0].Fields, "url")
	require.True(t, ok)
	assert.NotContains(t, urlVal, "secret")
	assert.NotContains(t, urlVal, "user:")
	assert.Equal(t, "http://example.com/path", urlVal)
}

// TestLoggingSanitizesHeaders 验证敏感头部在日志中被遮蔽。
func TestLoggingSanitizesHeaders(t *testing.T) {
	log := mocks.NewMockLogger()
	mw := LoggingMiddleware(log)

	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return &fetcher.Response{URL: req.URL, StatusCode: 200}, nil
	})

	_, err := handler(context.Background(), &fetcher.Request{
		URL: "http://example.com",
		Headers: map[string]string{
			"Accept":             "text/html",
			"Cookie":             "session=abc; token=xyz",
			"Authorization":      "Bearer secret-token",
			"Proxy-Authorization": "Basic dXNlcjpwYXNz",
		},
	})
	require.NoError(t, err)

	entries := log.Entries()
	require.Len(t, entries, 1)
	headers, ok := mapField(entries[0].Fields, "headers")
	require.True(t, ok)
	assert.Equal(t, "text/html", headers["Accept"])
	assert.Equal(t, "[REDACTED]", headers["Cookie"])
	assert.Equal(t, "[REDACTED]", headers["Authorization"])
	assert.Equal(t, "[REDACTED]", headers["Proxy-Authorization"])
}

// TestLoggingRecordsStatusAndDuration 验证日志记录状态码和耗时。
func TestLoggingRecordsStatusAndDuration(t *testing.T) {
	log := mocks.NewMockLogger()
	mw := LoggingMiddleware(log)

	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return &fetcher.Response{URL: req.URL, StatusCode: 200}, nil
	})

	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.NoError(t, err)

	entries := log.Entries()
	require.Len(t, entries, 1)
	status, ok := intField(entries[0].Fields, "status_code")
	require.True(t, ok)
	assert.Equal(t, int64(200), status)
	duration, ok := intField(entries[0].Fields, "duration_ms")
	require.True(t, ok)
	assert.GreaterOrEqual(t, duration, int64(0))
}

// TestLoggingErrorLevelOnFailure 验证失败时使用 Error 级别记录。
func TestLoggingErrorLevelOnFailure(t *testing.T) {
	log := mocks.NewMockLogger()
	mw := LoggingMiddleware(log)

	expectedErr := errors.New("请求失败")
	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return nil, expectedErr
	})

	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err)

	entries := log.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, "error", entries[0].Level)
}

// Package middleware 的单元测试。
package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jciyuan-spider-v2/internal/fetcher"
)

// TestRateLimitEnforcesDelay 验证限流中间件保证相邻请求间隔不低于配置值。
func TestRateLimitEnforcesDelay(t *testing.T) {
	mw := RateLimitMiddleware(100) // 100ms

	var timestamps []time.Time
	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		timestamps = append(timestamps, time.Now())
		return &fetcher.Response{URL: req.URL, StatusCode: 200}, nil
	})

	ctx := context.Background()
	_, err := handler(ctx, &fetcher.Request{URL: "http://example.com/1"})
	require.NoError(t, err)
	_, err = handler(ctx, &fetcher.Request{URL: "http://example.com/2"})
	require.NoError(t, err)

	require.Len(t, timestamps, 2)
	elapsed := timestamps[1].Sub(timestamps[0])
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(90),
		"两次请求之间应至少等待 100ms（容差 10ms）")
}

// TestRateLimitContextCancel 验证上下文取消时，等待阶段会立即返回错误。
func TestRateLimitContextCancel(t *testing.T) {
	mw := RateLimitMiddleware(1000) // 1s

	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return &fetcher.Response{URL: req.URL, StatusCode: 200}, nil
	})

	// 先触发第一次请求，使第二次必须等待。
	ctx := context.Background()
	_, err := handler(ctx, &fetcher.Request{URL: "http://example.com/1"})
	require.NoError(t, err)

	cancelCtx, cancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err = handler(cancelCtx, &fetcher.Request{URL: "http://example.com/2"})
	assert.ErrorIs(t, err, context.Canceled)
}

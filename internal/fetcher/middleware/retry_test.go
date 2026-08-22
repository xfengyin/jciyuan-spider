package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	spidererrors "jciyuan-spider/internal/errors"
	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/mocks"
)

// TestRetrySuccessNoRetry 验证一次成功时不会触发重试计数。
func TestRetrySuccessNoRetry(t *testing.T) {
	m := mocks.NewMockMetrics()
	mw := RetryMiddleware(3, m)

	calls := 0
	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		calls++
		return &fetcher.Response{URL: req.URL, StatusCode: 200}, nil
	})

	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Equal(t, int64(0), m.RetryCount())
}

// TestRetryEventuallySucceeds 验证可重试错误会在达到最大次数前继续重试。
func TestRetryEventuallySucceeds(t *testing.T) {
	m := mocks.NewMockMetrics()
	mw := RetryMiddleware(3, m)

	calls := 0
	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		calls++
		if calls < 3 {
			return nil, spidererrors.ErrRetryable
		}
		return &fetcher.Response{URL: req.URL, StatusCode: 200}, nil
	})

	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	assert.Equal(t, int64(2), m.RetryCount())
}

// TestRetryBlockedErrorStops 验证 BlockedError 不会触发重试并原样返回。
func TestRetryBlockedErrorStops(t *testing.T) {
	m := mocks.NewMockMetrics()
	mw := RetryMiddleware(3, m)

	blocked := &spidererrors.BlockedError{URL: "http://example.com", StatusCode: 403}
	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return nil, blocked
	})

	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, blocked))
	assert.Equal(t, int64(0), m.RetryCount())
}

// TestRetryCaptchaErrorStops 验证 CaptchaError 不会触发重试。
func TestRetryCaptchaErrorStops(t *testing.T) {
	m := mocks.NewMockMetrics()
	mw := RetryMiddleware(3, m)

	captcha := &spidererrors.CaptchaError{URL: "http://example.com"}
	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return nil, captcha
	})

	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, captcha))
	assert.Equal(t, int64(0), m.RetryCount())
}

// TestRetryContextCanceledDuringBackoff 验证上下文取消时重试等待会立即退出。
func TestRetryContextCanceledDuringBackoff(t *testing.T) {
	m := mocks.NewMockMetrics()
	mw := RetryMiddleware(3, m)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return nil, spidererrors.ErrRetryable
	})

	_, err := handler(ctx, &fetcher.Request{URL: "http://example.com"})
	assert.ErrorIs(t, err, context.Canceled)
}

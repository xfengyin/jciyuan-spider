package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	spidererrors "jciyuan-spider-v2/internal/errors"
	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/mocks"
	"jciyuan-spider-v2/internal/model"
)

// TestCircuitBreakerOpensAfterFailures 验证连续失败达到阈值后熔断器打开。
func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	cfg := model.CircuitBreakerConfig{
		FailureThreshold: 2,
		OpenDuration:     500 * time.Millisecond,
		HalfOpenRequests: 1,
	}
	m := mocks.NewMockMetrics()
	mw := CircuitBreakerMiddleware(cfg, m)

	calls := 0
	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		calls++
		return nil, spidererrors.ErrRetryable
	})

	_, err1 := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	_, err2 := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err1)
	require.Error(t, err2)

	// 第三次应被熔断器直接拒绝。
	_, err3 := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err3)
	assert.Contains(t, err3.Error(), "熔断器已打开")
	assert.Equal(t, 2, calls)
}

// TestCircuitBreakerHalfOpenAndClose 验证熔断器进入半开后，成功请求会关闭熔断器。
func TestCircuitBreakerHalfOpenAndClose(t *testing.T) {
	cfg := model.CircuitBreakerConfig{
		FailureThreshold: 1,
		OpenDuration:     50 * time.Millisecond,
		HalfOpenRequests: 1,
	}
	m := mocks.NewMockMetrics()
	mw := CircuitBreakerMiddleware(cfg, m)

	fail := true
	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		if fail {
			return nil, spidererrors.ErrRetryable
		}
		return &fetcher.Response{URL: req.URL, StatusCode: 200}, nil
	})

	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err)

	time.Sleep(80 * time.Millisecond)
	fail = false
	_, err = handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.NoError(t, err)

	// 此时应已关闭，请求可继续成功。
	_, err = handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.NoError(t, err)
}

// TestCircuitBreakerErrorRateOpens 验证错误率达到阈值后熔断器打开。
func TestCircuitBreakerErrorRateOpens(t *testing.T) {
	cfg := model.CircuitBreakerConfig{
		FailureThreshold:   100, // 避免连续失败触发
		ErrorRateThreshold: 0.5,
		WindowSize:         4,
		OpenDuration:       500 * time.Millisecond,
	}
	m := mocks.NewMockMetrics()
	mw := CircuitBreakerMiddleware(cfg, m)

	calls := 0
	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		calls++
		return nil, spidererrors.ErrRetryable
	})

	for i := 0; i < 4; i++ {
		_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
		require.Error(t, err)
	}

	// 第 4 次调用后窗口已满并触发熔断，第 5 次请求应被直接拒绝。
	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "熔断器已打开")
	assert.Equal(t, 4, calls)
}

// TestCircuitBreakerSuccessResetsFailures 验证成功后连续失败计数会被重置。
func TestCircuitBreakerSuccessResetsFailures(t *testing.T) {
	cfg := model.CircuitBreakerConfig{
		FailureThreshold: 2,
		OpenDuration:     500 * time.Millisecond,
	}
	m := mocks.NewMockMetrics()
	mw := CircuitBreakerMiddleware(cfg, m)

	fail := true
	calls := 0
	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		calls++
		if fail {
			return nil, spidererrors.ErrRetryable
		}
		return &fetcher.Response{URL: req.URL, StatusCode: 200}, nil
	})

	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err)

	fail = false
	_, err = handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.NoError(t, err)

	fail = true
	// 重置后需要再连续失败 2 次才会在第 4 次调用后打开；第 5 次请求被直接拒绝。
	_, _ = handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	_, _ = handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	_, err = handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "熔断器已打开")
	assert.Equal(t, 4, calls)
}

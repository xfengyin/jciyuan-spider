package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	spidererrors "jciyuan-spider/internal/errors"
	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/model"
)

// TestProxyRotateOnRetryableError 验证遇到可重试错误时会自动切换代理。
func TestProxyRotateOnRetryableError(t *testing.T) {
	pm := NewProxyManager(model.ProxyConfig{
		Strategy: "round_robin",
		Proxies:  []string{"http://proxy1:8080", "http://proxy2:8080"},
	})
	mw := ProxyRotateMiddleware(pm)

	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return nil, spidererrors.ErrRetryable
	})

	before := pm.Current()
	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err)
	after := pm.Current()

	assert.Equal(t, "http://proxy1:8080", before)
	assert.Equal(t, "http://proxy2:8080", after)
}

// TestProxyRotateNoRotateOnSuccess 验证成功时不会切换代理。
func TestProxyRotateNoRotateOnSuccess(t *testing.T) {
	pm := NewProxyManager(model.ProxyConfig{
		Strategy: "round_robin",
		Proxies:  []string{"http://proxy1:8080", "http://proxy2:8080"},
	})
	mw := ProxyRotateMiddleware(pm)

	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return &fetcher.Response{URL: req.URL, StatusCode: 200}, nil
	})

	before := pm.Current()
	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.NoError(t, err)
	after := pm.Current()

	assert.Equal(t, before, after)
}

// TestProxyRotateNoRotateOnBlockedError 验证被拦截错误不会切换代理。
func TestProxyRotateNoRotateOnBlockedError(t *testing.T) {
	pm := NewProxyManager(model.ProxyConfig{
		Strategy: "round_robin",
		Proxies:  []string{"http://proxy1:8080", "http://proxy2:8080"},
	})
	mw := ProxyRotateMiddleware(pm)

	handler := mw(func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
		return nil, &spidererrors.BlockedError{URL: req.URL, StatusCode: 403}
	})

	before := pm.Current()
	_, err := handler(context.Background(), &fetcher.Request{URL: "http://example.com"})
	require.Error(t, err)
	after := pm.Current()

	assert.Equal(t, before, after)
}

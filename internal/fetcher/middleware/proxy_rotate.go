// Package middleware 提供 Fetcher 中间件实现。
package middleware

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"net/url"
	"sync"

	spidererrors "jciyuan-spider/internal/errors"
	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/model"
)

// ProxyManager 代理池管理器，支持 round_robin/random/least_used 策略。
type ProxyManager struct {
	proxies  []string
	strategy string
	mu       sync.Mutex
	index    int
	usage    map[string]int
}

// NewProxyManager 创建代理管理器。
func NewProxyManager(cfg model.ProxyConfig) *ProxyManager {
	pm := &ProxyManager{
		proxies:  filterEmpty(cfg.Proxies),
		strategy: cfg.Strategy,
		usage:    make(map[string]int),
	}
	if pm.strategy == "" {
		pm.strategy = "round_robin"
	}
	return pm
}

// Current 返回当前应使用的代理地址；无代理时返回空字符串。
func (pm *ProxyManager) Current() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.currentLocked()
}

// currentLocked 必须在持有锁时调用。
func (pm *ProxyManager) currentLocked() string {
	if len(pm.proxies) == 0 {
		return ""
	}
	switch pm.strategy {
	case "random":
		return pm.proxies[rand.Intn(len(pm.proxies))]
	case "least_used":
		return pm.leastUsed()
	default: // round_robin
		proxy := pm.proxies[pm.index%len(pm.proxies)]
		pm.usage[proxy]++
		return proxy
	}
}

// leastUsed 返回使用次数最少的代理。
func (pm *ProxyManager) leastUsed() string {
	var selected string
	minUsage := int(^uint(0) >> 1)
	for _, p := range pm.proxies {
		if pm.usage[p] < minUsage {
			minUsage = pm.usage[p]
			selected = p
		}
	}
	pm.usage[selected]++
	return selected
}

// Rotate 切换到下一个代理。
func (pm *ProxyManager) Rotate() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if len(pm.proxies) == 0 {
		return
	}
	pm.index = (pm.index + 1) % len(pm.proxies)
}

// ProxyURL 返回当前代理的 *url.URL，无代理时返回 nil。
// 签名兼容 http.Transport.Proxy。
func (pm *ProxyManager) ProxyURL(*http.Request) (*url.URL, error) {
	proxy := pm.Current()
	if proxy == "" {
		return nil, nil
	}
	return url.Parse(proxy)
}

// ProxyRotateMiddleware 代理轮换中间件，遇到可重试错误时自动切换代理。
func ProxyRotateMiddleware(pm *ProxyManager) fetcher.Middleware {
	return func(next fetcher.Handler) fetcher.Handler {
		return func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
			resp, err := next(ctx, req)
			if err != nil && len(pm.proxies) > 0 {
				if errors.Is(err, spidererrors.ErrRetryable) {
					pm.Rotate()
				}
			}
			return resp, err
		}
	}
}

// filterEmpty 过滤空代理地址。
func filterEmpty(proxies []string) []string {
	out := make([]string, 0, len(proxies))
	for _, p := range proxies {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

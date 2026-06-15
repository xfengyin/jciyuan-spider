package fetcher

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"jciyuan-spider-v2/internal/metrics"
	"jciyuan-spider-v2/internal/model"
)

// HTTPFetcher HTTP 请求器实现
type HTTPFetcher struct {
	client     *http.Client
	config     *model.Config
	userAgents []string
	metrics    *metrics.Collector
	middleware []Middleware
}

// NewHTTPFetcher 创建 HTTP 请求器
func NewHTTPFetcher(cfg *model.Config, m *metrics.Collector) (*HTTPFetcher, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("创建 cookie jar 失败: %w", err)
	}

	client := &http.Client{
		Timeout: time.Duration(cfg.Spider.Timeout) * time.Second,
		Jar:     jar,
	}

	f := &HTTPFetcher{
		client:     client,
		config:     cfg,
		userAgents: cfg.Anticrawler.UserAgents,
		metrics:    m,
	}

	// 注册默认中间件
	f.middleware = []Middleware{
		RateLimitMiddleware(cfg.Spider.Delay),
		RetryMiddleware(cfg.Spider.MaxRetry, m),
	}

	return f, nil
}

// Fetch 获取页面内容
func (f *HTTPFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	f.setHeaders(req)

	// 执行中间件链
	handler := f.executeRequest
	for i := len(f.middleware) - 1; i >= 0; i-- {
		handler = f.middleware[i](handler)
	}

	return handler(ctx, req)
}

// executeRequest 执行实际 HTTP 请求
func (f *HTTPFetcher) executeRequest(ctx context.Context, req *http.Request) ([]byte, error) {
	resp, err := f.client.Do(req)
	if err != nil {
		f.metrics.IncrFail()
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	f.metrics.AddBytes(resp.ContentLength)

	if resp.StatusCode == http.StatusForbidden {
		f.metrics.IncrFail()
		return nil, &BlockedError{URL: req.URL.String(), StatusCode: 403, Message: "访问被禁止"}
	}

	if resp.StatusCode != http.StatusOK {
		f.metrics.IncrFail()
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, req.URL.String())
	}

	reader := resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip 解压失败: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	body, err := io.ReadAll(io.LimitReader(reader, 50*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if isCaptchaPage(body) {
		return nil, &BlockedError{URL: req.URL.String(), StatusCode: 403, Message: "验证码页面"}
	}

	f.metrics.IncrSuccess()
	return body, nil
}

// setHeaders 设置请求头
func (f *HTTPFetcher) setHeaders(req *http.Request) {
	if f.config.Anticrawler.RandomUA && len(f.userAgents) > 0 {
		ua := f.userAgents[rand.Intn(len(f.userAgents))]
		req.Header.Set("User-Agent", ua)
	} else if len(f.userAgents) > 0 {
		req.Header.Set("User-Agent", f.userAgents[0])
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Referer", f.config.Spider.BaseURL+"/")
	req.Header.Set("Origin", f.config.Spider.BaseURL)
}

// BlockedError 访问被阻止错误
type BlockedError struct {
	URL        string
	StatusCode int
	Message    string
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("Blocked: %s (HTTP %d) - %s", e.URL, e.StatusCode, e.Message)
}

// IsBlocked 判断是否为阻止错误
func IsBlocked(err error) bool {
	_, ok := err.(*BlockedError)
	return ok
}

// isCaptchaPage 检测验证码页面
func isCaptchaPage(body []byte) bool {
	content := strings.ToLower(string(body))
	keywords := []string{"验证码", " captcha", "安全验证", "请输入验证码"}
	for _, kw := range keywords {
		if strings.Contains(content, kw) {
			return true
		}
	}
	return false
}

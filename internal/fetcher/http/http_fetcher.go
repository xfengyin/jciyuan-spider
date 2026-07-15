// Package http 提供基于 net/http 的 Fetcher 实现，支持连接池、自动解压、
// 请求头管理、URL 白名单、响应体大小限制与中间件链。
package http

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	spidererrors "jciyuan-spider-v2/internal/errors"
	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/fetcher/middleware"
	"jciyuan-spider-v2/internal/logger"
	"jciyuan-spider-v2/internal/metrics"
	"jciyuan-spider-v2/internal/model"
)

// HTTPFetcher 实现 fetcher.Fetcher 接口的企业级 HTTP 请求器。
type HTTPFetcher struct {
	client     *http.Client
	cfg        model.FetcherConfig
	anti       model.AnticrawlerConfig
	spider     model.SpiderConfig
	userAgents []string
	uaIndex    int
	metrics    metrics.Metrics
	logger     logger.Logger
	chain      fetcher.Middleware
	proxyMgr   *middleware.ProxyManager
	robots     *RobotsChecker
}

// NewHTTPFetcher 创建 HTTPFetcher 实例，按配置组装中间件链并注册到 SPI。
func NewHTTPFetcher(
	cfg model.FetcherConfig,
	anti model.AnticrawlerConfig,
	spider model.SpiderConfig,
	mws []model.MiddlewareItem,
	m metrics.Metrics,
	l logger.Logger,
) (fetcher.Fetcher, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("创建 cookie jar 失败: %w", err)
	}

	proxyMgr := middleware.NewProxyManager(cfg.Proxy)
	transport := buildTransport(cfg.HTTP.Transport, proxyMgr)
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.HTTP.Timeout) * time.Second,
		Jar:       jar,
	}
	f := &HTTPFetcher{
		client:     client,
		cfg:        cfg,
		anti:       anti,
		spider:     spider,
		userAgents: anti.UserAgents,
		metrics:    m,
		logger:     l,
		proxyMgr:   proxyMgr,
	}

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !cfg.HTTP.FollowRedirects {
			return http.ErrUseLastResponse
		}
		// 拒绝外部域名重定向，防止跟随到白名单之外的站点。
		if !f.isAllowedHost(req.URL.Host) {
			return spidererrors.New(spidererrors.CategoryValidation, "重定向目标域名不在白名单内")
		}
		return nil
	}

	chain, err := f.buildMiddlewareChain(mws)
	if err != nil {
		return nil, fmt.Errorf("构建中间件链失败: %w", err)
	}
	f.chain = chain

	// 若开启 robots.txt 合规检查，启动时同步拉取并解析。
	if anti.RobotsTxtCheck && spider.BaseURL != "" {
		checker, err := newRobotsChecker(client, spider.BaseURL, l)
		if err != nil {
			l.Warn("robots.txt 加载失败，将跳过合规检查", logger.Err(err))
		} else {
			f.robots = checker
		}
	}

	return f, nil
}

func init() {
	// SPI 注册 HTTP Fetcher
	fetcher.Register("http", NewHTTPFetcher)
}

// buildTransport 根据配置构建 http.Transport，并接入代理管理器。
func buildTransport(cfg model.HTTPTransportConfig, pm *middleware.ProxyManager) *http.Transport {
	return &http.Transport{
		Proxy:               pm.ProxyURL,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		DisableKeepAlives:   cfg.DisableKeepAlives,
		TLSHandshakeTimeout: cfg.TLSHandshakeTimeout,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
}

// buildMiddlewareChain 按配置顺序组装中间件链。
func (f *HTTPFetcher) buildMiddlewareChain(items []model.MiddlewareItem) (fetcher.Middleware, error) {
	mws := make([]fetcher.Middleware, 0, len(items))
	for _, item := range items {
		mw, err := f.resolveMiddleware(item)
		if err != nil {
			return nil, err
		}
		if mw != nil {
			mws = append(mws, mw)
		}
	}
	return fetcher.Compose(mws...), nil
}

// resolveMiddleware 根据中间件配置项返回对应的 Middleware。
func (f *HTTPFetcher) resolveMiddleware(item model.MiddlewareItem) (fetcher.Middleware, error) {
	switch item.Name {
	case "trace":
		return middleware.TraceMiddleware(), nil
	case "metrics":
		return middleware.MetricsMiddleware(f.metrics), nil
	case "logging":
		return middleware.LoggingMiddleware(f.logger), nil
	case "rate_limit":
		return middleware.RateLimitMiddleware(f.spider.Delay), nil
	case "retry":
		maxRetry := f.cfg.HTTP.MaxRetry
		if maxRetry <= 0 {
			maxRetry = f.spider.MaxRetry
		}
		return middleware.RetryMiddleware(maxRetry, f.metrics), nil
	case "circuit_breaker":
		return middleware.CircuitBreakerMiddleware(f.cfg.CircuitBreaker, f.metrics), nil
	case "proxy_rotate":
		return middleware.ProxyRotateMiddleware(f.proxyMgr), nil
	default:
		return nil, fmt.Errorf("未知中间件: %s", item.Name)
	}
}

// Fetch 执行 HTTP 请求，先校验 URL 白名单与 robots.txt，再进入中间件链。
func (f *HTTPFetcher) Fetch(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
	if req == nil {
		return nil, spidererrors.New(spidererrors.CategoryValidation, "请求对象不能为空")
	}
	if err := f.validateURL(req.URL); err != nil {
		return nil, err
	}
	if f.anti.RobotsTxtCheck && f.robots != nil && !f.robots.IsAllowed(req.URL) {
		return nil, spidererrors.New(spidererrors.CategoryBlocked, "robots.txt 禁止抓取该路径")
	}
	if req.Method == "" {
		req.Method = http.MethodGet
	}

	handler := f.executeRequest
	if f.chain != nil {
		handler = f.chain(handler)
	}
	return handler(ctx, req)
}

// validateURL 校验请求 URL 是否在 SpiderConfig.BaseURL 对应域名白名单内。
func (f *HTTPFetcher) validateURL(rawURL string) error {
	baseHost, err := extractHost(f.spider.BaseURL)
	if err != nil || baseHost == "" {
		return nil // 未配置 baseURL 时不限制
	}
	targetHost, err := extractHost(rawURL)
	if err != nil {
		return spidererrors.Wrap(err, spidererrors.CategoryValidation, "URL 解析失败")
	}
	if targetHost != baseHost {
		return spidererrors.Wrap(
			fmt.Errorf("目标域名 %s 不在白名单 %s 内", targetHost, baseHost),
			spidererrors.CategoryValidation,
			"URL 白名单校验失败",
		)
	}
	return nil
}

// extractHost 从 URL 字符串中提取主机名（含端口）。
func extractHost(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return u.Host, nil
}

// isAllowedHost 判断目标主机是否在白名单内；未配置 BaseURL 时不限制。
func (f *HTTPFetcher) isAllowedHost(host string) bool {
	baseHost, err := extractHost(f.spider.BaseURL)
	if err != nil || baseHost == "" {
		return true
	}
	return host == baseHost
}

// executeRequest 执行实际 HTTP 请求，处理请求体、响应解压与大小限制。
func (f *HTTPFetcher) executeRequest(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
	body, err := f.requestBody(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	f.setHeaders(httpReq, req)
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := f.client.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %v", spidererrors.ErrRetryable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return nil, &spidererrors.BlockedError{
			URL:        req.URL,
			StatusCode: resp.StatusCode,
			Message:    "访问被禁止",
		}
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: HTTP 429", spidererrors.ErrRetryable)
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, fmt.Errorf("%w: HTTP %d", spidererrors.ErrRetryable, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, spidererrors.Wrap(
			fmt.Errorf("HTTP %d", resp.StatusCode),
			spidererrors.CategoryHTTP,
			"非预期状态码",
		)
	}

	reader, err := f.decompress(resp)
	if err != nil {
		return nil, fmt.Errorf("%w: 响应解压失败: %v", spidererrors.ErrRetryable, err)
	}

	maxBody := f.cfg.HTTP.MaxBodySize
	if maxBody <= 0 {
		maxBody = 50 * 1024 * 1024 // 默认 50MB
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(reader, maxBody+1))
	if err != nil {
		return nil, fmt.Errorf("%w: 读取响应失败: %v", spidererrors.ErrRetryable, err)
	}
	if int64(len(bodyBytes)) > maxBody {
		return nil, spidererrors.New(spidererrors.CategoryValidation, "响应体超过大小限制")
	}

	if isCaptchaPage(bodyBytes) {
		return nil, &spidererrors.CaptchaError{
			URL:     req.URL,
			Message: "验证码页面",
		}
	}

	return &fetcher.Response{
		URL:        req.URL,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       bodyBytes,
		Meta:       copyMeta(req.Meta),
	}, nil
}

// requestBody 返回请求体 reader；GET 默认无 Body。
func (f *HTTPFetcher) requestBody(req *fetcher.Request) (io.Reader, error) {
	if req.Method == http.MethodGet || req.Method == http.MethodHead {
		return nil, nil
	}
	if len(req.Body) == 0 {
		return nil, nil
	}
	return io.NopCloser(bytes.NewReader(req.Body)), nil
}

// setHeaders 设置默认请求头，可被 req.Headers 覆盖。
func (f *HTTPFetcher) setHeaders(httpReq *http.Request, req *fetcher.Request) {
	if len(f.userAgents) > 0 {
		ua := f.selectUA()
		httpReq.Header.Set("User-Agent", ua)
	}

	httpReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	httpReq.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	httpReq.Header.Set("Accept-Encoding", "gzip, deflate")
	httpReq.Header.Set("Connection", "keep-alive")
	httpReq.Header.Set("Cache-Control", "no-cache")

	if f.anti.RefererPolicy != "" && f.anti.RefererPolicy != "origin" {
		httpReq.Header.Set("Referer", f.anti.RefererPolicy)
	} else if f.spider.BaseURL != "" {
		httpReq.Header.Set("Referer", f.spider.BaseURL+"/")
	}

	if req.Method != http.MethodGet && req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
}

// selectUA 按配置随机或顺序选择 User-Agent。
func (f *HTTPFetcher) selectUA() string {
	if f.anti.RandomUA {
		return f.userAgents[rand.Intn(len(f.userAgents))]
	}
	ua := f.userAgents[f.uaIndex%len(f.userAgents)]
	f.uaIndex++
	return ua
}

// decompress 根据 Content-Encoding 返回解压后的 reader。
func (f *HTTPFetcher) decompress(resp *http.Response) (io.ReadCloser, error) {
	encoding := strings.ToLower(resp.Header.Get("Content-Encoding"))
	switch encoding {
	case "gzip":
		return gzip.NewReader(resp.Body)
	case "deflate":
		return flate.NewReader(resp.Body), nil
	default:
		return resp.Body, nil
	}
}

// copyMeta 复制请求 Meta 到响应，避免下游修改影响上游。
func copyMeta(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// Close 释放 HTTP 连接池资源。
func (f *HTTPFetcher) Close() error {
	f.client.CloseIdleConnections()
	return nil
}

// isCaptchaPage 检测页面内容是否包含验证码特征。
func isCaptchaPage(body []byte) bool {
	content := strings.ToLower(string(body))
	keywords := []string{"验证码", "captcha", "安全验证", "请输入验证码"}
	for _, kw := range keywords {
		if strings.Contains(content, kw) {
			return true
		}
	}
	return false
}

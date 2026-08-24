package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultHTTPUserAgent 默认 User-Agent。
const DefaultHTTPUserAgent = "jciyuan-spider-crawler/0.1 (+https://github.com/xfengyin/jciyuan-spider)"

const (
	defaultHTTPTimeout    = 30 * time.Second
	defaultHTTPMaxBody    = int64(8 << 20) // 8MB
	defaultHTTPMaxBodyStr = "8MB"
)

// HTTPOptions HTTPFetcher 的配置选项。
type HTTPOptions struct {
	// Timeout 单次请求总超时，默认 30s。
	Timeout time.Duration
	// UserAgent 请求 UA，默认 DefaultHTTPUserAgent。
	UserAgent string
	// Headers 附加请求头（会覆盖 UA 之外的默认值）。
	Headers map[string]string
	// MaxBodySize 响应体大小上限（字节），默认 8MB；超过时报错。
	MaxBodySize int64
	// FollowRedirects 是否跟随重定向，默认 true。
	FollowRedirects bool
}

// DefaultHTTPOptions 返回默认选项。
func DefaultHTTPOptions() HTTPOptions {
	return HTTPOptions{
		Timeout:         defaultHTTPTimeout,
		UserAgent:       DefaultHTTPUserAgent,
		MaxBodySize:     defaultHTTPMaxBody,
		FollowRedirects: true,
	}
}

// StatusError 非 2xx/3xx 响应状态错误。
type StatusError struct {
	Code   int
	Status string
	URL    string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("crawler: HTTP %d %s (url=%s)", e.Code, e.Status, e.URL)
}

// HTTPFetcher 基于 net/http 的默认抓取实现，只实现 Fetch 阶段。
// 仅依赖标准库；如需代理池、中间件链、反爬等企业级能力，可换用 internal/fetcher。
type HTTPFetcher struct {
	client *http.Client
	opts   HTTPOptions
}

// NewHTTPFetcher 创建默认 HTTP 抓取器；不传选项时使用默认值。
func NewHTTPFetcher(opts ...HTTPOptions) *HTTPFetcher {
	o := DefaultHTTPOptions()
	if len(opts) > 0 {
		if opts[0].Timeout > 0 {
			o.Timeout = opts[0].Timeout
		}
		if opts[0].UserAgent != "" {
			o.UserAgent = opts[0].UserAgent
		}
		if opts[0].Headers != nil {
			o.Headers = opts[0].Headers
		}
		if opts[0].MaxBodySize > 0 {
			o.MaxBodySize = opts[0].MaxBodySize
		}
		o.FollowRedirects = opts[0].FollowRedirects
	}
	client := &http.Client{Timeout: o.Timeout}
	if !o.FollowRedirects {
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return &HTTPFetcher{client: client, opts: o}
}

// Fetch 抓取 url 并返回 Page。非 2xx/3xx 状态返回 *StatusError。
func (f *HTTPFetcher) Fetch(ctx context.Context, url string) (*Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("crawler: 构造请求失败: %w", err)
	}
	req.Header.Set("User-Agent", f.opts.UserAgent)
	req.Header.Set("Accept", "*/*")
	// 注意：不手动设置 Accept-Encoding，交由 net/http 传输层自动请求 gzip
	// 并透明解压，避免收到原始 gzip 字节。
	for k, v := range f.opts.Headers {
		req.Header.Set(k, v)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crawler: 抓取 %s 失败: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, &StatusError{Code: resp.StatusCode, Status: resp.Status, URL: url}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, f.opts.MaxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("crawler: 读取响应失败: %w", err)
	}
	if int64(len(body)) > f.opts.MaxBodySize {
		return nil, fmt.Errorf("crawler: 响应体超过上限 %s (url=%s)", defaultHTTPMaxBodyStr, url)
	}

	return &Page{
		URL:        url,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
		Text:       string(body),
	}, nil
}

// HTTPCrawler 默认 HTTP 爬虫：Fetch 用 HTTPFetcher，Parse 透传原文，
// Extract 输出 {url, status, text} 条目，适合快速开始与简单抓取场景。
type HTTPCrawler struct {
	fetcher *HTTPFetcher
	name    string
}

// NewHTTPCrawler 创建默认 HTTP 爬虫（完整实现 Crawler 接口）。
func NewHTTPCrawler(opts ...HTTPOptions) *HTTPCrawler {
	return &HTTPCrawler{fetcher: NewHTTPFetcher(opts...), name: "http"}
}

// Name 返回爬虫名称。
func (c *HTTPCrawler) Name() string { return c.name }

// Fetch 抓取页面（委托 HTTPFetcher）。
func (c *HTTPCrawler) Fetch(ctx context.Context, url string) (*Page, error) {
	return c.fetcher.Fetch(ctx, url)
}

// Parse 直接返回页面文本作为中间表示。
func (c *HTTPCrawler) Parse(_ context.Context, page *Page) (any, error) {
	return page.Text, nil
}

// Extract 将文本封装为 {url, status, text} 条目。
func (c *HTTPCrawler) Extract(_ context.Context, parsed any) ([]Item, error) {
	text, ok := parsed.(string)
	if !ok {
		return nil, fmt.Errorf("crawler: 解析结果类型异常: %T", parsed)
	}
	return []Item{{"text": text}}, nil
}

// sharedFetcher 包级共享的默认抓取器，供 Fetch 便捷函数使用（http.Client 可并发复用）。
var sharedFetcher = NewHTTPFetcher()

// Fetch 一行式便捷函数：用默认 HTTP 抓取器抓取 url 并返回 Page。
//
//	page, err := crawler.Fetch(ctx, "https://example.com")
func Fetch(ctx context.Context, url string) (*Page, error) {
	return sharedFetcher.Fetch(ctx, url)
}

// IsStatusError 判断错误是否为 HTTP 状态错误。
func IsStatusError(err error) bool {
	_, ok := err.(*StatusError)
	return ok
}

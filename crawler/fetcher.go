// Package crawler - 爬虫核心模块
package crawler
import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"
	"jciyuan-spider-v2/model"
	
)
// ============================================================================
// 请求器
// ============================================================================
// Fetcher HTTP请求器
type Fetcher struct {
	client      *http.Client
	config      *model.Config
	cookieJar   *cookiejar.Jar
	userAgents  []string
	proxyIndex  int
	proxyMu     sync.Mutex
	requestMu   sync.Mutex
	lastRequest time.Time
	stats       *StatsCollector
}
// NewFetcher 创建请求器
func NewFetcher(config *model.Config, stats *StatsCollector) *Fetcher {
	jar, _ := cookiejar.New(nil)
	
	client := &http.Client{
		Timeout: time.Duration(config.Spider.Timeout) * time.Second,
		Jar:     jar,
	}
	
	return &Fetcher{
		client:     client,
		config:     config,
		cookieJar:  jar,
		userAgents: config.Anticrawler.UserAgents,
		stats:      stats,
	}
}
// Fetch 获取页面内容
func (f *Fetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	// 限流控制
	f.rateLimit()
	
	// 构建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	
	// 设置请求头
	f.setHeaders(req)
	
	// 重试机制
	var body []byte
	var lastErr error
	
	for attempt := 0; attempt <= f.config.Spider.MaxRetry; attempt++ {
		if attempt > 0 {
			f.stats.IncrRetry()
			time.Sleep(time.Duration(attempt*500) * time.Millisecond)
		}
		
		body, lastErr = f.doRequest(req, url)
		if lastErr == nil {
			break
		}
		
		// 如果是403或验证码，不重试
		if isBlockedError(lastErr) {
			break
		}
	}
	
	if lastErr != nil {
		f.stats.IncrFail()
		return nil, lastErr
	}
	
	f.stats.IncrSuccess()
	return body, nil
}
// doRequest 执行请求
func (f *Fetcher) doRequest(req *http.Request, url string) ([]byte, error) {
	start := time.Now()
	
	// 代理
	if f.config.Anticrawler.EnableProxy && len(f.config.Anticrawler.Proxies) > 0 {
		proxy := f.getProxy()
		if proxy != "" {
			f.client.Transport = &http.Transport{
				Proxy: http.ProxyURL(nil),
			}
			_ = proxy // 使用代理需要设置
		}
	}
	
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	duration := time.Since(start).Milliseconds()
	f.stats.RecordRequest(duration, resp.ContentLength)
	
	// 状态码检查
	if resp.StatusCode == http.StatusForbidden {
		return nil, &BlockedError{URL: url, StatusCode: 403, Message: "访问被禁止"}
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	
	// 解压
	reader := resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip解压失败: %w", err)
		}
		defer gz.Close()
		reader = gz
	}
	
	body, err := io.ReadAll(io.LimitReader(reader, 50*1024*1024)) // 限制50MB
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	
	// 检测验证码页面
	if isCaptchaPage(body) {
		return nil, &BlockedError{URL: url, StatusCode: 403, Message: "验证码页面"}
	}
	
	return body, nil
}
// setHeaders 设置请求头
func (f *Fetcher) setHeaders(req *http.Request) {
	// Random UA
	if f.config.Anticrawler.RandomUA && len(f.userAgents) > 0 {
		ua := f.userAgents[rand.Intn(len(f.userAgents))]
		req.Header.Set("User-Agent", ua)
	} else if len(f.userAgents) > 0 {
		req.Header.Set("User-Agent", f.userAgents[0])
	}
	
	// 其他请求头
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	
	// Referer
	req.Header.Set("Referer", f.config.Spider.BaseURL+"/")
	
	// Origin
	req.Header.Set("Origin", f.config.Spider.BaseURL)
}
// rateLimit 限流控制
func (f *Fetcher) rateLimit() {
	f.requestMu.Lock()
	defer f.requestMu.Unlock()
	
	delay := time.Duration(f.config.Spider.Delay) * time.Millisecond
	elapsed := time.Since(f.lastRequest)
	
	if elapsed < delay {
		time.Sleep(delay - elapsed)
	}
	
	f.lastRequest = time.Now()
}
// getProxy 获取代理
func (f *Fetcher) getProxy() string {
	f.proxyMu.Lock()
	defer f.proxyMu.Unlock()
	
	if len(f.config.Anticrawler.Proxies) == 0 {
		return ""
	}
	
	proxy := f.config.Anticrawler.Proxies[f.proxyIndex]
	f.proxyIndex = (f.proxyIndex + 1) % len(f.config.Anticrawler.Proxies)
	
	return proxy
}
// ============================================================================
// 错误类型
// ============================================================================
// BlockedError 访问被阻止错误
type BlockedError struct {
	URL        string
	StatusCode int
	Message    string
}
func (e *BlockedError) Error() string {
	return fmt.Sprintf("Blocked: %s (HTTP %d) - %s", e.URL, e.StatusCode, e.Message)
}
func isBlockedError(err error) bool {
	_, ok := err.(*BlockedError)
	return ok
}
func isCaptchaPage(body []byte) bool {
	content := strings.ToLower(string(body))
	captchaKeywords := []string{"验证码", " captcha", "安全验证", "请输入验证码", " captcha"}
	
	for _, keyword := range captchaKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	
	return false
}
// ============================================================================
// 统计收集器
// ============================================================================
// StatsCollector 统计收集器
type StatsCollector struct {
	mu           sync.RWMutex
	totalReq     int64
	successCount int64
	failCount    int64
	retryCount   int64
	parseCount   int64
	parseFail    int64
	totalBytes   int64
	lastReset    time.Time
}
// NewStatsCollector 创建统计收集器
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		lastReset: time.Now(),
	}
}
// IncrSuccess 成功计数
func (s *StatsCollector) IncrSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.successCount++
	s.totalReq++
}
// IncrFail 失败计数
func (s *StatsCollector) IncrFail() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failCount++
	s.totalReq++
}
// IncrRetry 重试计数
func (s *StatsCollector) IncrRetry() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retryCount++
}
// IncrParse 解析成功计数
func (s *StatsCollector) IncrParse() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parseCount++
}
// IncrParseFail 解析失败计数
func (s *StatsCollector) IncrParseFail() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parseFail++
}
// RecordRequest 记录请求
func (s *StatsCollector) RecordRequest(durationMs int64, bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalBytes += bytes
}
// GetStats 获取统计
func (s *StatsCollector) GetStats() model.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return model.Stats{
		TotalRequests:  s.totalReq,
		SuccessCount:   s.successCount,
		FailCount:      s.failCount,
		RetryCount:     s.retryCount,
		ParseCount:     s.parseCount,
		ParseFailCount: s.parseFail,
		Bandwidth:      s.totalBytes,
	}
}
// Reset 重置统计
func (s *StatsCollector) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.totalReq = 0
	s.successCount = 0
	s.failCount = 0
	s.retryCount = 0
	s.parseCount = 0
	s.parseFail = 0
	s.totalBytes = 0
	s.lastReset = time.Now()
}

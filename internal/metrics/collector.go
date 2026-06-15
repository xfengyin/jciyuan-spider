// Package metrics - 指标收集模块
package metrics

import (
	"sync"
	"time"

	"jciyuan-spider-v2/internal/model"
)

// Collector 指标收集器
type Collector struct {
	mu           sync.RWMutex
	totalReq     int64
	successCount int64
	failCount    int64
	retryCount   int64
	parseCount   int64
	parseFail    int64
	totalBytes   int64
	startTime    time.Time
}

// NewCollector 创建指标收集器
func NewCollector() *Collector {
	return &Collector{
		startTime: time.Now(),
	}
}

// IncrSuccess 成功计数
func (c *Collector) IncrSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.successCount++
	c.totalReq++
}

// IncrFail 失败计数
func (c *Collector) IncrFail() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failCount++
	c.totalReq++
}

// IncrRetry 重试计数
func (c *Collector) IncrRetry() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retryCount++
}

// IncrParse 解析成功计数
func (c *Collector) IncrParse() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.parseCount++
}

// IncrParseFail 解析失败计数
func (c *Collector) IncrParseFail() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.parseFail++
}

// AddBytes 记录流量
func (c *Collector) AddBytes(n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalBytes += n
}

// GetStats 获取统计快照
func (c *Collector) GetStats() model.Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return model.Stats{
		StartTime:      c.startTime,
		TotalRequests:  c.totalReq,
		SuccessCount:   c.successCount,
		FailCount:      c.failCount,
		RetryCount:     c.retryCount,
		ParseCount:     c.parseCount,
		ParseFailCount: c.parseFail,
		Bandwidth:      c.totalBytes,
	}
}

// Reset 重置统计
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalReq = 0
	c.successCount = 0
	c.failCount = 0
	c.retryCount = 0
	c.parseCount = 0
	c.parseFail = 0
	c.totalBytes = 0
	c.startTime = time.Now()
}

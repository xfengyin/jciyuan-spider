package mocks

import (
	"context"
	"sync"
	"time"

	"jciyuan-spider/internal/metrics"
	"jciyuan-spider/internal/model"
)

// MockMetrics 实现 metrics.Metrics 接口，用于测试中断言计数变化。
type MockMetrics struct {
	mu        sync.RWMutex
	success   int64
	fail      int64
	retry     int64
	parse     int64
	parseFail int64
	storage   int64
	storageFail int64
	bytes     int64
	queueSize int
	cbState   int
}

// compile-time 接口校验。
var _ metrics.Metrics = (*MockMetrics)(nil)

// NewMockMetrics 创建 MockMetrics 实例。
func NewMockMetrics() *MockMetrics {
	return &MockMetrics{}
}

// IncrSuccess 增加成功计数。
func (m *MockMetrics) IncrSuccess(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.success++
}

// IncrFail 增加失败计数。
func (m *MockMetrics) IncrFail(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fail++
}

// IncrRetry 增加重试计数。
func (m *MockMetrics) IncrRetry(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retry++
}

// IncrParse 增加解析成功计数。
func (m *MockMetrics) IncrParse(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parse++
}

// IncrParseFail 增加解析失败计数。
func (m *MockMetrics) IncrParseFail(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parseFail++
}

// IncrStorageSave 增加存储成功计数。
func (m *MockMetrics) IncrStorageSave(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storage++
}

// IncrStorageSaveFail 增加存储失败计数。
func (m *MockMetrics) IncrStorageSaveFail(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storageFail++
}

// AddBytes 累加流量字节数。
func (m *MockMetrics) AddBytes(ctx context.Context, n int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bytes += n
}

// RecordRequestDuration 记录请求耗时，Mock 实现无操作。
func (m *MockMetrics) RecordRequestDuration(ctx context.Context, d time.Duration) {}

// RecordRequestDurationWithStatus 按状态记录请求耗时，Mock 实现无操作。
func (m *MockMetrics) RecordRequestDurationWithStatus(ctx context.Context, status string, d time.Duration) {
}

// SetQueueSize 设置队列长度。
func (m *MockMetrics) SetQueueSize(ctx context.Context, n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queueSize = n
}

// SetCircuitBreakerState 设置熔断器状态。
func (m *MockMetrics) SetCircuitBreakerState(ctx context.Context, state int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cbState = state
}

// GetStats 获取统计快照。
func (m *MockMetrics) GetStats() model.Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return model.Stats{
		SuccessCount:     m.success,
		FailCount:        m.fail,
		RetryCount:       m.retry,
		ParseCount:       m.parse,
		ParseFailCount:   m.parseFail,
		StorageSaveCount: m.storage,
		StorageSaveFail:  m.storageFail,
		Bandwidth:        m.bytes,
	}
}

// RetryCount 返回重试次数，便于测试断言。
func (m *MockMetrics) RetryCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.retry
}

// FailCount 返回失败次数。
func (m *MockMetrics) FailCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fail
}

// ParseCount 返回解析成功次数。
func (m *MockMetrics) ParseCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.parse
}

// StorageSaveCount 返回存储成功次数。
func (m *MockMetrics) StorageSaveCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storage
}

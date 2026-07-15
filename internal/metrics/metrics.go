// Package metrics 提供可观测指标抽象，支持 memory 与 prometheus 两种后端。
package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"jciyuan-spider-v2/internal/model"
)

// Metrics 是 v3 指标接口，各模块通过该接口上报监控数据。
type Metrics interface {
	// IncrSuccess 请求成功计数
	IncrSuccess(ctx context.Context)
	// IncrFail 请求失败计数
	IncrFail(ctx context.Context)
	// IncrRetry 重试计数
	IncrRetry(ctx context.Context)
	// IncrParse 解析成功计数
	IncrParse(ctx context.Context)
	// IncrParseFail 解析失败计数
	IncrParseFail(ctx context.Context)
	// IncrStorageSave 存储保存成功计数
	IncrStorageSave(ctx context.Context)
	// IncrStorageSaveFail 存储保存失败计数
	IncrStorageSaveFail(ctx context.Context)
	// AddBytes 记录流量字节数
	AddBytes(ctx context.Context, n int64)
	// RecordRequestDuration 记录请求耗时（默认按 success 标签记录，兼容旧调用）
	RecordRequestDuration(ctx context.Context, d time.Duration)
	// RecordRequestDurationWithStatus 按状态标签记录请求耗时
	RecordRequestDurationWithStatus(ctx context.Context, status string, d time.Duration)
	// SetQueueSize 设置 Worker 队列长度
	SetQueueSize(ctx context.Context, n int)
	// SetCircuitBreakerState 设置熔断器状态（0=closed, 1=open, 2=half-open）
	SetCircuitBreakerState(ctx context.Context, state int)
	// GetStats 获取内存统计快照
	GetStats() model.Stats
}

// New 根据配置创建 Metrics 实现
func New(cfg model.MetricsConfig) (Metrics, error) {
	if !cfg.Enabled {
		return NewMemoryMetrics(), nil
	}
	switch cfg.Backend {
	case "prometheus":
		return NewPrometheusMetrics(cfg.Prometheus)
	case "memory", "":
		return NewMemoryMetrics(), nil
	default:
		return nil, fmt.Errorf("unknown metrics backend: %s", cfg.Backend)
	}
}

// memoryMetrics 内存指标实现，兼容 v2 Collector 行为
type memoryMetrics struct {
	mu           sync.RWMutex
	totalReq     int64
	successCount int64
	failCount    int64
	retryCount   int64
	parseCount   int64
	parseFail    int64
	storageSave  int64
	storageFail  int64
	totalBytes   int64
	queueSize    int
	cbState      int
	startTime    time.Time
}

// NewMemoryMetrics 创建内存指标实现
func NewMemoryMetrics() Metrics {
	return &memoryMetrics{startTime: time.Now()}
}

// IncrSuccess 请求成功计数
func (m *memoryMetrics) IncrSuccess(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.successCount++
	m.totalReq++
}

// IncrFail 请求失败计数
func (m *memoryMetrics) IncrFail(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failCount++
	m.totalReq++
}

// IncrRetry 重试计数
func (m *memoryMetrics) IncrRetry(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retryCount++
}

// IncrParse 解析成功计数
func (m *memoryMetrics) IncrParse(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parseCount++
}

// IncrParseFail 解析失败计数
func (m *memoryMetrics) IncrParseFail(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parseFail++
}

// IncrStorageSave 存储保存成功计数
func (m *memoryMetrics) IncrStorageSave(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storageSave++
}

// IncrStorageSaveFail 存储保存失败计数
func (m *memoryMetrics) IncrStorageSaveFail(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storageFail++
}

// AddBytes 记录流量字节数
func (m *memoryMetrics) AddBytes(ctx context.Context, n int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalBytes += n
}

// RecordRequestDuration 记录请求耗时（内存实现仅保留，Prometheus 实现会记录 histogram）
func (m *memoryMetrics) RecordRequestDuration(ctx context.Context, d time.Duration) {
}

// RecordRequestDurationWithStatus 按状态标签记录请求耗时（内存实现仅保留）
func (m *memoryMetrics) RecordRequestDurationWithStatus(ctx context.Context, status string, d time.Duration) {
}

// SetQueueSize 设置 Worker 队列长度
func (m *memoryMetrics) SetQueueSize(ctx context.Context, n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queueSize = n
}

// SetCircuitBreakerState 设置熔断器状态
func (m *memoryMetrics) SetCircuitBreakerState(ctx context.Context, state int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cbState = state
}

// GetStats 获取内存统计快照
func (m *memoryMetrics) GetStats() model.Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return model.Stats{
		StartTime:        m.startTime,
		TotalRequests:    m.totalReq,
		SuccessCount:     m.successCount,
		FailCount:        m.failCount,
		RetryCount:       m.retryCount,
		ParseCount:       m.parseCount,
		ParseFailCount:   m.parseFail,
		StorageSaveCount: m.storageSave,
		StorageSaveFail:  m.storageFail,
		Bandwidth:        m.totalBytes,
	}
}

// Collector 是 v2 旧版内存收集器别名，保留兼容。
// Deprecated: 新代码请使用 Metrics 接口。
type Collector = memoryMetrics

// NewCollector 创建旧版 Collector（兼容 v2）
// Deprecated: 请使用 NewMemoryMetrics 或 New。
func NewCollector() *Collector {
	return &memoryMetrics{startTime: time.Now()}
}

// prometheusMetrics Prometheus 指标实现
type prometheusMetrics struct {
	mem             *memoryMetrics
	registry        *prometheus.Registry
	reqTotal        *prometheus.CounterVec
	reqDuration     *prometheus.HistogramVec
	parseTotal      prometheus.Counter
	parseFail       prometheus.Counter
	storageSave     prometheus.Counter
	storageSaveFail prometheus.Counter
	queueSize       prometheus.Gauge
	cbState         prometheus.Gauge
	prometheusCfg   model.PrometheusConfig
}

// NewPrometheusMetrics 创建 Prometheus 指标实现，注册指标但不启动 HTTP 服务
func NewPrometheusMetrics(cfg model.PrometheusConfig) (Metrics, error) {
	pm := &prometheusMetrics{
		mem:           &memoryMetrics{startTime: time.Now()},
		registry:      prometheus.NewRegistry(),
		prometheusCfg: cfg,
	}

	pm.reqTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "spider_requests_total",
		Help: "Total number of requests",
	}, []string{"status"})

	pm.reqDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "spider_request_duration_seconds",
		Help:    "Request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"status"})

	pm.parseTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "spider_parse_total",
		Help: "Total number of parse attempts",
	})

	pm.parseFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "spider_parse_fail_total",
		Help: "Total number of parse failures",
	})

	pm.storageSave = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "spider_storage_save_total",
		Help: "Total number of storage saves",
	})

	pm.storageSaveFail = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "spider_storage_save_fail_total",
		Help: "Total number of storage save failures",
	})

	pm.queueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "spider_worker_queue_size",
		Help: "Current worker queue size",
	})

	pm.cbState = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "spider_circuit_breaker_state",
		Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
	})

	pm.registry.MustRegister(
		pm.reqTotal, pm.reqDuration, pm.parseTotal,
		pm.parseFail, pm.storageSave, pm.storageSaveFail,
		pm.queueSize, pm.cbState,
	)

	return pm, nil
}

// handler 返回 Prometheus HTTP handler
func (p *prometheusMetrics) handler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}

// PrometheusHandler 获取 Metrics 实例对应的 Prometheus HTTP handler
func PrometheusHandler(m Metrics) http.Handler {
	if pm, ok := m.(*prometheusMetrics); ok {
		return pm.handler()
	}
	return promhttp.Handler()
}

// IncrSuccess 请求成功计数
func (p *prometheusMetrics) IncrSuccess(ctx context.Context) {
	p.mem.IncrSuccess(ctx)
	p.reqTotal.WithLabelValues("success").Inc()
}

// IncrFail 请求失败计数
func (p *prometheusMetrics) IncrFail(ctx context.Context) {
	p.mem.IncrFail(ctx)
	p.reqTotal.WithLabelValues("fail").Inc()
}

// IncrRetry 重试计数
func (p *prometheusMetrics) IncrRetry(ctx context.Context) {
	p.mem.IncrRetry(ctx)
}

// IncrParse 解析成功计数
func (p *prometheusMetrics) IncrParse(ctx context.Context) {
	p.mem.IncrParse(ctx)
	p.parseTotal.Inc()
}

// IncrParseFail 解析失败计数
func (p *prometheusMetrics) IncrParseFail(ctx context.Context) {
	p.mem.IncrParseFail(ctx)
	p.parseFail.Inc()
}

// IncrStorageSave 存储保存成功计数
func (p *prometheusMetrics) IncrStorageSave(ctx context.Context) {
	p.mem.IncrStorageSave(ctx)
	p.storageSave.Inc()
}

// IncrStorageSaveFail 存储保存失败计数
func (p *prometheusMetrics) IncrStorageSaveFail(ctx context.Context) {
	p.mem.IncrStorageSaveFail(ctx)
	p.storageSaveFail.Inc()
}

// AddBytes 记录流量字节数
func (p *prometheusMetrics) AddBytes(ctx context.Context, n int64) {
	p.mem.AddBytes(ctx, n)
}

// RecordRequestDuration 记录请求耗时（默认 success 标签，兼容旧调用）
func (p *prometheusMetrics) RecordRequestDuration(ctx context.Context, d time.Duration) {
	p.reqDuration.WithLabelValues("success").Observe(d.Seconds())
}

// RecordRequestDurationWithStatus 按状态标签记录请求耗时
func (p *prometheusMetrics) RecordRequestDurationWithStatus(ctx context.Context, status string, d time.Duration) {
	p.reqDuration.WithLabelValues(status).Observe(d.Seconds())
}

// SetQueueSize 设置 Worker 队列长度
func (p *prometheusMetrics) SetQueueSize(ctx context.Context, n int) {
	p.mem.SetQueueSize(ctx, n)
	p.queueSize.Set(float64(n))
}

// SetCircuitBreakerState 设置熔断器状态
func (p *prometheusMetrics) SetCircuitBreakerState(ctx context.Context, state int) {
	p.mem.SetCircuitBreakerState(ctx, state)
	p.cbState.Set(float64(state))
}

// GetStats 获取内存统计快照
func (p *prometheusMetrics) GetStats() model.Stats {
	return p.mem.GetStats()
}

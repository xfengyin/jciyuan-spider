// Package middleware 提供 Fetcher 中间件实现。
package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/metrics"
	"jciyuan-spider/internal/model"
)

// CircuitState 熔断器状态。
type CircuitState int

const (
	// StateClosed 关闭状态，请求正常通过。
	StateClosed CircuitState = iota
	// StateOpen 打开状态，快速失败。
	StateOpen
	// StateHalfOpen 半开状态，放行探测请求。
	StateHalfOpen
)

// circuitBreaker 实现三态熔断器。
type circuitBreaker struct {
	cfg       model.CircuitBreakerConfig
	state     CircuitState
	failures  int
	successes int
	window    []bool // true 表示成功
	openAt    time.Time
	mu        sync.Mutex
	m         metrics.Metrics
}

// newCircuitBreaker 创建熔断器实例。
func newCircuitBreaker(cfg model.CircuitBreakerConfig, m metrics.Metrics) *circuitBreaker {
	return &circuitBreaker{
		cfg:    cfg,
		state:  StateClosed,
		window: make([]bool, 0, cfg.WindowSize),
		m:      m,
	}
}

// CircuitBreakerMiddleware 创建熔断器中间件。
func CircuitBreakerMiddleware(cfg model.CircuitBreakerConfig, m metrics.Metrics) fetcher.Middleware {
	cb := newCircuitBreaker(cfg, m)
	return func(next fetcher.Handler) fetcher.Handler {
		return func(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
			if err := cb.before(); err != nil {
				return nil, err
			}
			resp, err := next(ctx, req)
			cb.after(err)
			return resp, err
		}
	}
}

// before 在请求前检查熔断器状态。
func (cb *circuitBreaker) before() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateOpen:
		if time.Since(cb.openAt) > cb.cfg.OpenDuration {
			cb.state = StateHalfOpen
			cb.failures = 0
			cb.successes = 0
			cb.emitState(StateHalfOpen)
		} else {
			return fmt.Errorf("熔断器已打开，拒绝请求")
		}
	case StateHalfOpen:
		if cb.successes+cb.failures >= cb.cfg.HalfOpenRequests {
			return fmt.Errorf("熔断器半开探测中，暂拒绝请求")
		}
	}
	return nil
}

// after 在请求后更新熔断器状态。
func (cb *circuitBreaker) after(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	success := err == nil
	cb.recordWindow(success)

	switch cb.state {
	case StateHalfOpen:
		if success {
			cb.successes++
			if cb.successes >= cb.cfg.HalfOpenRequests {
				cb.reset(StateClosed)
			}
		} else {
			cb.open()
		}
	case StateClosed:
		if success {
			cb.failures = 0
			return
		}
		cb.failures++
		if cb.failures >= cb.cfg.FailureThreshold {
			cb.open()
			return
		}
		if cb.cfg.ErrorRateThreshold > 0 && len(cb.window) >= cb.cfg.WindowSize {
			failureRate := cb.calculateFailureRate()
			if failureRate >= cb.cfg.ErrorRateThreshold {
				cb.open()
			}
		}
	}
}

// recordWindow 记录窗口结果。
func (cb *circuitBreaker) recordWindow(success bool) {
	if cb.cfg.WindowSize <= 0 {
		return
	}
	if len(cb.window) >= cb.cfg.WindowSize {
		cb.window = cb.window[1:]
	}
	cb.window = append(cb.window, success)
}

// calculateFailureRate 计算窗口内错误率。
func (cb *circuitBreaker) calculateFailureRate() float64 {
	if len(cb.window) == 0 {
		return 0
	}
	failures := 0
	for _, ok := range cb.window {
		if !ok {
			failures++
		}
	}
	return float64(failures) / float64(len(cb.window))
}

// open 切换到 Open 状态。
func (cb *circuitBreaker) open() {
	cb.state = StateOpen
	cb.openAt = time.Now()
	cb.emitState(StateOpen)
}

// reset 切换到 Closed 状态并重置计数。
func (cb *circuitBreaker) reset(state CircuitState) {
	cb.state = state
	cb.failures = 0
	cb.successes = 0
	cb.window = cb.window[:0]
	cb.emitState(state)
}

// emitState 上报熔断器状态指标。
func (cb *circuitBreaker) emitState(state CircuitState) {
	if cb.m == nil {
		return
	}
	cb.m.SetCircuitBreakerState(context.Background(), int(state))
}

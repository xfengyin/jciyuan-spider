// Package worker 提供基于信号量的并发 WorkerPool，支持优雅关闭、上下文取消与任务 panic 恢复。
package worker

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/semaphore"
)

// Task 是 WorkerPool 执行的任务单元。
type Task interface {
	// Execute 执行任务，ctx 取消时应立即返回。
	Execute(ctx context.Context) error
}

// TaskFunc 允许将函数作为 Task 使用。
type TaskFunc func(ctx context.Context) error

// Execute 实现 Task 接口。
func (f TaskFunc) Execute(ctx context.Context) error {
	return f(ctx)
}

// Pool Worker 池。
type Pool struct {
	workers   int
	queue     chan Task
	sem       *semaphore.Weighted
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	closed    bool
	started   bool
	startOnce sync.Once
	onPanic   func(interface{})
}

// NewPool 创建 Worker 池。
// workers 为最大并发数，queueSize 为任务队列长度（0 表示无缓冲）。
func NewPool(workers, queueSize int) *Pool {
	if workers <= 0 {
		workers = 1
	}
	return &Pool{
		workers: workers,
		queue:   make(chan Task, queueSize),
		sem:     semaphore.NewWeighted(int64(workers)),
	}
}

// Start 使用 parent 作为父上下文启动 Worker 消费任务。
// 多次调用仅生效一次。
func (p *Pool) Start(parent context.Context) {
	p.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(parent)
		p.ctx = ctx
		p.cancel = cancel
		p.started = true
		for i := 0; i < p.workers; i++ {
			p.wg.Add(1)
			go p.loop()
		}
	})
}

// SetPanicHandler 设置任务 panic 时的回调，便于上层记录日志或上报。
func (p *Pool) SetPanicHandler(fn func(interface{})) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onPanic = fn
}

// Submit 提交任务到队列。
// 当池已关闭或 ctx 取消时返回错误。
func (p *Pool) Submit(ctx context.Context, task Task) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return fmt.Errorf("worker pool is closed")
	}
	p.mu.Unlock()

	select {
	case p.queue <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

// loop 是单个 Worker 的事件循环。
func (p *Pool) loop() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.queue:
			if !ok {
				return
			}
			// 使用信号量控制并发。
			if err := p.sem.Acquire(p.ctx, 1); err != nil {
				return
			}
			p.runTask(task)
			p.sem.Release(1)
		}
	}
}

// runTask 执行任务并捕获 panic，避免单个任务导致整个 Worker 崩溃。
func (p *Pool) runTask(task Task) {
	defer func() {
		if r := recover(); r != nil {
			p.mu.Lock()
			onPanic := p.onPanic
			p.mu.Unlock()
			if onPanic != nil {
				onPanic(r)
			}
		}
	}()
	_ = task.Execute(p.ctx)
}

// Stop 优雅关闭 WorkerPool：停止接收新任务，等待队列中任务执行完毕后释放资源。
func (p *Pool) Stop() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	close(p.queue)
	p.wg.Wait()
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// QueueSize 返回当前队列长度。
func (p *Pool) QueueSize() int {
	return len(p.queue)
}

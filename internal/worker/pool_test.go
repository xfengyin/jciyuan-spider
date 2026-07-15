// Package worker 的单元测试。
package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPoolSubmitAndExecute 验证提交的任务会被并发执行。
func TestPoolSubmitAndExecute(t *testing.T) {
	var counter int64
	p := NewPool(2, 10)
	p.Start(context.Background())

	for i := 0; i < 10; i++ {
		err := p.Submit(context.Background(), TaskFunc(func(ctx context.Context) error {
			atomic.AddInt64(&counter, 1)
			return nil
		}))
		require.NoError(t, err)
	}

	require.NoError(t, p.Stop())
	assert.Equal(t, int64(10), atomic.LoadInt64(&counter))
}

// TestPoolStopNoNewTasks 验证关闭后不能再提交任务。
func TestPoolStopNoNewTasks(t *testing.T) {
	p := NewPool(1, 1)
	p.Start(context.Background())
	require.NoError(t, p.Stop())

	err := p.Submit(context.Background(), TaskFunc(func(ctx context.Context) error { return nil }))
	require.Error(t, err)
}

// TestPoolContextCancel 验证父上下文取消后任务停止。
func TestPoolContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := NewPool(1, 1)
	p.Start(ctx)

	var started sync.WaitGroup
	started.Add(1)
	err := p.Submit(context.Background(), TaskFunc(func(ctx context.Context) error {
		started.Done()
		<-ctx.Done()
		return nil
	}))
	require.NoError(t, err)

	started.Wait()
	cancel()
	assert.Eventually(t, func() bool {
		return p.ctx.Err() != nil
	}, time.Second, 10*time.Millisecond)
	_ = p.Stop()
}

// TestPoolPanicRecover 验证任务 panic 不会导致 Worker 崩溃且回调被触发。
func TestPoolPanicRecover(t *testing.T) {
	var recovered interface{}
	var mu sync.Mutex

	p := NewPool(1, 1)
	p.SetPanicHandler(func(r interface{}) {
		mu.Lock()
		recovered = r
		mu.Unlock()
	})
	p.Start(context.Background())

	err := p.Submit(context.Background(), TaskFunc(func(ctx context.Context) error {
		panic("boom")
	}))
	require.NoError(t, err)

	require.NoError(t, p.Stop())
	mu.Lock()
	assert.Equal(t, "boom", recovered)
	mu.Unlock()
}

// TestPoolConcurrencyLimit 验证并发数受 workers 限制。
func TestPoolConcurrencyLimit(t *testing.T) {
	var running int64
	var maxRunning int64
	var mu sync.Mutex

	p := NewPool(2, 10)
	p.Start(context.Background())

	for i := 0; i < 10; i++ {
		err := p.Submit(context.Background(), TaskFunc(func(ctx context.Context) error {
			n := atomic.AddInt64(&running, 1)
			mu.Lock()
			if n > maxRunning {
				maxRunning = n
			}
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt64(&running, -1)
			return nil
		}))
		require.NoError(t, err)
	}

	require.NoError(t, p.Stop())
	assert.LessOrEqual(t, maxRunning, int64(2))
}

// TestPoolQueueSize 验证队列长度能正确反映待处理任务数。
func TestPoolQueueSize(t *testing.T) {
	p := NewPool(1, 5)
	p.Start(context.Background())

	started := make(chan struct{})
	done := make(chan struct{})
	err := p.Submit(context.Background(), TaskFunc(func(ctx context.Context) error {
		close(started)
		<-done
		return nil
	}))
	require.NoError(t, err)

	<-started
	err = p.Submit(context.Background(), TaskFunc(func(ctx context.Context) error { return nil }))
	require.NoError(t, err)
	assert.Equal(t, 1, p.QueueSize())

	close(done)
	require.NoError(t, p.Stop())
}

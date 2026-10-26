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

// TestPoolCancelPreventsFurtherTasks 验证池被取消后，队列里尚未开始的任务不会再被执行。
//
// 两个容易写错的地方，这条用例刻意都避开了：
//  1. 必须等第一个任务**真的开始执行**再取消。只等「队列里有 3 个任务」不够 ——
//     worker 可能刚取走任务、还没进入任务体，此时取消会让它一个任务都不执行（started=0），
//     测出来的失败与要测的语义无关。
//  2. 必须重复多轮。取消之后 worker 的 select 在「ctx.Done()」与「队列可读」之间**随机**选，
//     单轮有一半几率恰好走 ctx.Done 分支，于是「取消后还启动下一个任务」这个缺陷看不出来。
func TestPoolCancelPreventsFurtherTasks(t *testing.T) {
	const rounds = 40
	for i := 0; i < rounds; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		p := NewPool(1, 5)
		p.Start(ctx)

		var started int64
		release := make(chan struct{})
		firstStarted := make(chan struct{})

		require.NoError(t, p.Submit(context.Background(), TaskFunc(func(context.Context) error {
			atomic.AddInt64(&started, 1)
			close(firstStarted)
			<-release
			return nil
		})))

		// 等第一个任务真的进入任务体，否则「它到底跑没跑」本身就是随机的。
		<-firstStarted

		for j := 0; j < 3; j++ {
			require.NoError(t, p.Submit(context.Background(), TaskFunc(func(context.Context) error {
				atomic.AddInt64(&started, 1)
				return nil
			})))
		}

		// 3 个任务都在队列里排队，本身没有被执行。
		require.Eventually(t, func() bool { return p.QueueSize() == 3 }, time.Second, 5*time.Millisecond)

		cancel()
		close(release)

		assert.Eventually(t, func() bool { return p.ctx.Err() != nil }, time.Second, 5*time.Millisecond)
		_ = p.Stop()

		require.Equal(t, int64(1), atomic.LoadInt64(&started), "第 %d 轮：取消之后不应再启动新任务", i)
	}
}

// TestPoolRunningTaskSeesCancellation 验证 ctx 取消会立即传导到正在执行的任务。
// 对应「Acquire 必须在 ctx 取消后立即返回」的语义：无论并发控制怎么实现，
// 取消都必须让任务在毫秒级退出，而不是等它自己跑完。
func TestPoolRunningTaskSeesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := NewPool(1, 0)
	p.Start(ctx)

	started := make(chan struct{})
	returned := make(chan error, 1)
	require.NoError(t, p.Submit(context.Background(), TaskFunc(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		returned <- ctx.Err()
		return nil
	})))

	<-started
	cancel()

	select {
	case err := <-returned:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("ctx 取消后任务没有在 2s 内返回")
	}
	_ = p.Stop()
}

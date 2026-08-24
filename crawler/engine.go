package crawler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Options 是 Engine 的运行选项。
type Options struct {
	// Concurrency 并发 Worker 数，默认 1。
	Concurrency int
	// Delay 每次 Fetch 前的请求间隔，用于限速（默认 0）。
	Delay time.Duration
	// MaxRetry 单个 URL 失败后的最大重试次数，默认 0（不重试）。
	MaxRetry int
	// Timeout 单次 Fetch 的超时，默认 30s。
	Timeout time.Duration
	// OutputDir 结果输出目录；为空则不落盘，仅在内存中返回。
	OutputDir string
	// Quiet 关闭进度日志。
	Quiet bool
}

// Stats 一次 Run 的运行统计。
type Stats struct {
	Total   int           // 处理的 URL 总数
	Success int           // 成功抓取的 URL 数
	Fail    int           // 最终失败的 URL 数
	Retry   int           // 重试次数
	Items   int           // 抽取到的条目总数
	Start   time.Time     // 开始时间
	End     time.Time     // 结束时间
	Elapsed time.Duration // 总耗时
}

// Failure 记录单个 URL 的最终失败信息。
type Failure struct {
	URL   string
	Error string
}

// Result 一次 Run 的结果：统计、条目与失败明细。
type Result struct {
	Stats    Stats
	Items    []Item
	Failures []Failure
}

// Engine 通用爬虫引擎：调度 Crawler 的 Fetch → Parse → Extract 流水线，
// 提供并发控制、失败重试、请求间隔限速与 JSON Lines 结果输出。
type Engine struct {
	c    Crawler
	opts Options
	log  func(format string, args ...any)
}

// NewEngine 创建引擎。opts 中的零值字段会应用默认值。
func NewEngine(c Crawler, opts Options) *Engine {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	e := &Engine{c: c, opts: opts}
	if !opts.Quiet {
		e.log = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "[crawler] "+format+"\n", args...)
		}
	}
	return e
}

// Run 抓取 urls 中的全部 URL，返回统计、条目与失败明细。
//
// 单个 URL 抓取失败（含重试后仍失败）不会中断整个运行：失败会被记录在
// Result.Failures 与 Stats.Fail 中。仅当引擎自身无法运行（如输出目录不可写）
// 时才返回非 nil 错误。
func (e *Engine) Run(ctx context.Context, urls []string) (*Result, error) {
	if e.c == nil {
		return nil, fmt.Errorf("crawler: 引擎缺少 Crawler 实现")
	}

	var writer *jsonlWriter
	if e.opts.OutputDir != "" {
		var err error
		writer, err = newJSONLWriter(e.opts.OutputDir)
		if err != nil {
			return nil, fmt.Errorf("crawler: 初始化输出目录失败: %w", err)
		}
		defer func() { _ = writer.Close() }()
	}

	start := time.Now()
	result := &Result{Stats: Stats{Total: len(urls), Start: start}}
	mu := sync.Mutex{}

	// 任务队列 + 固定数量 Worker 的简单并发模型，避免对单 URL 任务顺序的依赖。
	jobs := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < e.opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range jobs {
				e.processOne(ctx, url, writer, result, &mu)
			}
		}()
	}

	for _, u := range urls {
		jobs <- u
	}
	close(jobs)
	wg.Wait()

	result.Stats.End = time.Now()
	result.Stats.Elapsed = result.Stats.End.Sub(result.Stats.Start)
	if e.log != nil {
		e.log("完成: 总数=%d 成功=%d 失败=%d 重试=%d 条目=%d 耗时=%s",
			result.Stats.Total, result.Stats.Success, result.Stats.Fail,
			result.Stats.Retry, result.Stats.Items, result.Stats.Elapsed.Round(time.Millisecond))
	}
	return result, nil
}

// processOne 处理单个 URL：有限次 Fetch → Parse → Extract，失败按配置重试。
func (e *Engine) processOne(ctx context.Context, url string, writer *jsonlWriter, result *Result, mu *sync.Mutex) {
	if e.opts.Delay > 0 {
		time.Sleep(e.opts.Delay)
	}

	var page *Page
	var err error
	attemptsUsed := 0
	for attempt := 0; attempt <= e.opts.MaxRetry; attempt++ {
		attemptsUsed = attempt + 1
		if attempt > 0 {
			if e.log != nil {
				e.log("重试 %s (第 %d 次)", url, attempt)
			}
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				err = ctx.Err()
				goto finish
			case <-time.After(backoff):
			}
		}

		fetchCtx := ctx
		cancel := func() {}
		if e.opts.Timeout > 0 {
			fetchCtx, cancel = context.WithTimeout(ctx, e.opts.Timeout)
		}
		page, err = e.c.Fetch(fetchCtx, url)
		cancel()
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}
	}

finish:
	if err != nil {
		mu.Lock()
		result.Stats.Fail++
		result.Stats.Retry += attemptsUsed - 1
		result.Failures = append(result.Failures, Failure{URL: url, Error: err.Error()})
		mu.Unlock()
		if e.log != nil {
			e.log("失败 %s: %v", url, err)
		}
		return
	}

	parsed, err := e.c.Parse(ctx, page)
	if err != nil {
		mu.Lock()
		result.Stats.Fail++
		result.Stats.Retry += attemptsUsed - 1
		result.Failures = append(result.Failures, Failure{URL: url, Error: "parse: " + err.Error()})
		mu.Unlock()
		if e.log != nil {
			e.log("解析失败 %s: %v", url, err)
		}
		return
	}

	items, err := e.c.Extract(ctx, parsed)
	if err != nil {
		mu.Lock()
		result.Stats.Fail++
		result.Stats.Retry += attemptsUsed - 1
		result.Failures = append(result.Failures, Failure{URL: url, Error: "extract: " + err.Error()})
		mu.Unlock()
		if e.log != nil {
			e.log("抽取失败 %s: %v", url, err)
		}
		return
	}

	mu.Lock()
	result.Stats.Success++
	result.Stats.Retry += attemptsUsed - 1
	result.Stats.Items += len(items)
	result.Items = append(result.Items, items...)
	mu.Unlock()

	if writer != nil {
		for _, it := range items {
			if e.opts.Delay > 0 {
				time.Sleep(e.opts.Delay) // 输出同样限速，避免对目标站造成突发压力
			}
			if err := writer.Write(it); err != nil {
				if e.log != nil {
					e.log("写入输出失败: %v", err)
				}
			}
		}
	}
}

// jsonlWriter 将条目以 JSON Lines 格式追加写入输出目录。
type jsonlWriter struct {
	file *os.File
	bw   *bufio.Writer
	mu   sync.Mutex
}

// newJSONLWriter 创建输出文件 output/items.jsonl（自动建目录）。
func newJSONLWriter(dir string) (*jsonlWriter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "items.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &jsonlWriter{file: f, bw: bufio.NewWriter(f)}, nil
}

// Write 追加写入一条 JSON 记录。
func (w *jsonlWriter) Write(item Item) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if _, err := w.bw.Write(data); err != nil {
		return err
	}
	return w.bw.WriteByte('\n')
}

// Close 刷盘并关闭文件。
func (w *jsonlWriter) Close() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.bw.Flush(); err != nil {
		return err
	}
	return w.file.Close()
}

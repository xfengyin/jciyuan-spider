package crawler

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockCrawler 测试用爬虫：Fetch 可注入失败，Parse/Extract 固定返回。
type mockCrawler struct {
	name       string
	failFirst  int // 前 N 次 Fetch 失败（模拟重试）
	attempts   atomic.Int32
	fetchCalls atomic.Int32
	active     atomic.Int32
	maxActive  atomic.Int32
	mu         sync.Mutex
}

func (m *mockCrawler) Name() string { return "mock" }

func (m *mockCrawler) Fetch(ctx context.Context, url string) (*Page, error) {
	m.fetchCalls.Add(1)
	m.active.Add(1)
	defer m.active.Add(-1)
	for {
		cur := m.active.Load()
		prev := m.maxActive.Load()
		if cur <= prev || m.maxActive.CompareAndSwap(prev, cur) {
			break
		}
	}

	if n := m.attempts.Add(1); n <= int32(m.failFirst) {
		return nil, errors.New("mock fetch error")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &Page{URL: url, StatusCode: 200, Body: []byte("<html><h1>mock</h1></html>"), Text: "<html><h1>mock</h1></html>"}, nil
}

func (m *mockCrawler) Parse(_ context.Context, page *Page) (any, error) {
	return strings.ToUpper(page.Text), nil
}

func (m *mockCrawler) Extract(_ context.Context, parsed any) ([]Item, error) {
	return []Item{{"url": "mock://x", "html": parsed}}, nil
}

func runMock(t *testing.T, c Crawler, opts Options, urls []string) *Result {
	t.Helper()
	res, err := NewEngine(c, opts).Run(context.Background(), urls)
	if err != nil {
		t.Fatalf("Run 返回错误: %v", err)
	}
	return res
}

func TestEngineRunsPipeline(t *testing.T) {
	m := &mockCrawler{}
	res := runMock(t, m, Options{Concurrency: 2}, []string{"http://a.example", "http://b.example"})

	if res.Stats.Total != 2 || res.Stats.Success != 2 || res.Stats.Fail != 0 {
		t.Fatalf("统计异常: %+v", res.Stats)
	}
	if res.Stats.Items != 2 || len(res.Items) != 2 {
		t.Fatalf("条目数量异常: items=%d len=%d", res.Stats.Items, len(res.Items))
	}
	for _, it := range res.Items {
		if it["html"] != "<HTML><H1>MOCK</H1></HTML>" {
			t.Fatalf("Parse/Extract 链路未生效: %+v", it)
		}
	}
}

func TestEngineRetryThenSuccess(t *testing.T) {
	m := &mockCrawler{failFirst: 2}
	res := runMock(t, m, Options{MaxRetry: 3, Quiet: true}, []string{"http://a.example"})

	if res.Stats.Success != 1 || res.Stats.Fail != 0 {
		t.Fatalf("期望重试后成功: %+v", res.Stats)
	}
	if res.Stats.Retry != 2 {
		t.Fatalf("期望 2 次重试，实际 %d", res.Stats.Retry)
	}
}

func TestEngineFailAfterRetries(t *testing.T) {
	m := &mockCrawler{failFirst: 99}
	res := runMock(t, m, Options{MaxRetry: 2, Quiet: true}, []string{"http://a.example"})

	if res.Stats.Success != 0 || res.Stats.Fail != 1 {
		t.Fatalf("统计异常: %+v", res.Stats)
	}
	if res.Stats.Retry != 2 {
		t.Fatalf("期望 2 次重试，实际 %d", res.Stats.Retry)
	}
	if len(res.Failures) != 1 || res.Failures[0].URL != "http://a.example" {
		t.Fatalf("失败明细异常: %+v", res.Failures)
	}
}

func TestEngineOutputDir(t *testing.T) {
	dir := t.TempDir()
	m := &mockCrawler{}
	res := runMock(t, m, Options{OutputDir: dir}, []string{"http://a.example", "http://b.example"})

	if res.Stats.Success != 2 {
		t.Fatalf("统计异常: %+v", res.Stats)
	}
	data, err := os.ReadFile(filepath.Join(dir, "items.jsonl"))
	if err != nil {
		t.Fatalf("读取 items.jsonl 失败: %v", err)
	}
	lines := 0
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) != "" {
			lines++
		}
	}
	if lines != 2 {
		t.Fatalf("期望 2 行 JSONL，实际 %d: %s", lines, data)
	}
}

func TestEngineConcurrency(t *testing.T) {
	m := &mockCrawler{}
	urls := make([]string, 20)
	for i := range urls {
		urls[i] = "http://a.example/" + string(rune('a'+i))
	}
	runMock(t, m, Options{Concurrency: 4, Quiet: true}, urls)
	if m.maxActive.Load() > 4 {
		t.Fatalf("并发超过限制: maxActive=%d", m.maxActive.Load())
	}
	if m.fetchCalls.Load() != 20 {
		t.Fatalf("Fetch 调用次数异常: %d", m.fetchCalls.Load())
	}
}

func TestEngineNilCrawler(t *testing.T) {
	_, err := NewEngine(nil, Options{}).Run(context.Background(), []string{"http://a.example"})
	if err == nil {
		t.Fatal("nil Crawler 应返回错误")
	}
}

func TestCrawlerRegisterBuild(t *testing.T) {
	Register("test-build", func(cfg map[string]any) (Crawler, error) {
		if cfg["flag"] != "on" {
			return nil, errors.New("配置缺失")
		}
		return &mockCrawler{}, nil
	})
	defer delete(registry, "test-build")

	c, err := Build("test-build", map[string]any{"flag": "on"})
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}
	if c.Name() != "mock" {
		t.Fatalf("名称异常: %s", c.Name())
	}
	if _, err := Build("test-build", map[string]any{}); err == nil {
		t.Fatal("配置校验应失败")
	}
	if _, err := Build("no-such", nil); err == nil {
		t.Fatal("未知名称应报错")
	}
}

func TestEngineDelayRespected(t *testing.T) {
	m := &mockCrawler{}
	start := time.Now()
	runMock(t, m, Options{Delay: 30 * time.Millisecond, Quiet: true}, []string{"http://a.example", "http://b.example"})
	if time.Since(start) < 50*time.Millisecond {
		t.Fatalf("请求间隔未生效: 耗时 %s", time.Since(start))
	}
}

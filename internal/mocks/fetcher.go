// Package mocks 提供爬虫各接口的可复用 Mock 实现，用于单元/集成测试。
package mocks

import (
	"context"
	"fmt"
	"sync"

	"jciyuan-spider-v2/internal/fetcher"
)

// MockResponse 定义单个 URL 的模拟响应或错误。
type MockResponse struct {
	Resp *fetcher.Response
	Err  error
}

// MockFetcher 实现 fetcher.Fetcher 接口，按 URL 返回预设响应。
type MockFetcher struct {
	mu          sync.RWMutex
	responses   map[string]MockResponse
	defaultErr  error
	calls       []string
	callRecords map[string]int
}

// NewMockFetcher 创建 MockFetcher 实例。
func NewMockFetcher() *MockFetcher {
	return &MockFetcher{
		responses:   make(map[string]MockResponse),
		calls:       make([]string, 0),
		callRecords: make(map[string]int),
	}
}

// SetResponse 为指定 URL 设置模拟响应。
func (m *MockFetcher) SetResponse(url string, resp *fetcher.Response, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[url] = MockResponse{Resp: resp, Err: err}
}

// SetDefaultError 设置未命中 URL 时的默认错误。
func (m *MockFetcher) SetDefaultError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultErr = err
}

// Fetch 按请求 URL 返回预设结果，并记录调用。
func (m *MockFetcher) Fetch(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("请求对象不能为空")
	}

	m.mu.Lock()
	m.calls = append(m.calls, req.URL)
	m.callRecords[req.URL]++
	resp, ok := m.responses[req.URL]
	defaultErr := m.defaultErr
	m.mu.Unlock()

	if !ok {
		if defaultErr != nil {
			return nil, defaultErr
		}
		return nil, fmt.Errorf("未找到 URL 的模拟响应: %s", req.URL)
	}
	return resp.Resp, resp.Err
}

// Calls 返回所有被调用过的 URL 列表（按调用顺序）。
func (m *MockFetcher) Calls() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, len(m.calls))
	copy(out, m.calls)
	return out
}

// CallCount 返回指定 URL 被调用的次数。
func (m *MockFetcher) CallCount(url string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callRecords[url]
}

// Close 释放资源，Mock 实现无操作。
func (m *MockFetcher) Close() error { return nil }

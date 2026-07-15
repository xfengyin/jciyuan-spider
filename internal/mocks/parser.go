package mocks

import (
	"context"
	"fmt"
	"sync"

	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/parser"
)

// MockParser 实现 parser.Parser 接口，按输入 URL 返回预设解析结果。
type MockParser struct {
	mu         sync.RWMutex
	results    map[string]*parser.ParseResult
	errs       map[string]error
	defaultErr error
}

// NewMockParser 创建 MockParser 实例。
func NewMockParser() *MockParser {
	return &MockParser{
		results: make(map[string]*parser.ParseResult),
		errs:    make(map[string]error),
	}
}

// SetResult 为指定 URL 设置解析结果或错误。
func (m *MockParser) SetResult(url string, result *parser.ParseResult, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results[url] = result
	if err != nil {
		m.errs[url] = err
	} else {
		delete(m.errs, url)
	}
}

// SetDefaultError 设置未命中 URL 时的默认错误。
func (m *MockParser) SetDefaultError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultErr = err
}

// Parse 按响应 URL 返回预设结果。
func (m *MockParser) Parse(ctx context.Context, resp *fetcher.Response) (*parser.ParseResult, error) {
	if resp == nil {
		return nil, fmt.Errorf("响应对象不能为空")
	}

	m.mu.RLock()
	result, hasResult := m.results[resp.URL]
	err, hasErr := m.errs[resp.URL]
	defaultErr := m.defaultErr
	m.mu.RUnlock()

	if !hasResult {
		if defaultErr != nil {
			return nil, defaultErr
		}
		return nil, fmt.Errorf("未找到 URL 的解析结果: %s", resp.URL)
	}
	if hasErr {
		return nil, err
	}
	return result, nil
}

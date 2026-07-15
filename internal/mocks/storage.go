package mocks

import (
	"context"
	"fmt"
	"sync"

	"jciyuan-spider-v2/internal/model"
	"jciyuan-spider-v2/internal/storage"
)

// compile-time 接口校验。
var (
	_ storage.Storage       = (*MockStorage)(nil)
	_ storage.StatusStorage = (*MockStorage)(nil)
)

// MockStorage 实现 storage.Storage 与 storage.StatusStorage 的内存版本，
// 用于单元/集成测试。
type MockStorage struct {
	mu        sync.RWMutex
	data      map[int64]*model.AnimeInfo
	statuses  map[int64]*model.CrawlStatus
	saveCount int
	saveErr   error
}

// NewMockStorage 创建 MockStorage 实例。
func NewMockStorage() *MockStorage {
	return &MockStorage{
		data:     make(map[int64]*model.AnimeInfo),
		statuses: make(map[int64]*model.CrawlStatus),
	}
}

// Save 保存动漫信息到内存。
func (m *MockStorage) Save(ctx context.Context, anime *model.AnimeInfo) error {
	if anime == nil {
		return fmt.Errorf("anime 不能为空")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	m.data[anime.ID] = anime
	m.saveCount++
	return nil
}

// Load 从内存加载动漫信息。
func (m *MockStorage) Load(ctx context.Context, animeID int64) (*model.AnimeInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[animeID], nil
}

// Exists 检查动漫是否存在。
func (m *MockStorage) Exists(ctx context.Context, animeID int64) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[animeID]
	return ok, nil
}

// SaveBatch 批量保存动漫信息。
func (m *MockStorage) SaveBatch(ctx context.Context, animes []*model.AnimeInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	for _, anime := range animes {
		if anime == nil {
			continue
		}
		m.data[anime.ID] = anime
		m.saveCount++
	}
	return nil
}

// SaveStatus 保存爬取状态到内存。
func (m *MockStorage) SaveStatus(ctx context.Context, status *model.CrawlStatus) error {
	if status == nil {
		return fmt.Errorf("status 不能为空")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses[status.AnimeID] = status
	return nil
}

// LoadStatus 从内存加载爬取状态。
func (m *MockStorage) LoadStatus(ctx context.Context, animeID int64) (*model.CrawlStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statuses[animeID], nil
}

// Close 释放资源，Mock 实现无操作。
func (m *MockStorage) Close() error { return nil }

// SetSaveError 设置 Save/SaveBatch 返回的固定错误，用于模拟存储失败。
func (m *MockStorage) SetSaveError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveErr = err
}

// SaveCount 返回 Save/SaveBatch 触发的保存次数。
func (m *MockStorage) SaveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.saveCount
}

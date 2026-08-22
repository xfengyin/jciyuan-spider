// Package di 提供 StatusStorage 的内存兜底实现，当存储后端未实现状态接口时使用。
package di

import (
	"context"
	"sync"
	"time"

	"jciyuan-spider/internal/model"
	"jciyuan-spider/internal/storage"
)

// memoryStatusStore 是基于内存的 CrawlStatus 存储，仅作兜底使用。
// 进程退出后状态会丢失，因此真正的断点续爬仍建议选用实现 StatusStorage 的持久化后端。
type memoryStatusStore struct {
	mu       sync.RWMutex
	statuses map[int64]*model.CrawlStatus
}

// compile-time 检查，确保 memoryStatusStore 实现 storage.StatusStorage。
var _ storage.StatusStorage = (*memoryStatusStore)(nil)

// newMemoryStatusStore 创建内存状态存储实例。
func newMemoryStatusStore() *memoryStatusStore {
	return &memoryStatusStore{
		statuses: make(map[int64]*model.CrawlStatus),
	}
}

// SaveStatus 保存爬取状态到内存。
func (s *memoryStatusStore) SaveStatus(_ context.Context, status *model.CrawlStatus) error {
	if status == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	status.LastCrawlAt = time.Now()
	s.statuses[status.AnimeID] = status
	return nil
}

// LoadStatus 从内存加载爬取状态。
func (s *memoryStatusStore) LoadStatus(_ context.Context, animeID int64) (*model.CrawlStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statuses[animeID], nil
}

package storage

import (
	"sync"

	"jciyuan-spider-v2/internal/model"
)

// MemoryStorage 内存缓存存储（装饰器模式，包装持久化存储）
type MemoryStorage struct {
	data        map[int64]*model.AnimeInfo
	statuses    map[int64]*model.CrawlStatus
	mu          sync.RWMutex
	persistence Storage
	statusStore StatusStorage
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage(persistence Storage, statusStore StatusStorage) *MemoryStorage {
	return &MemoryStorage{
		data:        make(map[int64]*model.AnimeInfo),
		statuses:    make(map[int64]*model.CrawlStatus),
		persistence: persistence,
		statusStore: statusStore,
	}
}

// Save 保存到内存和持久化层
func (s *MemoryStorage) Save(anime *model.AnimeInfo) error {
	s.mu.Lock()
	s.data[anime.ID] = anime
	s.mu.Unlock()

	if s.persistence != nil {
		return s.persistence.Save(anime)
	}
	return nil
}

// Load 优先从内存加载
func (s *MemoryStorage) Load(animeID int64) (*model.AnimeInfo, error) {
	s.mu.RLock()
	if anime, ok := s.data[animeID]; ok {
		s.mu.RUnlock()
		return anime, nil
	}
	s.mu.RUnlock()

	if s.persistence != nil {
		return s.persistence.Load(animeID)
	}
	return nil, nil
}

// Exists 检查是否存在
func (s *MemoryStorage) Exists(animeID int64) bool {
	s.mu.RLock()
	if _, ok := s.data[animeID]; ok {
		s.mu.RUnlock()
		return true
	}
	s.mu.RUnlock()

	if s.persistence != nil {
		return s.persistence.Exists(animeID)
	}
	return false
}

// SaveStatus 保存状态
func (s *MemoryStorage) SaveStatus(status *model.CrawlStatus) error {
	s.mu.Lock()
	s.statuses[status.AnimeID] = status
	s.mu.Unlock()

	if s.statusStore != nil {
		return s.statusStore.SaveStatus(status)
	}
	return nil
}

// LoadStatus 加载状态
func (s *MemoryStorage) LoadStatus(animeID int64) (*model.CrawlStatus, error) {
	s.mu.RLock()
	if status, ok := s.statuses[animeID]; ok {
		s.mu.RUnlock()
		return status, nil
	}
	s.mu.RUnlock()

	if s.statusStore != nil {
		return s.statusStore.LoadStatus(animeID)
	}
	return nil, nil
}

// Close 关闭存储
func (s *MemoryStorage) Close() error {
	if s.persistence != nil {
		return s.persistence.Close()
	}
	return nil
}

// Package memorystorage 提供基于内存的 L1 缓存装饰器，包装持久化 Storage 与 StatusStorage。
package memorystorage

import (
	"context"
	"sync"
	"time"

	"jciyuan-spider/internal/model"
	"jciyuan-spider/internal/storage"
)

// cacheItem 是带过期时间的缓存项。
type cacheItem[T any] struct {
	value      T
	expireAt   time.Time
}

// MemoryStorage 内存缓存存储（装饰器模式）。
type MemoryStorage struct {
	data        map[int64]cacheItem[*model.AnimeInfo]
	statuses    map[int64]cacheItem[*model.CrawlStatus]
	mu          sync.RWMutex
	ttl         time.Duration
	persistence storage.Storage
	statusStore storage.StatusStorage
}

// NewMemoryStorage 创建内存缓存装饰器。
func NewMemoryStorage(persistence storage.Storage, statusStore storage.StatusStorage, ttl time.Duration) *MemoryStorage {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &MemoryStorage{
		data:        make(map[int64]cacheItem[*model.AnimeInfo]),
		statuses:    make(map[int64]cacheItem[*model.CrawlStatus]),
		ttl:         ttl,
		persistence: persistence,
		statusStore: statusStore,
	}
}

// NewMemoryStorageFromConfig 根据 StorageConfig 包装已有 Storage 实例。
func NewMemoryStorageFromConfig(cfg model.StorageConfig, persistence storage.Storage) *MemoryStorage {
	var statusStore storage.StatusStorage
	if s, ok := persistence.(storage.StatusStorage); ok {
		statusStore = s
	}
	return NewMemoryStorage(persistence, statusStore, cfg.Memory.TTL)
}

// Save 保存到内存和持久化层。
func (s *MemoryStorage) Save(ctx context.Context, anime *model.AnimeInfo) error {
	s.mu.Lock()
	s.data[anime.ID] = cacheItem[*model.AnimeInfo]{value: anime, expireAt: time.Now().Add(s.ttl)}
	s.mu.Unlock()

	if s.persistence != nil {
		return s.persistence.Save(ctx, anime)
	}
	return nil
}

// Load 优先从内存加载，未命中或已过期则回源持久层。
func (s *MemoryStorage) Load(ctx context.Context, animeID int64) (*model.AnimeInfo, error) {
	s.mu.RLock()
	item, ok := s.data[animeID]
	s.mu.RUnlock()

	if ok && time.Now().Before(item.expireAt) {
		return item.value, nil
	}

	if s.persistence != nil {
		anime, err := s.persistence.Load(ctx, animeID)
		if err != nil || anime == nil {
			return anime, err
		}
		s.mu.Lock()
		s.data[animeID] = cacheItem[*model.AnimeInfo]{value: anime, expireAt: time.Now().Add(s.ttl)}
		s.mu.Unlock()
		return anime, nil
	}
	return nil, nil
}

// Exists 检查是否存在，优先内存。
func (s *MemoryStorage) Exists(ctx context.Context, animeID int64) (bool, error) {
	s.mu.RLock()
	item, ok := s.data[animeID]
	s.mu.RUnlock()

	if ok && time.Now().Before(item.expireAt) {
		return true, nil
	}

	if s.persistence != nil {
		return s.persistence.Exists(ctx, animeID)
	}
	return false, nil
}

// SaveBatch 批量保存，逐条写入内存并同步回写持久层。
func (s *MemoryStorage) SaveBatch(ctx context.Context, animes []*model.AnimeInfo) error {
	if len(animes) == 0 {
		return nil
	}

	// 先更新内存缓存，保持最终一致性。
	s.mu.Lock()
	now := time.Now()
	for _, anime := range animes {
		s.data[anime.ID] = cacheItem[*model.AnimeInfo]{value: anime, expireAt: now.Add(s.ttl)}
	}
	s.mu.Unlock()

	if s.persistence != nil {
		return s.persistence.SaveBatch(ctx, animes)
	}
	return nil
}

// SaveStatus 保存状态到内存和持久化层。
func (s *MemoryStorage) SaveStatus(ctx context.Context, status *model.CrawlStatus) error {
	s.mu.Lock()
	s.statuses[status.AnimeID] = cacheItem[*model.CrawlStatus]{value: status, expireAt: time.Now().Add(s.ttl)}
	s.mu.Unlock()

	if s.statusStore != nil {
		return s.statusStore.SaveStatus(ctx, status)
	}
	return nil
}

// LoadStatus 优先从内存加载状态。
func (s *MemoryStorage) LoadStatus(ctx context.Context, animeID int64) (*model.CrawlStatus, error) {
	s.mu.RLock()
	item, ok := s.statuses[animeID]
	s.mu.RUnlock()

	if ok && time.Now().Before(item.expireAt) {
		return item.value, nil
	}

	if s.statusStore != nil {
		status, err := s.statusStore.LoadStatus(ctx, animeID)
		if err != nil || status == nil {
			return status, err
		}
		s.mu.Lock()
		s.statuses[animeID] = cacheItem[*model.CrawlStatus]{value: status, expireAt: time.Now().Add(s.ttl)}
		s.mu.Unlock()
		return status, nil
	}
	return nil, nil
}

// Close 关闭存储，并清理内存缓存。
func (s *MemoryStorage) Close() error {
	s.mu.Lock()
	s.data = make(map[int64]cacheItem[*model.AnimeInfo])
	s.statuses = make(map[int64]cacheItem[*model.CrawlStatus])
	s.mu.Unlock()

	if s.persistence != nil {
		return s.persistence.Close()
	}
	return nil
}

// Evict 手动淘汰指定动漫的内存缓存。
func (s *MemoryStorage) Evict(animeID int64) {
	s.mu.Lock()
	delete(s.data, animeID)
	delete(s.statuses, animeID)
	s.mu.Unlock()
}

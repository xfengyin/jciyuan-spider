// Package memorystorage 的单元测试。
package memorystorage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jciyuan-spider-v2/internal/model"
)

// mockPersistence 是一个极简的持久化存储，用于验证内存装饰器是否回写。
type mockPersistence struct {
	saved   []*model.AnimeInfo
	status  *model.CrawlStatus
	loadFn  func(ctx context.Context, id int64) (*model.AnimeInfo, error)
	exists  map[int64]bool
}

func (m *mockPersistence) Save(ctx context.Context, anime *model.AnimeInfo) error {
	m.saved = append(m.saved, anime)
	return nil
}

func (m *mockPersistence) Load(ctx context.Context, animeID int64) (*model.AnimeInfo, error) {
	if m.loadFn != nil {
		return m.loadFn(ctx, animeID)
	}
	return nil, nil
}

func (m *mockPersistence) Exists(ctx context.Context, animeID int64) (bool, error) {
	return m.exists[animeID], nil
}

func (m *mockPersistence) SaveBatch(ctx context.Context, animes []*model.AnimeInfo) error {
	m.saved = append(m.saved, animes...)
	return nil
}

func (m *mockPersistence) SaveStatus(ctx context.Context, status *model.CrawlStatus) error {
	m.status = status
	return nil
}

func (m *mockPersistence) LoadStatus(ctx context.Context, animeID int64) (*model.CrawlStatus, error) {
	return m.status, nil
}

func (m *mockPersistence) Close() error { return nil }

// newTestAnime 返回一个用于测试的 AnimeInfo 实例。
func newTestAnime(id int64) *model.AnimeInfo {
	return &model.AnimeInfo{
		ID:          id,
		Title:       "测试动漫",
		Year:        "2024",
		Region:      "日本",
		Tags:        []string{"热血"},
		Description: "测试描述",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// TestMemoryStorageSaveAndLoad 验证保存后能从内存直接命中。
func TestMemoryStorageSaveAndLoad(t *testing.T) {
	persist := &mockPersistence{exists: make(map[int64]bool)}
	s := NewMemoryStorage(persist, persist, time.Hour)
	defer s.Close()

	anime := newTestAnime(100)
	require.NoError(t, s.Save(context.Background(), anime))

	loaded, err := s.Load(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, anime.Title, loaded.Title)
	assert.Len(t, persist.saved, 1)
}

// TestMemoryStorageExistsFromCache 验证命中缓存时直接返回存在。
func TestMemoryStorageExistsFromCache(t *testing.T) {
	persist := &mockPersistence{exists: make(map[int64]bool)}
	s := NewMemoryStorage(persist, persist, time.Hour)
	defer s.Close()

	require.NoError(t, s.Save(context.Background(), newTestAnime(100)))
	ok, err := s.Exists(context.Background(), 100)
	require.NoError(t, err)
	assert.True(t, ok)
}

// TestMemoryStorageCacheMiss 验证未命中时回源持久层。
func TestMemoryStorageCacheMiss(t *testing.T) {
	persist := &mockPersistence{
		exists: make(map[int64]bool),
		loadFn: func(ctx context.Context, id int64) (*model.AnimeInfo, error) {
			return newTestAnime(id), nil
		},
	}
	s := NewMemoryStorage(persist, persist, time.Hour)
	defer s.Close()

	loaded, err := s.Load(context.Background(), 200)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, int64(200), loaded.ID)
}

// TestMemoryStorageTTLExpiration 验证过期缓存不再命中。
func TestMemoryStorageTTLExpiration(t *testing.T) {
	persist := &mockPersistence{exists: make(map[int64]bool)}
	s := NewMemoryStorage(persist, persist, 1*time.Millisecond)
	defer s.Close()

	require.NoError(t, s.Save(context.Background(), newTestAnime(100)))
	time.Sleep(5 * time.Millisecond)

	ok, err := s.Exists(context.Background(), 100)
	require.NoError(t, err)
	assert.False(t, ok)
}

// TestMemoryStorageSaveBatch 验证批量保存会同步到持久层。
func TestMemoryStorageSaveBatch(t *testing.T) {
	persist := &mockPersistence{exists: make(map[int64]bool)}
	s := NewMemoryStorage(persist, persist, time.Hour)
	defer s.Close()

	animes := []*model.AnimeInfo{
		newTestAnime(301),
		newTestAnime(302),
	}
	require.NoError(t, s.SaveBatch(context.Background(), animes))
	assert.Len(t, persist.saved, 2)
}

// TestMemoryStorageSaveAndLoadStatus 验证状态缓存与回写。
func TestMemoryStorageSaveAndLoadStatus(t *testing.T) {
	persist := &mockPersistence{exists: make(map[int64]bool)}
	s := NewMemoryStorage(persist, persist, time.Hour)
	defer s.Close()

	status := &model.CrawlStatus{
		AnimeID:      100,
		Status:       "running",
		CurrentIndex: 5,
	}
	require.NoError(t, s.SaveStatus(context.Background(), status))

	loaded, err := s.LoadStatus(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, status.Status, loaded.Status)
	assert.NotNil(t, persist.status)
}

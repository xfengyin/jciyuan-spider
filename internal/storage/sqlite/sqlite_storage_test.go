// Package sqlitestorage 的单元测试。
package sqlitestorage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jciyuan-spider-v2/internal/model"
)

// newTestAnime 返回一个用于测试的 AnimeInfo 实例。
func newTestAnime(id int64) *model.AnimeInfo {
	return &model.AnimeInfo{
		ID:          id,
		Title:       "测试动漫",
		Year:        "2024",
		Region:      "日本",
		Tags:        []string{"热血", "冒险"},
		Description: "测试描述",
		Episodes: []model.Episode{
			{AnimeID: id, Number: 1, Title: "第01集", URL: "https://example.com/1"},
			{AnimeID: id, Number: 2, Title: "第02集", URL: "https://example.com/2"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// TestSQLiteStorageSaveAndLoad 验证保存后加载完整记录与分集。
func TestSQLiteStorageSaveAndLoad(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStorage(dsn)
	require.NoError(t, err)
	defer s.Close()

	anime := newTestAnime(100)
	require.NoError(t, s.Save(context.Background(), anime))

	loaded, err := s.Load(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, anime.Title, loaded.Title)
	assert.Equal(t, anime.Tags, loaded.Tags)
	require.Len(t, loaded.Episodes, 2)
	assert.Equal(t, 1, loaded.Episodes[0].Number)
	assert.Equal(t, 2, loaded.Episodes[1].Number)
}

// TestSQLiteStorageUpsert 验证重复保存会更新内容。
func TestSQLiteStorageUpsert(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStorage(dsn)
	require.NoError(t, err)
	defer s.Close()

	anime := newTestAnime(100)
	require.NoError(t, s.Save(context.Background(), anime))

	anime.Title = "更新标题"
	require.NoError(t, s.Save(context.Background(), anime))

	loaded, err := s.Load(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "更新标题", loaded.Title)
}

// TestSQLiteStorageExists 验证存在性检查。
func TestSQLiteStorageExists(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStorage(dsn)
	require.NoError(t, err)
	defer s.Close()

	ok, err := s.Exists(context.Background(), 200)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, s.Save(context.Background(), newTestAnime(200)))
	ok, err = s.Exists(context.Background(), 200)
	require.NoError(t, err)
	assert.True(t, ok)
}

// TestSQLiteStorageSaveBatch 验证批量保存后记录可加载。
func TestSQLiteStorageSaveBatch(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStorage(dsn)
	require.NoError(t, err)
	defer s.Close()

	animes := []*model.AnimeInfo{
		newTestAnime(301),
		newTestAnime(302),
	}
	require.NoError(t, s.SaveBatch(context.Background(), animes))

	for _, anime := range animes {
		loaded, err := s.Load(context.Background(), anime.ID)
		require.NoError(t, err)
		require.NotNil(t, loaded)
		assert.Equal(t, anime.Title, loaded.Title)
	}
}

// TestSQLiteStorageSaveAndLoadStatus 验证状态保存与加载。
func TestSQLiteStorageSaveAndLoadStatus(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStorage(dsn)
	require.NoError(t, err)
	defer s.Close()

	status := &model.CrawlStatus{
		AnimeID:      100,
		Status:       "running",
		CurrentIndex: 5,
		TotalCount:   10,
	}
	require.NoError(t, s.SaveStatus(context.Background(), status))

	loaded, err := s.LoadStatus(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, status.Status, loaded.Status)
	assert.Equal(t, status.CurrentIndex, loaded.CurrentIndex)
}

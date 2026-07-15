// Package jsonstorage 的单元测试。
package jsonstorage

import (
	"context"
	"fmt"
	"os"
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
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// TestJSONStorageSaveAndLoad 验证保存后可正确加载。
func TestJSONStorageSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	s, err := NewJSONStorage(dir)
	require.NoError(t, err)

	anime := newTestAnime(100)
	require.NoError(t, s.Save(context.Background(), anime))

	loaded, err := s.Load(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, anime.Title, loaded.Title)
	assert.Equal(t, anime.Tags, loaded.Tags)
}

// TestJSONStorageLoadNotFound 验证不存在的记录返回 nil。
func TestJSONStorageLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	s, err := NewJSONStorage(dir)
	require.NoError(t, err)

	loaded, err := s.Load(context.Background(), 999)
	require.NoError(t, err)
	assert.Nil(t, loaded)
}

// TestJSONStorageExists 验证 Exists 方法行为。
func TestJSONStorageExists(t *testing.T) {
	dir := t.TempDir()
	s, err := NewJSONStorage(dir)
	require.NoError(t, err)

	ok, err := s.Exists(context.Background(), 200)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, s.Save(context.Background(), newTestAnime(200)))
	ok, err = s.Exists(context.Background(), 200)
	require.NoError(t, err)
	assert.True(t, ok)
}

// TestJSONStorageSaveBatch 验证批量保存后每个文件都存在。
func TestJSONStorageSaveBatch(t *testing.T) {
	dir := t.TempDir()
	s, err := NewJSONStorage(dir)
	require.NoError(t, err)

	animes := []*model.AnimeInfo{
		newTestAnime(301),
		newTestAnime(302),
	}
	require.NoError(t, s.SaveBatch(context.Background(), animes))

	for _, anime := range animes {
		filename := filepath.Join(dir, fmt.Sprintf("%d.json", anime.ID))
		_, err := os.Stat(filename)
		require.NoError(t, err)
	}
}

// TestJSONStorageSaveBatchEmpty 验证空批量保存不报错。
func TestJSONStorageSaveBatchEmpty(t *testing.T) {
	dir := t.TempDir()
	s, err := NewJSONStorage(dir)
	require.NoError(t, err)
	assert.NoError(t, s.SaveBatch(context.Background(), nil))
}

// TestJSONStorageSaveAndLoadStatus 验证状态保存与加载。
func TestJSONStorageSaveAndLoadStatus(t *testing.T) {
	dir := t.TempDir()
	s, err := NewJSONStorage(dir)
	require.NoError(t, err)

	status := &model.CrawlStatus{
		AnimeID:     100,
		Status:      "running",
		CurrentIndex: 5,
		TotalCount:  10,
	}
	require.NoError(t, s.SaveStatus(context.Background(), status))

	loaded, err := s.LoadStatus(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, status.Status, loaded.Status)
	assert.Equal(t, status.CurrentIndex, loaded.CurrentIndex)
}

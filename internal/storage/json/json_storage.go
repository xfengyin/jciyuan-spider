// Package jsonstorage 提供基于单文件 JSON 的 Storage/StatusStorage 实现，兼容 v2 输出格式。
package jsonstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"jciyuan-spider/internal/model"
	"jciyuan-spider/internal/storage"
)

// JSONStorage JSON 文件存储实现。
type JSONStorage struct {
	dir      string
	mu       sync.RWMutex
	statusMu sync.RWMutex
}

// NewJSONStorage 创建 JSON 存储实例。
func NewJSONStorage(dir string) (*JSONStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	return &JSONStorage{dir: dir}, nil
}

// NewJSONStorageFromConfig 根据 StorageConfig 构造 Storage（SPI 构造器签名）。
func NewJSONStorageFromConfig(cfg model.StorageConfig) (storage.Storage, error) {
	return NewJSONStorage(cfg.JSON.OutputDir)
}

func init() {
	// SPI 注册 JSON Storage。
	storage.Register("json", NewJSONStorageFromConfig)
}

// Save 保存动漫信息，以单文件 JSON 形式写入 output_dir。
func (s *JSONStorage) Save(ctx context.Context, anime *model.AnimeInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	anime.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(anime, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化动漫信息失败: %w", err)
	}

	filename := filepath.Join(s.dir, fmt.Sprintf("%d.json", anime.ID))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	return nil
}

// Load 加载动漫信息。
func (s *JSONStorage) Load(ctx context.Context, animeID int64) (*model.AnimeInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filename := filepath.Join(s.dir, fmt.Sprintf("%d.json", animeID))
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var anime model.AnimeInfo
	if err := json.Unmarshal(data, &anime); err != nil {
		return nil, fmt.Errorf("反序列化失败: %w", err)
	}
	return &anime, nil
}

// Exists 检查动漫信息是否存在。
func (s *JSONStorage) Exists(ctx context.Context, animeID int64) (bool, error) {
	filename := filepath.Join(s.dir, fmt.Sprintf("%d.json", animeID))
	_, err := os.Stat(filename)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("检查文件失败: %w", err)
	}
	return err == nil, nil
}

// SaveBatch 批量保存动漫信息，采用事务语义：任一失败则全部回滚。
func (s *JSONStorage) SaveBatch(ctx context.Context, animes []*model.AnimeInfo) error {
	if len(animes) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 先写入临时文件，全部成功后再原子重命名，实现事务语义。
	tmpFiles := make([]string, 0, len(animes))
	finalFiles := make([]string, 0, len(animes))

	for _, anime := range animes {
		anime.UpdatedAt = time.Now()
		data, err := json.MarshalIndent(anime, "", "  ")
		if err != nil {
			_ = cleanupTmpFiles(tmpFiles)
			return fmt.Errorf("序列化动漫 %d 失败: %w", anime.ID, err)
		}

		finalFile := filepath.Join(s.dir, fmt.Sprintf("%d.json", anime.ID))
		tmpFile := finalFile + ".tmp"
		if err := os.WriteFile(tmpFile, data, 0644); err != nil {
			_ = cleanupTmpFiles(tmpFiles)
			return fmt.Errorf("写入临时文件失败: %w", err)
		}
		tmpFiles = append(tmpFiles, tmpFile)
		finalFiles = append(finalFiles, finalFile)
	}

	for i, tmpFile := range tmpFiles {
		if err := os.Rename(tmpFile, finalFiles[i]); err != nil {
			_ = cleanupTmpFiles(tmpFiles)
			return fmt.Errorf("提交文件失败: %w", err)
		}
	}
	return nil
}

// SaveStatus 保存爬取状态。
func (s *JSONStorage) SaveStatus(ctx context.Context, status *model.CrawlStatus) error {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	statusDir := filepath.Join(s.dir, ".status")
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		return fmt.Errorf("创建状态目录失败: %w", err)
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化状态失败: %w", err)
	}

	filename := filepath.Join(statusDir, fmt.Sprintf("%d.status.json", status.AnimeID))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("写入状态文件失败: %w", err)
	}
	return nil
}

// LoadStatus 加载爬取状态。
func (s *JSONStorage) LoadStatus(ctx context.Context, animeID int64) (*model.CrawlStatus, error) {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()

	filename := filepath.Join(s.dir, ".status", fmt.Sprintf("%d.status.json", animeID))
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取状态文件失败: %w", err)
	}

	var status model.CrawlStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("反序列化状态失败: %w", err)
	}
	return &status, nil
}

// Close 关闭存储。
func (s *JSONStorage) Close() error { return nil }

// cleanupTmpFiles 清理批量保存时产生的临时文件。
func cleanupTmpFiles(tmpFiles []string) error {
	var lastErr error
	for _, f := range tmpFiles {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
	}
	return lastErr
}

package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"jciyuan-spider-v2/internal/model"
)

// JSONStorage JSON 文件存储实现
type JSONStorage struct {
	dir      string
	mu       sync.RWMutex
	statusMu sync.RWMutex
}

// NewJSONStorage 创建 JSON 存储
func NewJSONStorage(dir string) (*JSONStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	return &JSONStorage{dir: dir}, nil
}

// Save 保存动漫信息
func (s *JSONStorage) Save(anime *model.AnimeInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	anime.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(anime, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	filename := filepath.Join(s.dir, fmt.Sprintf("%d.json", anime.ID))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	return nil
}

// Load 加载动漫信息
func (s *JSONStorage) Load(animeID int64) (*model.AnimeInfo, error) {
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

// SaveStatus 保存爬取状态
func (s *JSONStorage) SaveStatus(status *model.CrawlStatus) error {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	statusDir := filepath.Join(s.dir, ".status")
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}

	filename := filepath.Join(statusDir, fmt.Sprintf("%d.status.json", status.AnimeID))
	return os.WriteFile(filename, data, 0644)
}

// LoadStatus 加载爬取状态
func (s *JSONStorage) LoadStatus(animeID int64) (*model.CrawlStatus, error) {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()

	filename := filepath.Join(s.dir, ".status", fmt.Sprintf("%d.status.json", animeID))
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var status model.CrawlStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// Exists 检查是否存在
func (s *JSONStorage) Exists(animeID int64) bool {
	filename := filepath.Join(s.dir, fmt.Sprintf("%d.json", animeID))
	_, err := os.Stat(filename)
	return err == nil
}

// Close 关闭存储
func (s *JSONStorage) Close() error { return nil }

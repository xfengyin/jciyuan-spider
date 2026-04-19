// Package storage - 存储模块
package storage

import (
	"strings"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"jciyuan-spider-v2/model"
)

// ============================================================================
// 存储接口
// ============================================================================

// Storage 存储接口
type Storage interface {
	Save(anime *model.AnimeInfo) error
	Load(animeID int64) (*model.AnimeInfo, error)
	SaveStatus(status *model.CrawlStatus) error
	LoadStatus(animeID int64) (*model.CrawlStatus, error)
	Exists(animeID int64) bool
	Close() error
}

// ============================================================================
// JSON存储
// ============================================================================

// JSONStorage JSON文件存储
type JSONStorage struct {
	dir      string
	mu       sync.RWMutex
	statusMu sync.RWMutex
}

// NewJSONStorage 创建JSON存储
func NewJSONStorage(dir string) (*JSONStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	
	return &JSONStorage{
		dir: dir,
	}, nil
}

// Save 保存动漫信息
func (s *JSONStorage) Save(anime *model.AnimeInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 更新anime的UpdatedAt
	anime.UpdatedAt = time.Now()
	
	// 序列化
	data, err := json.MarshalIndent(anime, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	
	// 构建文件路径
	filename := filepath.Join(s.dir, fmt.Sprintf("%d.json", anime.ID))
	
	// 写入文件
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
	
	// 确保目录存在
	statusDir := filepath.Join(s.dir, ".status")
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		return err
	}
	
	filename := filepath.Join(statusDir, fmt.Sprintf("%d.status.json", status.AnimeID))
	
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	
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
func (s *JSONStorage) Close() error {
	return nil
}

// ============================================================================
// 内存存储（用于缓存）
// ============================================================================

// MemoryStorage 内存存储
type MemoryStorage struct {
	data      map[int64]*model.AnimeInfo
	statuses  map[int64]*model.CrawlStatus
	mu        sync.RWMutex
	persistence *JSONStorage
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage(persistence *JSONStorage) *MemoryStorage {
	return &MemoryStorage{
		data:      make(map[int64]*model.AnimeInfo),
		statuses:  make(map[int64]*model.CrawlStatus),
		persistence: persistence,
	}
}

// Save 保存到内存和持久化
func (s *MemoryStorage) Save(anime *model.AnimeInfo) error {
	s.mu.Lock()
	s.data[anime.ID] = anime
	s.mu.Unlock()
	
	if s.persistence != nil {
		return s.persistence.Save(anime)
	}
	return nil
}

// Load 加载
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

// SaveStatus 保存状态
func (s *MemoryStorage) SaveStatus(status *model.CrawlStatus) error {
	s.mu.Lock()
	s.statuses[status.AnimeID] = status
	s.mu.Unlock()
	
	if s.persistence != nil {
		return s.persistence.SaveStatus(status)
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
	
	if s.persistence != nil {
		return s.persistence.LoadStatus(animeID)
	}
	return nil, nil
}

// Exists 检查是否存在
func (s *MemoryStorage) Exists(animeID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if _, ok := s.data[animeID]; ok {
		return true
	}
	
	if s.persistence != nil {
		return s.persistence.Exists(animeID)
	}
	return false
}

// Close 关闭存储
func (s *MemoryStorage) Close() error {
	if s.persistence != nil {
		return s.persistence.Close()
	}
	return nil
}

// ============================================================================
// 工具函数
// ============================================================================

// FormatJSON 格式化JSON
func FormatJSON(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatEpisodeList 格式化剧集列表为文本
func FormatEpisodeList(anime *model.AnimeInfo) string {
	if len(anime.Episodes) == 0 {
		return "暂无剧集"
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s (%d集)\n\n", anime.Title, anime.EpisodeNum))
	
	for _, ep := range anime.Episodes {
		vipTag := ""
		if ep.IsVIP {
			vipTag = " [VIP]"
		}
		sb.WriteString(fmt.Sprintf("[%02d] %s%s\n", ep.Number, ep.Title, vipTag))
	}
	
	return sb.String()
}

// FormatM3U8 生成M3U8播放列表
func FormatM3U8(anime *model.AnimeInfo) string {
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:3\n")
	sb.WriteString(fmt.Sprintf("#PLAYLIST:%s\n", anime.Title))
	sb.WriteString("#GENERATOR:jciyuan-spider\n\n")
	
	for _, ep := range anime.Episodes {
		if ep.M3U8URL != "" {
			sb.WriteString(fmt.Sprintf("#EXTINF:-1,%s\n", ep.Title))
			sb.WriteString(ep.M3U8URL + "\n\n")
		}
	}
	
	return sb.String()
}

// SaveM3U8 保存M3U8文件
func SaveM3U8(anime *model.AnimeInfo, dir string) error {
	filename := filepath.Join(dir, fmt.Sprintf("%s.m3u8", anime.Title))
	content := FormatM3U8(anime)
	return os.WriteFile(filename, []byte(content), 0644)
}

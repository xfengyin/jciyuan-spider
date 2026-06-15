// Package resume - 断点续爬管理
package resume

import (
	"fmt"
	"time"

	"jciyuan-spider-v2/internal/model"
	"jciyuan-spider-v2/internal/storage"
)

// Manager 断点续爬管理器
type Manager struct {
	statusStore storage.StatusStorage
}

// NewManager 创建断点续爬管理器
func NewManager(statusStore storage.StatusStorage) *Manager {
	return &Manager{statusStore: statusStore}
}

// LoadStatus 加载上次爬取状态
func (m *Manager) LoadStatus(animeID int64) (*model.CrawlStatus, error) {
	if m.statusStore == nil {
		return nil, nil
	}
	status, err := m.statusStore.LoadStatus(animeID)
	if err != nil {
		return nil, fmt.Errorf("加载爬取状态失败: %w", err)
	}
	return status, nil
}

// SaveStatus 保存当前爬取状态
func (m *Manager) SaveStatus(status *model.CrawlStatus) error {
	if m.statusStore == nil {
		return nil
	}
	status.LastCrawlAt = time.Now()
	return m.statusStore.SaveStatus(status)
}

// ShouldResume 判断是否需要续爬
func (m *Manager) ShouldResume(animeID int64) bool {
	status, err := m.LoadStatus(animeID)
	if err != nil || status == nil {
		return false
	}
	return status.Status == "paused" || status.Status == "running"
}

// IsCompleted 判断是否已完成
func (m *Manager) IsCompleted(animeID int64) bool {
	status, err := m.LoadStatus(animeID)
	if err != nil || status == nil {
		return false
	}
	return status.Status == "completed"
}

// MarkRunning 标记为运行中
func (m *Manager) MarkRunning(animeID int64) error {
	return m.SaveStatus(&model.CrawlStatus{
		AnimeID: animeID,
		Status:  "running",
	})
}

// MarkCompleted 标记为已完成
func (m *Manager) MarkCompleted(animeID int64) error {
	return m.SaveStatus(&model.CrawlStatus{
		AnimeID: animeID,
		Status:  "completed",
	})
}

// MarkFailed 标记为失败
func (m *Manager) MarkFailed(animeID int64, errMsg string) error {
	return m.SaveStatus(&model.CrawlStatus{
		AnimeID:  animeID,
		Status:   "failed",
		ErrorMsg: errMsg,
	})
}

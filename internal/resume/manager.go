// Package resume 负责断点续爬状态的加载、保存与状态机转换。
package resume

import (
	"context"
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
func (m *Manager) LoadStatus(ctx context.Context, animeID int64) (*model.CrawlStatus, error) {
	if m.statusStore == nil {
		return nil, nil
	}
	status, err := m.statusStore.LoadStatus(ctx, animeID)
	if err != nil {
		return nil, fmt.Errorf("加载爬取状态失败: %w", err)
	}
	return status, nil
}

// SaveStatus 保存当前爬取状态
func (m *Manager) SaveStatus(ctx context.Context, status *model.CrawlStatus) error {
	if m.statusStore == nil {
		return nil
	}
	status.LastCrawlAt = time.Now()
	return m.statusStore.SaveStatus(ctx, status)
}

// ShouldResume 判断是否需要续爬
func (m *Manager) ShouldResume(ctx context.Context, animeID int64) bool {
	status, err := m.LoadStatus(ctx, animeID)
	if err != nil || status == nil {
		return false
	}
	return status.Status == "paused" || status.Status == "running"
}

// IsCompleted 判断是否已完成
func (m *Manager) IsCompleted(ctx context.Context, animeID int64) bool {
	status, err := m.LoadStatus(ctx, animeID)
	if err != nil || status == nil {
		return false
	}
	return status.Status == "completed"
}

// MarkRunning 标记为运行中
func (m *Manager) MarkRunning(ctx context.Context, animeID int64) error {
	return m.SaveStatus(ctx, &model.CrawlStatus{
		AnimeID: animeID,
		Status:  "running",
	})
}

// MarkCompleted 标记为已完成
func (m *Manager) MarkCompleted(ctx context.Context, animeID int64) error {
	return m.SaveStatus(ctx, &model.CrawlStatus{
		AnimeID: animeID,
		Status:  "completed",
	})
}

// MarkFailed 标记为失败
func (m *Manager) MarkFailed(ctx context.Context, animeID int64, errMsg string) error {
	return m.SaveStatus(ctx, &model.CrawlStatus{
		AnimeID:  animeID,
		Status:   "failed",
		ErrorMsg: errMsg,
	})
}

// Package storage 定义数据持久化抽象与 SPI 注册机制，支持 JSON/SQLite/MySQL/S3 等多种后端。
package storage

import (
	"context"
	"fmt"

	"jciyuan-spider/internal/model"
)

// Storage 持久化存储接口，所有存储后端必须实现
type Storage interface {
	// Save 保存动漫信息
	Save(ctx context.Context, anime *model.AnimeInfo) error
	// Load 加载动漫信息
	Load(ctx context.Context, animeID int64) (*model.AnimeInfo, error)
	// Exists 检查是否存在
	Exists(ctx context.Context, animeID int64) (bool, error)
	// SaveBatch 批量保存动漫信息
	SaveBatch(ctx context.Context, animes []*model.AnimeInfo) error
	// Close 关闭存储
	Close() error
}

// StatusStorage 爬取状态存储接口
type StatusStorage interface {
	// SaveStatus 保存爬取状态
	SaveStatus(ctx context.Context, status *model.CrawlStatus) error
	// LoadStatus 加载爬取状态
	LoadStatus(ctx context.Context, animeID int64) (*model.CrawlStatus, error)
}

// Builder 构造器函数签名，用于 SPI 注册
type Builder func(cfg model.StorageConfig) (Storage, error)

// registry 存储已注册的 Storage 构造器
var registry = make(map[string]Builder)

// Register 注册 Storage 实现，name 对应配置 storage.type
func Register(name string, b Builder) {
	registry[name] = b
}

// Build 按配置构建 Storage 实例
func Build(cfg model.StorageConfig) (Storage, error) {
	b, ok := registry[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown storage type: %s", cfg.Type)
	}
	return b(cfg)
}

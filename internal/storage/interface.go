// Package storage - 存储抽象层
package storage

import "jciyuan-spider-v2/internal/model"

// 注：单一实现但保留接口以支持 DI 和未来扩展。
// 当前实现：MemoryStorage / JSONStorage

// Storage 持久化存储接口
type Storage interface {
	// Save 保存动漫信息
	Save(anime *model.AnimeInfo) error
	// Load 加载动漫信息
	Load(animeID int64) (*model.AnimeInfo, error)
	// Exists 检查是否存在
	Exists(animeID int64) bool
	// Close 关闭存储
	Close() error
}

// StatusStorage 爬取状态存储接口
type StatusStorage interface {
	// SaveStatus 保存爬取状态
	SaveStatus(status *model.CrawlStatus) error
	// LoadStatus 加载爬取状态
	LoadStatus(animeID int64) (*model.CrawlStatus, error)
}

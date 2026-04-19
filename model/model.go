// Package model - 数据结构定义
package model

import (
	"time"
)

// ============================================================================
// 动漫信息
// ============================================================================

// AnimeInfo 动漫基本信息
type AnimeInfo struct {
	ID          int64     `json:"id"`           // 动漫ID
	Title       string    `json:"title"`        // 标题
	Alias       string    `json:"alias"`        // 别名
	Year        string    `json:"year"`         // 年份
	Region      string    `json:"region"`       // 地区
	Category    string    `json:"category"`     // 分类
	Tags        []string  `json:"tags"`         // 标签
	CoverImage  string    `json:"cover_image"`  // 封面图
	Description string    `json:"description"`  // 简介
	UpdateDate  string    `json:"update_date"`  // 更新日期
	EpisodeNum  int       `json:"episode_num"`  // 总集数
	UpdateNum   int       `json:"update_num"`   // 更新到第几集
	DoubanURL   string    `json:"douban_url"`   // 豆瓣链接
	DetailURL   string    `json:"detail_url"`   // 详情页链接
	Episodes    []Episode `json:"episodes"`     // 剧集列表
	Status      int       `json:"status"`       // 状态: 0=未爬, 1=爬取中, 2=已完成
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
	UpdatedAt   time.Time `json:"updated_at"`   // 更新时间
}

// ============================================================================
// 剧集信息
// ============================================================================

// Episode 单集信息
type Episode struct {
	ID          int64     `json:"id"`          // 剧集ID
	AnimeID     int64     `json:"anime_id"`    // 所属动漫ID
	Number      int       `json:"number"`       // 集数
	Title       string    `json:"title"`        // 标题
	URL         string    `json:"url"`          // 播放页URL
	M3U8URL     string    `json:"m3u8_url"`    // M3U8播放地址
	PlaySources []string  `json:"play_sources"` // 可用播放源
	IsVIP       bool      `json:"is_vip"`       // 是否VIP
	IsCrawled   bool      `json:"is_crawled"`   // 是否已爬取
	UpdateTime  string    `json:"update_time"`  // 发布时间
	CreatedAt   time.Time `json:"created_at"`  // 创建时间
}

// ============================================================================
// 爬虫状态
// ============================================================================

// CrawlStatus 爬取状态
type CrawlStatus struct {
	AnimeID      int64     `json:"anime_id"`      // 动漫ID
	CurrentPage  int       `json:"current_page"`  // 当前页码
	TotalPages  int       `json:"total_pages"`   // 总页数
	CurrentIndex int       `json:"current_index"` // 当前索引
	TotalCount  int       `json:"total_count"`   // 总数量
	SuccessCount int       `json:"success_count"` // 成功数量
	FailCount   int       `json:"fail_count"`    // 失败数量
	RetryCount  int       `json:"retry_count"`   // 重试次数
	Status      string    `json:"status"`        // 状态: pending, running, paused, completed, failed
	LastCrawlAt time.Time `json:"last_crawl_at"` // 最后爬取时间
	ErrorMsg    string    `json:"error_msg"`     // 错误信息
}

// ============================================================================
// 统计信息
// ============================================================================

// Stats 爬虫统计信息
type Stats struct {
	StartTime     time.Time `json:"start_time"`      // 开始时间
	EndTime       time.Time `json:"end_time"`        // 结束时间
	TotalRequests int64     `json:"total_requests"`   // 总请求数
	SuccessCount  int64     `json:"success_count"`   // 成功数
	FailCount     int64     `json:"fail_count"`      // 失败数
	RetryCount    int64     `json:"retry_count"`     // 重试数
	ParseCount    int64     `json:"parse_count"`     // 解析成功数
	ParseFailCount int64    `json:"parse_fail_count"`// 解析失败数
	Bandwidth     int64     `json:"bandwidth"`       // 带宽（字节）
}

// ============================================================================
// 请求记录
// ============================================================================

// RequestRecord 请求记录
type RequestRecord struct {
	ID        int64     `json:"id"`        // 记录ID
	URL       string    `json:"url"`       // 请求URL
	Method    string    `json:"method"`    // 请求方法
	Status    int       `json:"status"`    // HTTP状态码
	Duration  int64     `json:"duration"`  // 耗时（毫秒）
	RespSize  int64     `json:"resp_size"` // 响应大小
	Retry     int       `json:"retry"`     // 重试次数
	Error     string    `json:"error"`     // 错误信息
	CreatedAt time.Time `json:"created_at"`// 创建时间
}

// ============================================================================
// 配置结构
// ============================================================================

// Config 爬虫配置
type Config struct {
	Spider     SpiderConfig     `yaml:"spider"`
	Anticrawler AnticrawlerConfig `yaml:"anticrawler"`
	Crawl      CrawlConfig      `yaml:"crawl"`
	Storage    StorageConfig    `yaml:"storage"`
	Log        LogConfig        `yaml:"log"`
	Stats      StatsConfig      `yaml:"stats"`
}

// SpiderConfig 爬虫基础配置
type SpiderConfig struct {
	BaseURL    string `yaml:"base_url"`
	Delay      int    `yaml:"delay"`
	Timeout    int    `yaml:"timeout"`
	MaxRetry   int    `yaml:"max_retry"`
	Concurrency int   `yaml:"concurrency"`
}

// AnticrawlerConfig 反爬配置
type AnticrawlerConfig struct {
	EnableProxy bool     `yaml:"enable_proxy"`
	Proxies     []string `yaml:"proxies"`
	RandomUA    bool     `yaml:"random_ua"`
	UserAgents  []string `yaml:"user_agents"`
	KeepCookie  bool     `yaml:"keep_cookie"`
}

// CrawlConfig 爬取配置
type CrawlConfig struct {
	AnimeID     int64  `yaml:"anime_id"`
	Resume      bool   `yaml:"resume"`
	Incremental bool   `yaml:"incremental"`
	MaxEpisodes int    `yaml:"max_episodes"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	OutputDir string `yaml:"output_dir"`
	SaveJSON  bool   `yaml:"save_json"`
	SaveSQLite bool  `yaml:"save_sqlite"`
	DBPath    string `yaml:"db_path"`
	SaveM3U8  bool   `yaml:"save_m3u8"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	Console    bool   `yaml:"console"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
}

// StatsConfig 统计配置
type StatsConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
}

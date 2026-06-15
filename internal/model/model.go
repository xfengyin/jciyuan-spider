// Package model - 数据结构定义
package model

import "time"

// AnimeInfo 动漫基本信息
type AnimeInfo struct {
	ID          int64     `json:"id" yaml:"id"`
	Title       string    `json:"title" yaml:"title"`
	Alias       string    `json:"alias" yaml:"alias"`
	Year        string    `json:"year" yaml:"year"`
	Region      string    `json:"region" yaml:"region"`
	Category    string    `json:"category" yaml:"category"`
	Tags        []string  `json:"tags" yaml:"tags"`
	CoverImage  string    `json:"cover_image" yaml:"cover_image"`
	Description string    `json:"description" yaml:"description"`
	UpdateDate  string    `json:"update_date" yaml:"update_date"`
	EpisodeNum  int       `json:"episode_num" yaml:"episode_num"`
	UpdateNum   int       `json:"update_num" yaml:"update_num"`
	DoubanURL   string    `json:"douban_url" yaml:"douban_url"`
	DetailURL   string    `json:"detail_url" yaml:"detail_url"`
	Episodes    []Episode `json:"episodes" yaml:"episodes"`
	Status      int       `json:"status" yaml:"status"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" yaml:"updated_at"`
}

// Episode 单集信息
type Episode struct {
	ID          int64     `json:"id" yaml:"id"`
	AnimeID     int64     `json:"anime_id" yaml:"anime_id"`
	Number      int       `json:"number" yaml:"number"`
	Title       string    `json:"title" yaml:"title"`
	URL         string    `json:"url" yaml:"url"`
	M3U8URL     string    `json:"m3u8_url" yaml:"m3u8_url"`
	PlaySources []string  `json:"play_sources" yaml:"play_sources"`
	IsVIP       bool      `json:"is_vip" yaml:"is_vip"`
	IsCrawled   bool      `json:"is_crawled" yaml:"is_crawled"`
	UpdateTime  string    `json:"update_time" yaml:"update_time"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
}

// CrawlStatus 爬取状态
type CrawlStatus struct {
	AnimeID      int64     `json:"anime_id" yaml:"anime_id"`
	CurrentPage  int       `json:"current_page" yaml:"current_page"`
	TotalPages   int       `json:"total_pages" yaml:"total_pages"`
	CurrentIndex int       `json:"current_index" yaml:"current_index"`
	TotalCount   int       `json:"total_count" yaml:"total_count"`
	SuccessCount int       `json:"success_count" yaml:"success_count"`
	FailCount    int       `json:"fail_count" yaml:"fail_count"`
	RetryCount   int       `json:"retry_count" yaml:"retry_count"`
	Status       string    `json:"status" yaml:"status"`
	LastCrawlAt  time.Time `json:"last_crawl_at" yaml:"last_crawl_at"`
	ErrorMsg     string    `json:"error_msg" yaml:"error_msg"`
}

// Stats 爬虫统计信息
type Stats struct {
	StartTime      time.Time `json:"start_time" yaml:"start_time"`
	EndTime        time.Time `json:"end_time" yaml:"end_time"`
	TotalRequests  int64     `json:"total_requests" yaml:"total_requests"`
	SuccessCount   int64     `json:"success_count" yaml:"success_count"`
	FailCount      int64     `json:"fail_count" yaml:"fail_count"`
	RetryCount     int64     `json:"retry_count" yaml:"retry_count"`
	ParseCount     int64     `json:"parse_count" yaml:"parse_count"`
	ParseFailCount int64     `json:"parse_fail_count" yaml:"parse_fail_count"`
	Bandwidth      int64     `json:"bandwidth" yaml:"bandwidth"`
}

// Config 爬虫配置
type Config struct {
	Spider      SpiderConfig      `yaml:"spider"`
	Anticrawler AnticrawlerConfig `yaml:"anticrawler"`
	Crawl       CrawlConfig       `yaml:"crawl"`
	Storage     StorageConfig     `yaml:"storage"`
	Log         LogConfig         `yaml:"log"`
	Stats       StatsConfig       `yaml:"stats"`
}

// SpiderConfig 爬虫基础配置
type SpiderConfig struct {
	BaseURL     string `yaml:"base_url"`
	Delay       int    `yaml:"delay"`
	Timeout     int    `yaml:"timeout"`
	MaxRetry    int    `yaml:"max_retry"`
	Concurrency int    `yaml:"concurrency"`
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
	AnimeID     int64 `yaml:"anime_id"`
	Resume      bool  `yaml:"resume"`
	Incremental bool  `yaml:"incremental"`
	MaxEpisodes int   `yaml:"max_episodes"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	OutputDir  string `yaml:"output_dir"`
	SaveJSON   bool   `yaml:"save_json"`
	SaveSQLite bool   `yaml:"save_sqlite"`
	DBPath     string `yaml:"db_path"`
	SaveM3U8   bool   `yaml:"save_m3u8"`
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

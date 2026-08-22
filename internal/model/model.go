// Package model 定义爬虫核心业务实体与配置结构。
package model

import "time"

// AnimeInfo 动漫基本信息
type AnimeInfo struct {
	ID          int64     `json:"id" yaml:"id"`
	Title       string    `json:"title" yaml:"title"`
	Year        string    `json:"year" yaml:"year"`
	Region      string    `json:"region" yaml:"region"`
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
	AnimeID   int64     `json:"anime_id" yaml:"anime_id"`
	Number    int       `json:"number" yaml:"number"`
	Title     string    `json:"title" yaml:"title"`
	URL       string    `json:"url" yaml:"url"`
	M3U8URL   string    `json:"m3u8_url" yaml:"m3u8_url"`
	IsVIP     bool      `json:"is_vip" yaml:"is_vip"`
	IsCrawled bool      `json:"is_crawled" yaml:"is_crawled"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// CrawlStatus 爬取状态，用于断点续爬与进度跟踪
type CrawlStatus struct {
	AnimeID      int64     `json:"anime_id" yaml:"anime_id"`
	Status       string    `json:"status" yaml:"status"` // idle/running/completed/failed/paused
	CurrentIndex int       `json:"current_index" yaml:"current_index"`
	TotalCount   int       `json:"total_count" yaml:"total_count"`
	SuccessCount int       `json:"success_count" yaml:"success_count"`
	FailCount    int       `json:"fail_count" yaml:"fail_count"`
	RetryCount   int       `json:"retry_count" yaml:"retry_count"`
	ErrorMsg     string    `json:"error_msg" yaml:"error_msg"`
	LastCrawlAt  time.Time `json:"last_crawl_at" yaml:"last_crawl_at"`
}

// Stats 爬虫统计信息
type Stats struct {
	StartTime        time.Time `json:"start_time" yaml:"start_time"`
	EndTime          time.Time `json:"end_time" yaml:"end_time"`
	TotalRequests    int64     `json:"total_requests" yaml:"total_requests"`
	SuccessCount     int64     `json:"success_count" yaml:"success_count"`
	FailCount        int64     `json:"fail_count" yaml:"fail_count"`
	RetryCount       int64     `json:"retry_count" yaml:"retry_count"`
	ParseCount       int64     `json:"parse_count" yaml:"parse_count"`
	ParseFailCount   int64     `json:"parse_fail_count" yaml:"parse_fail_count"`
	StorageSaveCount int64     `json:"storage_save_count" yaml:"storage_save_count"`
	StorageSaveFail  int64     `json:"storage_save_fail" yaml:"storage_save_fail"`
	Bandwidth        int64     `json:"bandwidth" yaml:"bandwidth"`
}

// Config 爬虫全局配置，与 config.yaml 一一对应
type Config struct {
	App         AppConfig         `yaml:"app"`
	Spider      SpiderConfig      `yaml:"spider"`
	Anticrawler AnticrawlerConfig `yaml:"anticrawler"`
	Fetcher     FetcherConfig     `yaml:"fetcher"`
	Parser      ParserConfig      `yaml:"parser"`
	Storage     StorageConfig     `yaml:"storage"`
	Crawl       CrawlConfig       `yaml:"crawl"`
	Middlewares []MiddlewareItem  `yaml:"middlewares"`
	Metrics     MetricsConfig     `yaml:"metrics"`
	Log         LogConfig         `yaml:"log"`
}

// AppConfig 应用级配置
type AppConfig struct {
	Name          string `yaml:"name"`
	Mode          string `yaml:"mode"`            // cli | server
	TraceIDHeader string `yaml:"trace_id_header"` // 透传 traceId 的 HTTP Header 名
}

// SpiderConfig 爬虫基础配置
type SpiderConfig struct {
	BaseURL          string `yaml:"base_url"`
	DetailURLPattern string `yaml:"detail_url_pattern"`
	Delay            int    `yaml:"delay"`          // 请求间隔 ms
	Timeout          int    `yaml:"timeout"`        // 请求超时 s
	MaxRetry         int    `yaml:"max_retry"`      // 最大重试次数
	Concurrency      int    `yaml:"concurrency"`    // 全局并发 Worker 数
	QueueSize        int    `yaml:"queue_size"`     // 任务队列长度
}

// AnticrawlerConfig 反爬与浏览器指纹配置
type AnticrawlerConfig struct {
	RandomUA       bool     `yaml:"random_ua"`
	UserAgents     []string `yaml:"user_agents"`
	KeepCookie     bool     `yaml:"keep_cookie"`
	RefererPolicy  string   `yaml:"referer_policy"`
	RobotsTxtCheck bool     `yaml:"robots_txt_check"`
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	FailureThreshold   int           `yaml:"failure_threshold"`    // 触发熔断的连续失败次数
	ErrorRateThreshold float64       `yaml:"error_rate_threshold"` // 触发熔断的错误率阈值（0~1）
	WindowSize         int           `yaml:"window_size"`          // 错误率统计窗口内请求数
	OpenDuration       time.Duration `yaml:"open_duration"`        // Open 态持续时间
	HalfOpenRequests   int           `yaml:"half_open_requests"`   // HalfOpen 态放行探测请求数
}

// FetcherConfig Fetcher 插件配置
type FetcherConfig struct {
	Type           string               `yaml:"type"` // http | selenium | playwright
	HTTP           HTTPFetcherConfig    `yaml:"http"`
	Proxy          ProxyConfig          `yaml:"proxy"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
}

// HTTPFetcherConfig HTTP 传输层配置
type HTTPFetcherConfig struct {
	Timeout         int                 `yaml:"timeout"`   // 请求超时 s
	MaxRetry        int                 `yaml:"max_retry"` // 最大重试次数
	Transport       HTTPTransportConfig `yaml:"transport"`
	FollowRedirects bool                `yaml:"follow_redirects"`
	MaxBodySize     int64               `yaml:"max_body_size"` // 字节
}

// HTTPTransportConfig http.Transport 参数
type HTTPTransportConfig struct {
	MaxIdleConns        int           `yaml:"max_idle_conns"`
	MaxConnsPerHost     int           `yaml:"max_conns_per_host"`
	IdleConnTimeout     time.Duration `yaml:"idle_conn_timeout"`
	DisableKeepAlives   bool          `yaml:"disable_keep_alives"`
	TLSHandshakeTimeout time.Duration `yaml:"tls_handshake_timeout"`
}

// ProxyConfig 代理池配置
type ProxyConfig struct {
	Enable   bool     `yaml:"enable"`
	Strategy string   `yaml:"strategy"` // round_robin | random | least_used
	Proxies  []string `yaml:"proxies"`
}

// ParserConfig Parser 插件配置
type ParserConfig struct {
	Type string           `yaml:"type"` // html | json | ai
	HTML HTMLParserConfig `yaml:"html"`
}

// HTMLParserConfig HTML 解析器配置
type HTMLParserConfig struct {
	Encoding   string            `yaml:"encoding"` // auto | utf-8 | gbk
	Extractors []ExtractorConfig `yaml:"extractors"`
}

// ExtractorConfig 字段提取规则
type ExtractorConfig struct {
	Field       string            `yaml:"field"`
	Selector    SelectorConfig    `yaml:"selector"`
	Processors  []ProcessorConfig `yaml:"processors"`
	Multiple    bool              `yaml:"multiple"`
	Deduplicate bool              `yaml:"deduplicate"`
	SortBy      string            `yaml:"sort_by"` // number | text
}

// SelectorConfig 选择器配置
type SelectorConfig struct {
	Type  string `yaml:"type"` // css | xpath | regex
	Value string `yaml:"value"`
	Attr  string `yaml:"attr,omitempty"` // css/xpath 提取属性名，空则提取文本
}

// ProcessorConfig 字段后处理器配置
type ProcessorConfig struct {
	Type      string            `yaml:"type"`
	Separator string            `yaml:"separator,omitempty"`
	Params    map[string]string `yaml:"params,omitempty"` // 额外参数，如 regex_replace 的 pattern/replacement
}

// StorageConfig Storage 插件配置
type StorageConfig struct {
	Type   string              `yaml:"type"` // json | sqlite | mysql | s3
	JSON   JSONStorageConfig   `yaml:"json"`
	SQLite SQLiteStorageConfig `yaml:"sqlite"`
	MySQL  MySQLStorageConfig  `yaml:"mysql"`
	S3     S3StorageConfig     `yaml:"s3"`
	Memory MemoryStorageConfig `yaml:"memory"`
	Output OutputConfig        `yaml:"output"`
}

// JSONStorageConfig JSON 存储配置
type JSONStorageConfig struct {
	OutputDir string `yaml:"output_dir"`
}

// SQLiteStorageConfig SQLite 存储配置
type SQLiteStorageConfig struct {
	DSN string `yaml:"dsn"`
}

// MySQLStorageConfig MySQL 存储配置
type MySQLStorageConfig struct {
	DSN string `yaml:"dsn"`
}

// S3StorageConfig S3 对象存储配置
type S3StorageConfig struct {
	Endpoint        string `yaml:"endpoint"`
	Region          string `yaml:"region"`
	Bucket          string `yaml:"bucket"`
	KeyPrefix       string `yaml:"key_prefix"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
}

// MemoryStorageConfig 内存缓存装饰器配置
type MemoryStorageConfig struct {
	Enable bool          `yaml:"enable"`
	TTL    time.Duration `yaml:"ttl"`
}

// OutputConfig 输出开关配置
type OutputConfig struct {
	SaveJSON    bool `yaml:"save_json"`
	SaveSQLite  bool `yaml:"save_sqlite"`
	SaveM3U8    bool `yaml:"save_m3u8"`
	SaveRawHTML bool `yaml:"save_raw_html"`
}

// CrawlConfig 爬取任务配置
type CrawlConfig struct {
	AnimeID        int64 `yaml:"anime_id"`
	Resume         bool  `yaml:"resume"`
	Incremental    bool  `yaml:"incremental"`
	MaxEpisodes    int   `yaml:"max_episodes"` // 0=全部
}

// MiddlewareItem 中间件配置项
type MiddlewareItem struct {
	Name   string                 `yaml:"name"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// MetricsConfig Metrics 配置
type MetricsConfig struct {
	Enabled    bool             `yaml:"enabled"`
	Backend    string           `yaml:"backend"` // memory | prometheus
	Prometheus PrometheusConfig `yaml:"prometheus"`
}

// PrometheusConfig Prometheus 暴露配置
type PrometheusConfig struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"` // text | json
	File       string `yaml:"file"`
	Console    bool   `yaml:"console"`
	MaxSize    int    `yaml:"max_size"` // MB
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"` // 天
	Compress   bool   `yaml:"compress"`
}

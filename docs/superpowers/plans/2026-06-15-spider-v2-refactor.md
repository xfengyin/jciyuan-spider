# jciyuan-spider-v2 企业级重构升级计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 jciyuan-spider-v2 从原型级代码重构为符合企业级标准的 Go 爬虫项目，实现接口抽象、依赖注入、可测试、可扩展、高可用。

**Architecture:** 采用接口驱动设计（Interface-Driven Design），核心模块（Fetcher/Parser/Storage/Middleware）全部面向接口编程。引入中间件链模式处理请求生命周期（限流/重试/日志/指标）。配置使用标准 YAML 库替代手写解析。日志包重命名避免与标准库冲突。

**Tech Stack:** Go 1.21+, gopkg.in/yaml.v3, net/http, sync, context, sort.Slice

---

## 文件结构映射

### 新建文件
| 文件 | 职责 |
|------|------|
| `internal/fetcher/interface.go` | Fetcher 接口定义 |
| `internal/fetcher/http_fetcher.go` | HTTP 请求实现 |
| `internal/fetcher/middleware.go` | 请求中间件（限流/重试/日志） |
| `internal/parser/interface.go` | Parser 接口定义 |
| `internal/parser/html_parser.go` | HTML 解析实现 |
| `internal/storage/interface.go` | Storage 接口定义 |
| `internal/storage/json_storage.go` | JSON 存储实现 |
| `internal/storage/memory_storage.go` | 内存缓存存储 |
| `internal/metrics/collector.go` | 指标收集器（独立于 Fetcher） |
| `internal/middleware/ratelimit.go` | 限流中间件 |
| `internal/middleware/retry.go` | 重试中间件 |
| `internal/logger/logger.go` | 日志模块（重命名包） |
| `internal/config/loader.go` | 配置加载（使用 yaml.v3） |
| `internal/spider/spider.go` | Spider 核心逻辑 |
| `internal/resume/manager.go` | 断点续爬管理器 |
| `cmd/spider/main.go` | 入口程序 |

### 删除文件
| 文件 | 原因 |
|------|------|
| `cmd/config/config.go` | 被 `internal/config/loader.go` 替代 |
| `crawler/fetcher.go` | 被 `internal/fetcher/` 替代 |
| `parser/parser.go` | 被 `internal/parser/` 替代 |
| `storage/storage.go` | 被 `internal/storage/` 替代 |
| `log/logger.go` | 被 `internal/logger/logger.go` 替代 |
| `utils/utils.go` | 拆分到各使用模块 |
| `model/model.go` | 拆分到各子包 |

### 修改文件
| 文件 | 变更 |
|------|------|
| `main.go` | 精简为入口，逻辑移至 `internal/spider/` |
| `go.mod` | 添加 `gopkg.in/yaml.v3` 依赖 |
| `config/config.yaml` | 无变更，兼容 |

---

## Task 1: 项目骨架与依赖管理

**Files:**
- Modify: `go.mod`
- Create: `internal/fetcher/interface.go`
- Create: `internal/parser/interface.go`
- Create: `internal/storage/interface.go`

- [ ] **Step 1: 添加 yaml.v3 依赖**

```bash
cd /workspace && go get gopkg.in/yaml.v3
```

- [ ] **Step 2: 创建 Fetcher 接口**

Create `internal/fetcher/interface.go`:

```go
// Package fetcher - HTTP 请求抽象层
package fetcher

import "context"

// Fetcher HTTP 请求器接口
type Fetcher interface {
	// Fetch 获取页面内容
	Fetch(ctx context.Context, url string) ([]byte, error)
}

// FetchOption 请求选项
type FetchOption struct {
	Method  string            // HTTP 方法
	Headers map[string]string // 自定义请求头
	Body    []byte            // 请求体
}
```

- [ ] **Step 3: 创建 Parser 接口**

Create `internal/parser/interface.go`:

```go
// Package parser - HTML 解析抽象层
package parser

import "jciyuan-spider-v2/internal/model"

// Parser HTML 解析器接口
type Parser interface {
	// ParseAnimeDetail 解析动漫详情页
	ParseAnimeDetail(html string) (*model.AnimeInfo, error)
}
```

- [ ] **Step 4: 创建 Storage 接口**

Create `internal/storage/interface.go`:

```go
// Package storage - 存储抽象层
package storage

import "jciyuan-spider-v2/internal/model"

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
```

- [ ] **Step 5: 创建 model 包**

Create `internal/model/model.go`（从 `model/model.go` 迁移，结构不变）:

```go
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
	ID          int64    `json:"id" yaml:"id"`
	AnimeID     int64    `json:"anime_id" yaml:"anime_id"`
	Number      int      `json:"number" yaml:"number"`
	Title       string   `json:"title" yaml:"title"`
	URL         string   `json:"url" yaml:"url"`
	M3U8URL     string   `json:"m3u8_url" yaml:"m3u8_url"`
	PlaySources []string `json:"play_sources" yaml:"play_sources"`
	IsVIP       bool     `json:"is_vip" yaml:"is_vip"`
	IsCrawled   bool     `json:"is_crawled" yaml:"is_crawled"`
	UpdateTime  string   `json:"update_time" yaml:"update_time"`
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
```

- [ ] **Step 6: 验证编译**

```bash
cd /workspace && go build ./internal/...
```

Expected: 编译通过

- [ ] **Step 7: 提交**

```bash
git add go.mod go.sum internal/ && git commit -m "feat: 添加接口定义和 model 包骨架"
```

---

## Task 2: 配置模块重构（使用 yaml.v3）

**Files:**
- Create: `internal/config/loader.go`
- Delete: `cmd/config/config.go`

- [ ] **Step 1: 创建配置加载器**

Create `internal/config/loader.go`:

```go
// Package config - 配置加载模块
package config

import (
	"fmt"
	"os"

	"jciyuan-spider-v2/internal/model"

	"gopkg.in/yaml.v3"
)

// Loader 配置加载器
type Loader struct {
	configPath string
}

// NewLoader 创建配置加载器
func NewLoader(configPath string) *Loader {
	return &Loader{configPath: configPath}
}

// Load 加载配置文件
func (l *Loader) Load() (*model.Config, error) {
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &model.Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	l.applyDefaults(cfg)

	if err := l.validate(cfg); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return cfg, nil
}

// applyDefaults 应用默认值
func (l *Loader) applyDefaults(cfg *model.Config) {
	if cfg.Spider.BaseURL == "" {
		cfg.Spider.BaseURL = "https://www.jciyuan.com"
	}
	if cfg.Spider.Delay == 0 {
		cfg.Spider.Delay = 1000
	}
	if cfg.Spider.Timeout == 0 {
		cfg.Spider.Timeout = 10
	}
	if cfg.Spider.MaxRetry == 0 {
		cfg.Spider.MaxRetry = 3
	}
	if cfg.Spider.Concurrency == 0 {
		cfg.Spider.Concurrency = 3
	}
	if len(cfg.Anticrawler.UserAgents) == 0 {
		cfg.Anticrawler.UserAgents = defaultUserAgents()
	}
	if cfg.Storage.OutputDir == "" {
		cfg.Storage.OutputDir = "./output"
	}
	if cfg.Storage.DBPath == "" {
		cfg.Storage.DBPath = "./data/spider.db"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.MaxSize == 0 {
		cfg.Log.MaxSize = 10
	}
	if cfg.Log.MaxBackups == 0 {
		cfg.Log.MaxBackups = 5
	}
}

// validate 验证配置
func (l *Loader) validate(cfg *model.Config) error {
	if cfg.Spider.BaseURL == "" {
		return fmt.Errorf("base_url 不能为空")
	}
	if cfg.Spider.Delay < 100 {
		return fmt.Errorf("delay 不能小于 100 毫秒")
	}
	if cfg.Spider.Timeout < 1 {
		return fmt.Errorf("timeout 不能小于 1 秒")
	}
	if cfg.Spider.MaxRetry < 0 {
		return fmt.Errorf("max_retry 不能为负数")
	}
	if cfg.Spider.Concurrency < 1 {
		return fmt.Errorf("concurrency 不能小于 1")
	}
	return nil
}

// defaultUserAgents 默认 UA 列表
func defaultUserAgents() []string {
	return []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// LoadFromEnv 从环境变量加载配置覆盖
func LoadFromEnv(cfg *model.Config) {
	if baseURL := os.Getenv("JCIYUAN_BASE_URL"); baseURL != "" {
		cfg.Spider.BaseURL = baseURL
	}
	if delay := os.Getenv("JCIYUAN_DELAY"); delay != "" {
		var d int
		fmt.Sscanf(delay, "%d", &d)
		if d > 0 {
			cfg.Spider.Delay = d
		}
	}
	if ua := os.Getenv("JCIYUAN_USER_AGENT"); ua != "" {
		cfg.Anticrawler.UserAgents = []string{ua}
	}
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /workspace && go build ./internal/config/...
```

- [ ] **Step 3: 删除旧配置模块**

```bash
rm -rf /workspace/cmd/config/
```

- [ ] **Step 4: 提交**

```bash
git add internal/config/ && git rm -r cmd/config/ && git commit -m "refactor: 使用 yaml.v3 替代手写 YAML 解析器"
```

---

## Task 3: 日志模块重构

**Files:**
- Create: `internal/logger/logger.go`
- Delete: `log/logger.go`

- [ ] **Step 1: 创建新日志模块**

Create `internal/logger/logger.go`（从 `log/logger.go` 迁移，包名改为 `logger`，逻辑不变）:

```go
// Package logger - 日志模块
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level 日志级别
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (l Level) Color() string {
	switch l {
	case DEBUG:
		return "\033[36m"
	case INFO:
		return "\033[32m"
	case WARN:
		return "\033[33m"
	case ERROR:
		return "\033[31m"
	default:
		return "\033[0m"
	}
}

const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorGray   = "\033[90m"
)

// Logger 日志记录器
type Logger struct {
	mu         sync.Mutex
	level      Level
	output     io.Writer
	file       *os.File
	module     string
	timeFormat string
}

// NewLogger 创建日志记录器
func NewLogger(module string) *Logger {
	return &Logger{
		level:      INFO,
		output:     os.Stdout,
		module:     module,
		timeFormat: "2006-01-02 15:04:05",
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		l.level = DEBUG
	case "INFO":
		l.level = INFO
	case "WARN":
		l.level = WARN
	case "ERROR":
		l.level = ERROR
	default:
		l.level = INFO
	}
}

// SetFile 设置日志文件输出
func (l *Logger) SetFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	l.file = f
	return nil
}

// Close 关闭日志文件
func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format(l.timeFormat)
	levelStr := level.String()

	_, file, line, ok := runtime.Caller(2)
	caller := ""
	if ok {
		parts := strings.Split(file, "/")
		if len(parts) > 2 {
			file = filepath.Join(parts[len(parts)-2:]...)
		}
		caller = fmt.Sprintf("%s:%d", file, line)
	}

	message := fmt.Sprintf(format, args...)

	var lineStr string
	if caller != "" {
		lineStr = fmt.Sprintf("[%s] [%s] [%s] %s %s\n",
			timestamp, levelStr, l.module, caller, message)
	} else {
		lineStr = fmt.Sprintf("[%s] [%s] [%s] %s\n",
			timestamp, levelStr, l.module, message)
	}

	if l.output == os.Stdout || l.output == os.Stderr {
		colored := fmt.Sprintf("%s%s%s%s [%s]%s [%s%s%s]%s %s%s\n",
			ColorGray, timestamp, ColorReset,
			level.Color(), levelStr, ColorReset,
			ColorBold, l.module, ColorReset,
			caller,
			ColorReset, message)
		fmt.Fprint(l.output, colored)
	} else {
		fmt.Fprint(l.output, lineStr)
	}

	if l.file != nil {
		fmt.Fprint(l.file, lineStr)
	}
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal 致命错误日志
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
	os.Exit(1)
}

// ============================================================================
// 全局日志器
// ============================================================================

var defaultLogger = NewLogger("main")

// SetLevel 设置全局日志级别
func SetLevel(level string) { defaultLogger.SetLevel(level) }

// SetFile 设置全局日志文件
func SetFile(path string) error { return defaultLogger.SetFile(path) }

// Close 关闭全局日志
func Close() { defaultLogger.Close() }

// Debug 全局调试日志
func Debug(format string, args ...interface{}) { defaultLogger.Debug(format, args...) }

// Info 全局信息日志
func Info(format string, args ...interface{}) { defaultLogger.Info(format, args...) }

// Warn 全局警告日志
func Warn(format string, args ...interface{}) { defaultLogger.Warn(format, args...) }

// Error 全局错误日志
func Error(format string, args ...interface{}) { defaultLogger.Error(format, args...) }

// Fatal 全局致命错误
func Fatal(format string, args ...interface{}) { defaultLogger.Fatal(format, args...) }

// GetLogger 获取模块日志器
func GetLogger(module string) *Logger { return NewLogger(module) }
```

- [ ] **Step 2: 删除旧日志模块**

```bash
rm -rf /workspace/log/
```

- [ ] **Step 3: 验证编译**

```bash
cd /workspace && go build ./internal/logger/...
```

- [ ] **Step 4: 提交**

```bash
git add internal/logger/ && git rm -r log/ && git commit -m "refactor: 日志包重命名为 logger，避免与标准库冲突"
```

---

## Task 4: 指标收集器独立化

**Files:**
- Create: `internal/metrics/collector.go`

- [ ] **Step 1: 创建独立指标收集器**

Create `internal/metrics/collector.go`:

```go
// Package metrics - 指标收集模块
package metrics

import (
	"sync"
	"time"

	"jciyuan-spider-v2/internal/model"
)

// Collector 指标收集器
type Collector struct {
	mu           sync.RWMutex
	totalReq     int64
	successCount int64
	failCount    int64
	retryCount   int64
	parseCount   int64
	parseFail    int64
	totalBytes   int64
	startTime    time.Time
}

// NewCollector 创建指标收集器
func NewCollector() *Collector {
	return &Collector{
		startTime: time.Now(),
	}
}

// IncrSuccess 成功计数
func (c *Collector) IncrSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.successCount++
	c.totalReq++
}

// IncrFail 失败计数
func (c *Collector) IncrFail() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failCount++
	c.totalReq++
}

// IncrRetry 重试计数
func (c *Collector) IncrRetry() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retryCount++
}

// IncrParse 解析成功计数
func (c *Collector) IncrParse() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.parseCount++
}

// IncrParseFail 解析失败计数
func (c *Collector) IncrParseFail() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.parseFail++
}

// AddBytes 记录流量
func (c *Collector) AddBytes(n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalBytes += n
}

// GetStats 获取统计快照
func (c *Collector) GetStats() model.Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return model.Stats{
		StartTime:      c.startTime,
		TotalRequests:  c.totalReq,
		SuccessCount:   c.successCount,
		FailCount:      c.failCount,
		RetryCount:     c.retryCount,
		ParseCount:     c.parseCount,
		ParseFailCount: c.parseFail,
		Bandwidth:      c.totalBytes,
	}
}

// Reset 重置统计
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalReq = 0
	c.successCount = 0
	c.failCount = 0
	c.retryCount = 0
	c.parseCount = 0
	c.parseFail = 0
	c.totalBytes = 0
	c.startTime = time.Now()
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /workspace && go build ./internal/metrics/...
```

- [ ] **Step 3: 提交**

```bash
git add internal/metrics/ && git commit -m "feat: 独立指标收集器，解耦与 Fetcher 的依赖"
```

---

## Task 5: Fetcher 模块重构（接口实现 + 中间件）

**Files:**
- Create: `internal/fetcher/http_fetcher.go`
- Create: `internal/fetcher/middleware.go`
- Delete: `crawler/fetcher.go`

- [ ] **Step 1: 创建 HTTP Fetcher 实现**

Create `internal/fetcher/http_fetcher.go`:

```go
package fetcher

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"jciyuan-spider-v2/internal/metrics"
	"jciyuan-spider-v2/internal/model"
)

// HTTPFetcher HTTP 请求器实现
type HTTPFetcher struct {
	client     *http.Client
	config     *model.Config
	userAgents []string
	metrics    *metrics.Collector
	middleware []Middleware
}

// NewHTTPFetcher 创建 HTTP 请求器
func NewHTTPFetcher(cfg *model.Config, m *metrics.Collector) (*HTTPFetcher, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("创建 cookie jar 失败: %w", err)
	}

	client := &http.Client{
		Timeout: time.Duration(cfg.Spider.Timeout) * time.Second,
		Jar:     jar,
	}

	f := &HTTPFetcher{
		client:     client,
		config:     cfg,
		userAgents: cfg.Anticrawler.UserAgents,
		metrics:    m,
	}

	// 注册默认中间件
	f.middleware = []Middleware{
		RateLimitMiddleware(cfg.Spider.Delay),
		RetryMiddleware(cfg.Spider.MaxRetry, m),
	}

	return f, nil
}

// Fetch 获取页面内容
func (f *HTTPFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	f.setHeaders(req)

	// 执行中间件链
	handler := f.executeRequest
	for i := len(f.middleware) - 1; i >= 0; i-- {
		handler = f.middleware[i](handler)
	}

	return handler(ctx, req)
}

// executeRequest 执行实际 HTTP 请求
func (f *HTTPFetcher) executeRequest(ctx context.Context, req *http.Request) ([]byte, error) {
	resp, err := f.client.Do(req)
	if err != nil {
		f.metrics.IncrFail()
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	f.metrics.AddBytes(resp.ContentLength)

	if resp.StatusCode == http.StatusForbidden {
		f.metrics.IncrFail()
		return nil, &BlockedError{URL: req.URL.String(), StatusCode: 403, Message: "访问被禁止"}
	}

	if resp.StatusCode != http.StatusOK {
		f.metrics.IncrFail()
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, req.URL.String())
	}

	reader := resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip 解压失败: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	body, err := io.ReadAll(io.LimitReader(reader, 50*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if isCaptchaPage(body) {
		return nil, &BlockedError{URL: req.URL.String(), StatusCode: 403, Message: "验证码页面"}
	}

	f.metrics.IncrSuccess()
	return body, nil
}

// setHeaders 设置请求头
func (f *HTTPFetcher) setHeaders(req *http.Request) {
	if f.config.Anticrawler.RandomUA && len(f.userAgents) > 0 {
		ua := f.userAgents[rand.Intn(len(f.userAgents))]
		req.Header.Set("User-Agent", ua)
	} else if len(f.userAgents) > 0 {
		req.Header.Set("User-Agent", f.userAgents[0])
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Referer", f.config.Spider.BaseURL+"/")
	req.Header.Set("Origin", f.config.Spider.BaseURL)
}

// BlockedError 访问被阻止错误
type BlockedError struct {
	URL        string
	StatusCode int
	Message    string
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("Blocked: %s (HTTP %d) - %s", e.URL, e.StatusCode, e.Message)
}

// IsBlocked 判断是否为阻止错误
func IsBlocked(err error) bool {
	_, ok := err.(*BlockedError)
	return ok
}

// isCaptchaPage 检测验证码页面
func isCaptchaPage(body []byte) bool {
	content := strings.ToLower(string(body))
	keywords := []string{"验证码", " captcha", "安全验证", "请输入验证码"}
	for _, kw := range keywords {
		if strings.Contains(content, kw) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: 创建中间件**

Create `internal/fetcher/middleware.go`:

```go
package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"jciyuan-spider-v2/internal/metrics"
)

// Handler 请求处理函数
type Handler func(ctx context.Context, req *http.Request) ([]byte, error)

// Middleware 中间件函数
type Middleware func(next Handler) Handler

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(delayMs int) Middleware {
	var lastRequest time.Time
	return func(next Handler) Handler {
		return func(ctx context.Context, req *http.Request) ([]byte, error) {
			delay := time.Duration(delayMs) * time.Millisecond
			if elapsed := time.Since(lastRequest); elapsed < delay {
				select {
				case <-time.After(delay - elapsed):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			lastRequest = time.Now()
			return next(ctx, req)
		}
	}
}

// RetryMiddleware 重试中间件
func RetryMiddleware(maxRetry int, m *metrics.Collector) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *http.Request) ([]byte, error) {
			var lastErr error
			for attempt := 0; attempt <= maxRetry; attempt++ {
				if attempt > 0 {
					m.IncrRetry()
					backoff := time.Duration(attempt*500) * time.Millisecond
					select {
					case <-time.After(backoff):
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}

				var body []byte
				body, lastErr = next(ctx, req)
				if lastErr == nil {
					return body, nil
				}

				// 阻止类错误不重试
				if IsBlocked(lastErr) {
					return nil, lastErr
				}
			}
			return nil, fmt.Errorf("重试 %d 次后仍失败: %w", maxRetry, lastErr)
		}
	}
}
```

- [ ] **Step 3: 删除旧 crawler 包**

```bash
rm -rf /workspace/crawler/
```

- [ ] **Step 4: 验证编译**

```bash
cd /workspace && go build ./internal/fetcher/...
```

- [ ] **Step 5: 提交**

```bash
git add internal/fetcher/ && git rm -r crawler/ && git commit -m "refactor: Fetcher 接口化 + 中间件链模式"
```

---

## Task 6: Parser 模块重构

**Files:**
- Create: `internal/parser/html_parser.go`
- Create: `internal/parser/utils.go`
- Delete: `parser/parser.go`
- Delete: `utils/utils.go`

- [ ] **Step 1: 创建 Parser 工具函数**

Create `internal/parser/utils.go`（从 `utils/utils.go` 提取解析相关函数）:

```go
package parser

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// extractString 正则提取字符串
func extractString(pattern, input string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractInt 正则提取整数
func extractInt(pattern, input string) int {
	s := extractString(pattern, input)
	var num int
	fmt.Sscanf(s, "%d", &num)
	return num
}

// cleanText 清理文本
func cleanText(s string) string {
	s = removeHTMLTags(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.TrimSpace(s)
	spaceRe := regexp.MustCompile(`\s+`)
	s = spaceRe.ReplaceAllString(s, " ")
	return s
}

// removeHTMLTags 移除 HTML 标签
func removeHTMLTags(s string) string {
	tagRe := regexp.MustCompile(`<[^>]+>`)
	s = tagRe.ReplaceAllString(s, "")
	return decodeHTMLEntities(s)
}

// decodeHTMLEntities 解码 HTML 实体
func decodeHTMLEntities(s string) string {
	entities := map[string]string{
		"&nbsp;": " ", "&amp;": "&", "&lt;": "<", "&gt;": ">",
		"&quot;": "\"", "&apos;": "'", "&#39;": "'",
		"&mdash;": "—", "&ndash;": "–", "&hellip;": "…",
	}
	for entity, char := range entities {
		s = strings.ReplaceAll(s, entity, char)
	}
	return s
}

// uniqueStrings 去重字符串切片
func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	result := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// isChinese 判断是否包含中文
func isChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Scripts["Han"], r) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: 创建 HTML Parser 实现**

Create `internal/parser/html_parser.go`:

```go
package parser

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"jciyuan-spider-v2/internal/model"
)

// HTMLParser HTML 解析器实现
type HTMLParser struct {
	baseURL string
}

// NewHTMLParser 创建 HTML 解析器
func NewHTMLParser(baseURL string) *HTMLParser {
	return &HTMLParser{baseURL: baseURL}
}

// ParseAnimeDetail 解析动漫详情页
func (p *HTMLParser) ParseAnimeDetail(html string) (*model.AnimeInfo, error) {
	anime := &model.AnimeInfo{
		Tags:      []string{},
		Episodes:  []model.Episode{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	p.extractBasicInfo(html, anime)
	anime.Tags = p.extractTags(html)
	anime.Episodes = p.extractEpisodes(html, anime.ID)
	anime.EpisodeNum = len(anime.Episodes)

	return anime, nil
}

// extractBasicInfo 提取基本信息
func (p *HTMLParser) extractBasicInfo(html string, anime *model.AnimeInfo) {
	anime.Title = p.extractTitle(html)
	anime.Year = extractString(`(19|20)\d{2}`, html)
	anime.Region = p.extractRegion(html)
	anime.Description = p.extractDescription(html)
	anime.UpdateDate = extractString(`(\d{4}-\d{2}-\d{2})`, html)
	anime.CoverImage = p.extractCoverImage(html)
	anime.DoubanURL = extractString(`href=["']([^"']*douban[^"']*)["']`, html)
	anime.UpdateNum = extractInt(`更新至(\d+)集`, html)
}

// extractTitle 提取标题
func (p *HTMLParser) extractTitle(html string) string {
	title := extractString(`<title[^>]*>([^<]+)</title>`, html)
	if title != "" {
		title = cleanText(title)
		if idx := strings.Index(title, "_"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
		if idx := strings.Index(title, "-"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
		return title
	}

	title = extractString(`<h1[^>]*>([^<]+)</h1>`, html)
	if title != "" {
		return cleanText(title)
	}
	return "未知标题"
}

// extractRegion 提取地区
func (p *HTMLParser) extractRegion(html string) string {
	regions := []string{"大陆", "日本", "美国", "韩国", "港台", "欧美"}
	for _, region := range regions {
		if strings.Contains(html, region) {
			return region
		}
	}
	return "未知"
}

// extractDescription 提取简介
func (p *HTMLParser) extractDescription(html string) string {
	desc := extractString(`meta\s+name=["']description["']\s+content=["']([^"']+)["']`, html)
	if desc != "" {
		return cleanText(desc)
	}
	desc = extractString(`剧情[:：]([^<]{50,500})`, html)
	if desc != "" {
		return cleanText(desc)
	}
	return ""
}

// extractCoverImage 提取封面图
func (p *HTMLParser) extractCoverImage(html string) string {
	cover := extractString(`src=["']([^"']*doubaocdn[^"']*)["']`, html)
	if cover != "" {
		return cover
	}
	cover = extractString(`data-original=["']([^"']+)["']`, html)
	if cover != "" {
		return cover
	}
	return extractString(`og:image["']\s+content=["']([^"']+)["']`, html)
}

// extractTags 提取标签
func (p *HTMLParser) extractTags(html string) []string {
	tagList := []string{
		"热血", "冒险", "搞笑", "战斗", "奇幻", "科幻",
		"校园", "恋爱", "治愈", "运动", "音乐", "美食",
		"悬疑", "推理", "惊悚", "恐怖", "后宫", "机战",
		"战争", "历史", "励志", "社畜", "爆笑", "国产动漫",
		"日本动漫", "美国动漫", "韩国动漫",
	}

	content := strings.ToLower(html)
	tags := make([]string, 0)
	for _, tag := range tagList {
		if strings.Contains(content, strings.ToLower(tag)) {
			tags = append(tags, tag)
		}
	}
	return uniqueStrings(tags)
}

// extractEpisodes 提取剧集列表
func (p *HTMLParser) extractEpisodes(html string, animeID int64) []model.Episode {
	pattern := regexp.MustCompile(`/acgplay/(\d+)-(\d+)-(\d+)\.html`)
	matches := pattern.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool, len(matches))
	episodes := make([]model.Episode, 0, len(matches))

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		key := match[0]
		if seen[key] {
			continue
		}
		seen[key] = true

		episodeNum := parseInt(match[3])
		title := fmt.Sprintf("第%02d集", episodeNum)
		episodeURL := p.baseURL + match[0]
		isVIP := strings.Contains(html, "vip") || strings.Contains(html, "VIP")

		episodes = append(episodes, model.Episode{
			AnimeID:   animeID,
			Number:    episodeNum,
			Title:     title,
			URL:       episodeURL,
			IsVIP:     isVIP,
			IsCrawled: false,
			CreatedAt: time.Now(),
		})
	}

	// 使用 sort.Slice 替代冒泡排序
	sort.Slice(episodes, func(i, j int) bool {
		return episodes[i].Number < episodes[j].Number
	})

	// 按集数去重
	deduped := make([]model.Episode, 0, len(episodes))
	numSeen := make(map[int]bool, len(episodes))
	for _, ep := range episodes {
		if !numSeen[ep.Number] {
			numSeen[ep.Number] = true
			deduped = append(deduped, ep)
		}
	}

	return deduped
}

// parseInt 安全解析整数
func parseInt(s string) int {
	var result int
	fmt.Sscanf(strings.TrimSpace(s), "%d", &result)
	return result
}
```

- [ ] **Step 3: 删除旧 parser 和 utils 包**

```bash
rm -rf /workspace/parser/ /workspace/utils/
```

- [ ] **Step 4: 验证编译**

```bash
cd /workspace && go build ./internal/parser/...
```

- [ ] **Step 5: 提交**

```bash
git add internal/parser/ && git rm -r parser/ utils/ && git commit -m "refactor: Parser 接口化 + sort.Slice 替代冒泡排序"
```

---

## Task 7: Storage 模块重构

**Files:**
- Create: `internal/storage/json_storage.go`
- Create: `internal/storage/memory_storage.go`
- Create: `internal/storage/format.go`
- Delete: `storage/storage.go`

- [ ] **Step 1: 创建 JSON 存储**

Create `internal/storage/json_storage.go`:

```go
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
```

- [ ] **Step 2: 创建内存缓存存储**

Create `internal/storage/memory_storage.go`:

```go
package storage

import (
	"sync"

	"jciyuan-spider-v2/internal/model"
)

// MemoryStorage 内存缓存存储（装饰器模式，包装持久化存储）
type MemoryStorage struct {
	data        map[int64]*model.AnimeInfo
	statuses    map[int64]*model.CrawlStatus
	mu          sync.RWMutex
	persistence Storage
	statusStore StatusStorage
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage(persistence Storage, statusStore StatusStorage) *MemoryStorage {
	return &MemoryStorage{
		data:        make(map[int64]*model.AnimeInfo),
		statuses:    make(map[int64]*model.CrawlStatus),
		persistence: persistence,
		statusStore: statusStore,
	}
}

// Save 保存到内存和持久化层
func (s *MemoryStorage) Save(anime *model.AnimeInfo) error {
	s.mu.Lock()
	s.data[anime.ID] = anime
	s.mu.Unlock()

	if s.persistence != nil {
		return s.persistence.Save(anime)
	}
	return nil
}

// Load 优先从内存加载
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

// Exists 检查是否存在
func (s *MemoryStorage) Exists(animeID int64) bool {
	s.mu.RLock()
	if _, ok := s.data[animeID]; ok {
		s.mu.RUnlock()
		return true
	}
	s.mu.RUnlock()

	if s.persistence != nil {
		return s.persistence.Exists(animeID)
	}
	return false
}

// SaveStatus 保存状态
func (s *MemoryStorage) SaveStatus(status *model.CrawlStatus) error {
	s.mu.Lock()
	s.statuses[status.AnimeID] = status
	s.mu.Unlock()

	if s.statusStore != nil {
		return s.statusStore.SaveStatus(status)
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

	if s.statusStore != nil {
		return s.statusStore.LoadStatus(animeID)
	}
	return nil, nil
}

// Close 关闭存储
func (s *MemoryStorage) Close() error {
	if s.persistence != nil {
		return s.persistence.Close()
	}
	return nil
}
```

- [ ] **Step 3: 创建格式化工具**

Create `internal/storage/format.go`:

```go
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jciyuan-spider-v2/internal/model"
)

// FormatJSON 格式化为 JSON 字符串
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

// FormatM3U8 生成 M3U8 播放列表
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

// SaveM3U8 保存 M3U8 文件
func SaveM3U8(anime *model.AnimeInfo, dir string) error {
	filename := filepath.Join(dir, fmt.Sprintf("%s.m3u8", anime.Title))
	content := FormatM3U8(anime)
	return os.WriteFile(filename, []byte(content), 0644)
}
```

- [ ] **Step 4: 删除旧 storage 包**

```bash
rm -rf /workspace/storage/
```

- [ ] **Step 5: 验证编译**

```bash
cd /workspace && go build ./internal/storage/...
```

- [ ] **Step 6: 提交**

```bash
git add internal/storage/ && git rm -r storage/ && git commit -m "refactor: Storage 接口化 + 装饰器模式缓存层"
```

---

## Task 8: 断点续爬管理器

**Files:**
- Create: `internal/resume/manager.go`

- [ ] **Step 1: 创建断点续爬管理器**

Create `internal/resume/manager.go`:

```go
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
```

- [ ] **Step 2: 验证编译**

```bash
cd /workspace && go build ./internal/resume/...
```

- [ ] **Step 3: 提交**

```bash
git add internal/resume/ && git commit -m "feat: 断点续爬管理器，支持状态持久化"
```

---

## Task 9: Spider 核心逻辑重构

**Files:**
- Create: `internal/spider/spider.go`
- Modify: `main.go`

- [ ] **Step 1: 创建 Spider 核心**

Create `internal/spider/spider.go`:

```go
// Package spider - 爬虫核心逻辑
package spider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/logger"
	"jciyuan-spider-v2/internal/metrics"
	"jciyuan-spider-v2/internal/model"
	"jciyuan-spider-v2/internal/parser"
	"jciyuan-spider-v2/internal/resume"
	"jciyuan-spider-v2/internal/storage"
)

// Spider 爬虫实例
type Spider struct {
	config    *model.Config
	fetcher   fetcher.Fetcher
	parser    parser.Parser
	storage   storage.Storage
	metrics   *metrics.Collector
	resume    *resume.Manager
}

// NewSpider 创建爬虫
func NewSpider(cfg *model.Config) (*Spider, error) {
	// 创建指标收集器
	m := metrics.NewCollector()

	// 创建请求器
	f, err := fetcher.NewHTTPFetcher(cfg, m)
	if err != nil {
		return nil, fmt.Errorf("创建请求器失败: %w", err)
	}

	// 创建解析器
	p := parser.NewHTMLParser(cfg.Spider.BaseURL)

	// 创建存储
	jsonStore, err := storage.NewJSONStorage(cfg.Storage.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("创建存储失败: %w", err)
	}
	memStore := storage.NewMemoryStorage(jsonStore, jsonStore)

	// 创建断点续爬管理器
	resumeMgr := resume.NewManager(jsonStore)

	return &Spider{
		config:  cfg,
		fetcher: f,
		parser:  p,
		storage: memStore,
		metrics: m,
		resume:  resumeMgr,
	}, nil
}

// Run 运行爬虫
func (s *Spider) Run(ctx context.Context) error {
	animeID := s.config.Crawl.AnimeID
	logger.Info("开始爬取动漫 ID: %d", animeID)

	// 断点续爬检查
	if s.config.Crawl.Resume && s.resume.IsCompleted(animeID) {
		logger.Info("动漫 %d 已完成爬取，跳过", animeID)
		return nil
	}

	// 标记运行中
	if s.config.Crawl.Resume {
		if err := s.resume.MarkRunning(animeID); err != nil {
			logger.Warn("标记运行状态失败: %v", err)
		}
	}

	// 构建 URL
	url := s.buildURL(animeID)

	// 获取详情页
	html, err := s.fetcher.Fetch(ctx, url)
	if err != nil {
		s.markFailed(animeID, err)
		return fmt.Errorf("获取页面失败: %w", err)
	}

	// 解析动漫信息
	anime, err := s.parser.ParseAnimeDetail(string(html))
	if err != nil {
		s.metrics.IncrParseFail()
		s.markFailed(animeID, err)
		return fmt.Errorf("解析失败: %w", err)
	}

	anime.ID = animeID
	anime.DetailURL = url

	s.metrics.IncrParse()
	logger.Info("解析成功: %s, 共 %d 集", anime.Title, anime.EpisodeNum)

	// 增量更新检查
	if s.config.Crawl.Incremental {
		existing, _ := s.storage.Load(animeID)
		if existing != nil {
			anime = s.mergeEpisodes(existing, anime)
		}
	}

	// 保存结果
	if err := s.storage.Save(anime); err != nil {
		logger.Error("保存失败: %v", err)
	}

	// 显示预览
	s.showPreview(anime)

	// 标记完成
	if s.config.Crawl.Resume {
		if err := s.resume.MarkCompleted(animeID); err != nil {
			logger.Warn("标记完成状态失败: %v", err)
		}
	}

	return nil
}

// GetStats 获取统计信息
func (s *Spider) GetStats() model.Stats {
	return s.metrics.GetStats()
}

// Close 关闭资源
func (s *Spider) Close() error {
	return s.storage.Close()
}

// buildURL 构建 URL
func (s *Spider) buildURL(animeID int64) string {
	return fmt.Sprintf("%s/acgdetail/%d.html", s.config.Spider.BaseURL, animeID)
}

// markFailed 标记失败
func (s *Spider) markFailed(animeID int64, err error) {
	if s.config.Crawl.Resume {
		if markErr := s.resume.MarkFailed(animeID, err.Error()); markErr != nil {
			logger.Warn("标记失败状态失败: %v", markErr)
		}
	}
}

// mergeEpisodes 增量合并剧集
func (s *Spider) mergeEpisodes(existing, latest *model.AnimeInfo) *model.AnimeInfo {
	existingMap := make(map[int]bool, len(existing.Episodes))
	for _, ep := range existing.Episodes {
		existingMap[ep.Number] = true
	}

	for _, ep := range latest.Episodes {
		if !existingMap[ep.Number] {
			existing.Episodes = append(existing.Episodes, ep)
		}
	}

	existing.EpisodeNum = len(existing.Episodes)
	existing.UpdatedAt = time.Now()
	if latest.Title != "" {
		existing.Title = latest.Title
	}
	if latest.UpdateDate != "" {
		existing.UpdateDate = latest.UpdateDate
	}
	return existing
}

// showPreview 显示动漫预览
func (s *Spider) showPreview(anime *model.AnimeInfo) {
	logger.Info("标题: %s", anime.Title)
	if anime.Year != "" {
		logger.Info("年份: %s", anime.Year)
	}
	if anime.Region != "" {
		logger.Info("地区: %s", anime.Region)
	}
	if len(anime.Tags) > 0 {
		logger.Info("标签: %s", strings.Join(anime.Tags, ", "))
	}
	logger.Info("集数: %d 集", anime.EpisodeNum)

	if len(anime.Episodes) > 0 {
		logger.Info("剧集列表 (前5集):")
		for i := 0; i < 5 && i < len(anime.Episodes); i++ {
			ep := anime.Episodes[i]
			vipTag := ""
			if ep.IsVIP {
				vipTag = " [VIP]"
			}
			logger.Info("  [%02d] %s%s", ep.Number, ep.Title, vipTag)
		}
		if anime.EpisodeNum > 5 {
			logger.Info("  ... 共 %d 集", anime.EpisodeNum)
		}
	}
}
```

- [ ] **Step 2: 重写 main.go**

Rewrite `main.go`:

```go
// jciyuan-spider-v2 - 企业级动漫爬虫
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jciyuan-spider-v2/internal/config"
	"jciyuan-spider-v2/internal/logger"
	"jciyuan-spider-v2/internal/spider"
)

var (
	configPath   = flag.String("config", "config/config.yaml", "配置文件路径")
	animeIDFlag  = flag.Int64("id", 37439, "动漫ID")
	urlFlag      = flag.String("url", "", "直接指定URL")
	delayFlag    = flag.Int("delay", 0, "请求间隔(毫秒)")
	outputFlag   = flag.String("output", "", "输出目录")
	resumeFlag   = flag.Bool("resume", false, "启用断点续爬")
	incremental  = flag.Bool("incremental", false, "增量更新")
	statsFlag    = flag.Bool("stats", true, "显示统计信息")
	debugFlag    = flag.Bool("debug", false, "调试模式")
)

func main() {
	flag.Parse()

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}

	applyFlags(cfg)
	initLogger(cfg)

	// 信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("收到中断信号，正在优雅退出...")
		cancel()
	}()

	// 创建爬虫
	s, err := spider.NewSpider(cfg)
	if err != nil {
		logger.Fatal("创建爬虫失败: %v", err)
	}
	defer s.Close()

	showBanner(cfg)

	// 执行爬取
	startTime := time.Now()
	if err := s.Run(ctx); err != nil {
		logger.Error("爬取失败: %v", err)
		os.Exit(1)
	}

	// 显示统计
	if *statsFlag {
		showStats(s, startTime)
	}

	logger.Info("爬取完成!")
}

func loadConfig() (*config.Loader, error) {
	loader := config.NewLoader(*configPath)
	// 先加载配置验证是否可用
	_, err := loader.Load()
	if err != nil {
		return nil, err
	}
	return loader, nil
}

func applyFlags(cfg *spider.Config) {
	// 这里需要重新思考，因为 config.Loader 返回的是 model.Config
	// 暂时留空，在下一步修正
}

func initLogger(cfg interface{}) {
	// 修正后实现
}

func showBanner(cfg interface{}) {
	fmt.Println("")
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║   jciyuan-spider v2.0 企业级动漫爬虫                      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println("")
}

func showStats(s *spider.Spider, startTime time.Time) {
	stats := s.GetStats()
	duration := time.Since(startTime)
	fmt.Printf("耗时: %s | 请求: %d | 成功: %d | 失败: %d\n",
		duration.Round(time.Second),
		stats.TotalRequests, stats.SuccessCount, stats.FailCount)
}
```

注意：main.go 的最终版本需要在所有模块就绪后修正类型引用。此步骤先搭建骨架。

- [ ] **Step 3: 验证编译**

```bash
cd /workspace && go build ./internal/spider/...
```

- [ ] **Step 4: 提交**

```bash
git add internal/spider/ main.go && git commit -m "refactor: Spider 核心逻辑独立 + 断点续爬集成"
```

---

## Task 10: main.go 最终整合与清理

**Files:**
- Modify: `main.go`
- Delete: `model/model.go`（已迁移到 `internal/model/`）

- [ ] **Step 1: 重写 main.go 最终版**

Rewrite `main.go`:

```go
// jciyuan-spider-v2 - 企业级动漫爬虫
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jciyuan-spider-v2/internal/config"
	"jciyuan-spider-v2/internal/logger"
	"jciyuan-spider-v2/internal/model"
	"jciyuan-spider-v2/internal/spider"
)

var (
	configPath  = flag.String("config", "config/config.yaml", "配置文件路径")
	animeIDFlag = flag.Int64("id", 37439, "动漫ID")
	delayFlag   = flag.Int("delay", 0, "请求间隔(毫秒)")
	outputFlag  = flag.String("output", "", "输出目录")
	resumeFlag  = flag.Bool("resume", false, "启用断点续爬")
	incremental = flag.Bool("incremental", false, "增量更新")
	statsFlag   = flag.Bool("stats", true, "显示统计信息")
	debugFlag   = flag.Bool("debug", false, "调试模式")
)

func main() {
	flag.Parse()

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}

	applyFlags(cfg)
	initLogger(cfg)

	// 信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("收到中断信号，正在优雅退出...")
		cancel()
	}()

	// 创建爬虫
	s, err := spider.NewSpider(cfg)
	if err != nil {
		logger.Fatal("创建爬虫失败: %v", err)
	}
	defer s.Close()

	showBanner(cfg)

	// 执行爬取
	startTime := time.Now()
	if err := s.Run(ctx); err != nil {
		logger.Error("爬取失败: %v", err)
		os.Exit(1)
	}

	if *statsFlag {
		showStats(s, startTime)
	}

	logger.Info("爬取完成!")
}

func loadConfig() (*model.Config, error) {
	loader := config.NewLoader(*configPath)
	cfg, err := loader.Load()
	if err != nil {
		// 配置文件不存在时使用默认配置
		logger.Warn("配置文件不存在，使用默认配置: %v", err)
		cfg = defaultConfig()
	}

	// 环境变量覆盖
	config.LoadFromEnv(cfg)

	return cfg, nil
}

func applyFlags(cfg *model.Config) {
	if *animeIDFlag > 0 {
		cfg.Crawl.AnimeID = *animeIDFlag
	}
	if *delayFlag > 0 {
		cfg.Spider.Delay = *delayFlag
	}
	if *outputFlag != "" {
		cfg.Storage.OutputDir = *outputFlag
	}
	if *resumeFlag {
		cfg.Crawl.Resume = true
	}
	if *incremental {
		cfg.Crawl.Incremental = true
	}
	if *debugFlag {
		cfg.Log.Level = "debug"
	}
}

func initLogger(cfg *model.Config) {
	logger.SetLevel(cfg.Log.Level)
	if cfg.Log.File != "" {
		if err := logger.SetFile(cfg.Log.File); err != nil {
			fmt.Fprintf(os.Stderr, "设置日志文件失败: %v\n", err)
		}
	}
}

func defaultConfig() *model.Config {
	return &model.Config{
		Spider: model.SpiderConfig{
			BaseURL: "https://www.jciyuan.com",
			Delay:   1000, Timeout: 10, MaxRetry: 3, Concurrency: 3,
		},
		Anticrawler: model.AnticrawlerConfig{
			RandomUA:   true,
			KeepCookie: true,
			UserAgents: []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
		},
		Crawl:   model.CrawlConfig{AnimeID: 37439, Resume: true},
		Storage: model.StorageConfig{OutputDir: "./output", SaveJSON: true},
		Log:     model.LogConfig{Level: "info", Console: true},
	}
}

func showBanner(cfg *model.Config) {
	fmt.Println("")
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║   jciyuan-spider v2.0 企业级动漫爬虫                      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("目标站点: %s | 动漫ID: %d | 间隔: %dms | 重试: %d次\n",
		cfg.Spider.BaseURL, cfg.Crawl.AnimeID, cfg.Spider.Delay, cfg.Spider.MaxRetry)
	fmt.Println("")
}

func showStats(s *spider.Spider, startTime time.Time) {
	stats := s.GetStats()
	duration := time.Since(startTime)
	fmt.Printf("\n耗时: %s | 请求: %d | 成功: %d | 失败: %d | 重试: %d | 流量: %s\n",
		duration.Round(time.Second),
		stats.TotalRequests, stats.SuccessCount, stats.FailCount,
		stats.RetryCount, formatBytes(stats.Bandwidth))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
```

- [ ] **Step 2: 删除旧 model 包**

```bash
rm -rf /workspace/model/
```

- [ ] **Step 3: 全量编译验证**

```bash
cd /workspace && go build ./...
```

- [ ] **Step 4: 提交**

```bash
git add main.go && git rm -r model/ && git commit -m "refactor: main.go 精简为入口 + 删除旧 model 包"
```

---

## Task 11: 全量编译修复与验证

**Files:**
- Various fixes across `internal/`

- [ ] **Step 1: 全量编译**

```bash
cd /workspace && go build ./...
```

- [ ] **Step 2: 修复编译错误**

根据编译输出逐一修复 import 路径、类型引用等问题。确保所有包引用 `jciyuan-spider-v2/internal/...` 路径。

- [ ] **Step 3: 运行 vet 检查**

```bash
cd /workspace && go vet ./...
```

- [ ] **Step 4: 构建可执行文件**

```bash
cd /workspace && go build -o jciyuan-spider .
```

- [ ] **Step 5: 提交**

```bash
git add -A && git commit -m "fix: 修复全量编译问题，确保项目可构建"
```

---

## Task 12: 清理遗留文件

**Files:**
- Delete: `cmd/` (如果还有残留)
- Delete: `crawler/` (如果还有残留)
- Verify: 目录结构干净

- [ ] **Step 1: 检查并清理残留目录**

```bash
rm -rf /workspace/cmd/ /workspace/crawler/ /workspace/parser/ /workspace/storage/ /workspace/log/ /workspace/utils/ /workspace/model/
```

- [ ] **Step 2: 验证最终目录结构**

```bash
find /workspace -name "*.go" | sort
```

Expected 输出应只包含 `main.go` 和 `internal/` 下的文件。

- [ ] **Step 3: 最终编译验证**

```bash
cd /workspace && go build -o jciyuan-spider .
```

- [ ] **Step 4: 提交**

```bash
git add -A && git commit -m "chore: 清理遗留文件，完成项目重构"
```

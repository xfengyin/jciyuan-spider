// Package di 提供依赖注入容器，根据配置装配 Fetcher/Parser/Storage/Middleware/WorkerPool。
package di

import (
	"context"
	"fmt"

	"jciyuan-spider-v2/internal/config"
	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/logger"
	"jciyuan-spider-v2/internal/metrics"
	"jciyuan-spider-v2/internal/model"
	"jciyuan-spider-v2/internal/parser"
	"jciyuan-spider-v2/internal/resume"
	"jciyuan-spider-v2/internal/spider"
	"jciyuan-spider-v2/internal/storage"
	"jciyuan-spider-v2/internal/worker"

	// 触发各插件的 init 注册
	_ "jciyuan-spider-v2/internal/fetcher/http"
	_ "jciyuan-spider-v2/internal/parser/html"
	_ "jciyuan-spider-v2/internal/storage/json"
	memorystorage "jciyuan-spider-v2/internal/storage/memory"
	_ "jciyuan-spider-v2/internal/storage/mysql"
	_ "jciyuan-spider-v2/internal/storage/s3"
	_ "jciyuan-spider-v2/internal/storage/sqlite"
)

// Container 依赖注入容器
type Container struct {
	cfg *model.Config
}

// NewContainer 创建容器
func NewContainer(cfg *model.Config) *Container {
	return &Container{cfg: cfg}
}

// BuildLogger 构建日志器
func (c *Container) BuildLogger() (logger.Logger, error) {
	return logger.New(c.cfg.Log)
}

// BuildMetrics 构建指标收集器
func (c *Container) BuildMetrics() (metrics.Metrics, error) {
	return metrics.New(c.cfg.Metrics)
}

// BuildFetcher 构建 Fetcher 实例，并按 cfg.Middlewares 顺序组装中间件链。
func (c *Container) BuildFetcher(m metrics.Metrics, l logger.Logger) (fetcher.Fetcher, error) {
	return fetcher.Build(c.cfg.Fetcher, c.cfg.Anticrawler, c.cfg.Spider, c.cfg.Middlewares, m, l)
}

// BuildParser 构建 Parser 实例
func (c *Container) BuildParser() (parser.Parser, error) {
	return parser.Build(c.cfg.Parser)
}

// BuildStorage 构建 Storage 实例，并根据配置包装 MemoryStorage 缓存装饰器。
func (c *Container) BuildStorage() (storage.Storage, error) {
	backend, err := storage.Build(c.cfg.Storage)
	if err != nil {
		return nil, err
	}

	if c.cfg.Storage.Memory.Enable {
		return memorystorage.NewMemoryStorageFromConfig(c.cfg.Storage, backend), nil
	}
	return backend, nil
}

// BuildWorkerPool 构建 WorkerPool
func (c *Container) BuildWorkerPool() *worker.Pool {
	return worker.NewPool(c.cfg.Spider.Concurrency, c.cfg.Spider.QueueSize)
}

// BuildResumeManager 构建断点续爬管理器
func (c *Container) BuildResumeManager(statusStore storage.StatusStorage) *resume.Manager {
	return resume.NewManager(statusStore)
}

// BuildSpider 构建完全装配的爬虫实例
func (c *Container) BuildSpider(ctx context.Context) (*spider.Spider, error) {
	log, err := c.BuildLogger()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	m, err := c.BuildMetrics()
	if err != nil {
		return nil, fmt.Errorf("build metrics: %w", err)
	}

	fetcherImpl, err := c.BuildFetcher(m, log)
	if err != nil {
		return nil, fmt.Errorf("build fetcher: %w", err)
	}

	parserImpl, err := c.BuildParser()
	if err != nil {
		return nil, fmt.Errorf("build parser: %w", err)
	}

	storageImpl, err := c.BuildStorage()
	if err != nil {
		return nil, fmt.Errorf("build storage: %w", err)
	}

	// 断点续爬状态存储优先复用 storage 的 StatusStorage 能力；
	// 若后端未实现，则使用内存兜底，保证引擎仍可运行（进程退出后状态会丢失）。
	statusStore, ok := storageImpl.(storage.StatusStorage)
	if !ok {
		log.Warn("当前存储后端未实现 StatusStorage，断点续爬状态将仅存于内存")
		statusStore = newMemoryStatusStore()
	}

	pool := c.BuildWorkerPool()
	resumeMgr := c.BuildResumeManager(statusStore)

	s := spider.New(spider.Options{
		Config:  c.cfg,
		Fetcher: fetcherImpl,
		Parser:  parserImpl,
		Storage: storageImpl,
		Resume:  resumeMgr,
		Pool:    pool,
		Metrics: m,
		Logger:  log,
	})

	return s, nil
}

// MustLoad 加载配置文件并应用环境变量
func MustLoad(configPath string) *model.Config {
	loader := config.NewLoader(configPath)
	cfg, err := loader.Load()
	if err != nil {
		// 配置文件不存在时使用默认配置
		cfg = defaultConfig()
	}
	config.LoadFromEnv(cfg)
	return cfg
}

// defaultConfig 返回默认配置
func defaultConfig() *model.Config {
	return &model.Config{
		App: model.AppConfig{
			Name:          "jciyuan-spider-v3",
			Mode:          "cli",
			TraceIDHeader: "X-Request-ID",
		},
		Spider: model.SpiderConfig{
			BaseURL:          "https://www.jciyuan.com",
			DetailURLPattern: "{{base_url}}/acgdetail/{{id}}.html",
			Delay:            1000,
			Timeout:          10,
			MaxRetry:         3,
			Concurrency:      3,
			QueueSize:        100,
		},
		Anticrawler: model.AnticrawlerConfig{
			RandomUA:       true,
			KeepCookie:     true,
			UserAgents:     []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
			RefererPolicy:  "https://www.jciyuan.com/",
			RobotsTxtCheck: false,
		},
		Fetcher: model.FetcherConfig{
			Type: "http",
			HTTP: model.HTTPFetcherConfig{
				Timeout:         10,
				MaxRetry:        3,
				FollowRedirects: true,
				MaxBodySize:     50 * 1024 * 1024,
			},
		},
		Parser: model.ParserConfig{
			Type: "html",
			HTML: model.HTMLParserConfig{Encoding: "auto"},
		},
		Storage: model.StorageConfig{
			Type: "json",
			JSON: model.JSONStorageConfig{OutputDir: "./output"},
			Output: model.OutputConfig{
				SaveJSON: true,
			},
		},
		Crawl: model.CrawlConfig{
			AnimeID: 37439,
			Resume:  true,
		},
		Metrics: model.MetricsConfig{
			Enabled: true,
			Backend: "memory",
		},
		Log: model.LogConfig{
			Level:   "info",
			Format:  "text",
			Console: true,
		},
	}
}

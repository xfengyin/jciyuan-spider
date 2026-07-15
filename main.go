// jciyuan-spider-v3 - 企业级动漫爬虫命令行入口。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jciyuan-spider-v2/internal/config"
	"jciyuan-spider-v2/internal/di"
	"jciyuan-spider-v2/internal/health"
	"jciyuan-spider-v2/internal/logger"
	"jciyuan-spider-v2/internal/model"
	"jciyuan-spider-v2/internal/spider"
)

var (
	configPath  = flag.String("config", "config/config.yaml", "配置文件路径")
	animeIDFlag = flag.Int64("id", 0, "动漫ID")
	delayFlag   = flag.Int("delay", 0, "请求间隔(毫秒)")
	outputFlag  = flag.String("output", "", "输出目录")
	resumeFlag  = flag.Bool("resume", false, "启用断点续爬")
	incremental = flag.Bool("incremental", false, "增量更新")
	statsFlag   = flag.Bool("stats", true, "显示统计信息")
	debugFlag   = flag.Bool("debug", false, "调试模式")
)

func main() {
	flag.Parse()

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}
	applyFlags(cfg)

	log, err := logger.New(cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志器失败: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("收到中断信号，正在优雅退出...")
		cancel()
	}()

	traceID := generateTraceID()
	ctx = logger.WithTraceID(ctx, traceID)
	log = log.WithTraceID(traceID)

	container := di.NewContainer(cfg)
	s, err := container.BuildSpider(ctx)
	if err != nil {
		log.Fatal("创建爬虫失败", logger.Err(err))
	}
	defer s.Close()

	startTime := time.Now()
	startHealthServer(ctx, cfg, log, s, startTime)

	showBanner(cfg)

	if err := s.Run(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Error("爬取失败", logger.Err(err))
			os.Exit(1)
		}
		log.Warn("爬取被取消", logger.Err(err))
	}

	if *statsFlag {
		showStats(s, startTime, log)
	}

	log.Info("爬取完成")
}

// loadConfig 从文件加载配置；若文件不存在则回退到默认配置，并应用环境变量覆盖。
func loadConfig() (*model.Config, error) {
	loader := config.NewLoader(*configPath)
	cfg, err := loader.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置文件加载失败，使用默认配置: %v\n", err)
		cfg = defaultConfig()
	}
	config.LoadFromEnv(cfg)
	return cfg, nil
}

// applyFlags 将命令行 flag 覆盖到配置对象。
func applyFlags(cfg *model.Config) {
	if *animeIDFlag > 0 {
		cfg.Crawl.AnimeID = *animeIDFlag
	}
	if *delayFlag > 0 {
		cfg.Spider.Delay = *delayFlag
	}
	if *outputFlag != "" {
		cfg.Storage.JSON.OutputDir = *outputFlag
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

// defaultConfig 返回命令行入口的默认配置，保证无配置文件时仍可运行。
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
			RandomUA:   true,
			KeepCookie: true,
			UserAgents: []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
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

// generateTraceID 生成一条简单的全链路 Trace ID。
func generateTraceID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
}

// showBanner 打印启动横幅。
func showBanner(cfg *model.Config) {
	fmt.Println("")
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║   jciyuan-spider v3.0 企业级动漫爬虫                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("目标站点: %s | 动漫ID: %d | 间隔: %dms | 重试: %d次\n",
		cfg.Spider.BaseURL, cfg.Crawl.AnimeID, cfg.Spider.Delay, cfg.Spider.MaxRetry)
	fmt.Println("")
}

// startHealthServer 根据配置启动健康检查服务；当 metrics backend 为 prometheus 时，
// /healthz 与 /metrics 共用 MetricsConfig.Prometheus.Port 端口。
func startHealthServer(ctx context.Context, cfg *model.Config, log logger.Logger, s *spider.Spider, startTime time.Time) {
	if cfg.Metrics.Backend != "prometheus" {
		return
	}
	checker := health.NewChecker(
		cfg,
		log,
		s.Metrics(),
		func() string { return string(s.State()) },
		startTime,
	)
	if err := checker.Start(ctx); err != nil {
		log.Warn("启动健康/指标服务失败", logger.Err(err))
		return
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = checker.Stop(shutdownCtx)
	}()
}

// showStats 打印运行统计信息。
func showStats(s interface{ GetStats() model.Stats }, startTime time.Time, log logger.Logger) {
	stats := s.GetStats()
	duration := time.Since(startTime)
	log.Info("运行统计",
		logger.String("duration", duration.Round(time.Second).String()),
		logger.Int64("requests", stats.TotalRequests),
		logger.Int64("success", stats.SuccessCount),
		logger.Int64("fail", stats.FailCount),
		logger.Int64("retry", stats.RetryCount),
		logger.Int64("parse", stats.ParseCount),
		logger.Int64("parse_fail", stats.ParseFailCount),
		logger.Int64("storage_save", stats.StorageSaveCount),
		logger.Int64("storage_save_fail", stats.StorageSaveFail),
		logger.String("bandwidth", formatBytes(stats.Bandwidth)),
	)
}

// formatBytes 将字节数转换为可读字符串。
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

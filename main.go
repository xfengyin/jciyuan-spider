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

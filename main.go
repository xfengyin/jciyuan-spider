// jciyuan-spider-v2 - 企业级动漫爬虫
// 
// 基于 SUPERSpider 理念设计，支持：
// - Random UA / Referer
// - 请求限流 / 重试机制
// - Cookie保持
// - 代理池支持
// - goroutine并发池
// - 断点续爬
// - JSON/SQLite存储
// - 统计与监控
// 
// @author Kongming Agent
// @version 2.0.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"jciyuan-spider-v2/cmd/config"
	"jciyuan-spider-v2/crawler"
	"jciyuan-spider-v2/log"
	"jciyuan-spider-v2/model"
	"jciyuan-spider-v2/parser"
	"jciyuan-spider-v2/storage"
	
)

// ============================================================================
// 全局变量
// ============================================================================

var (
	configPath = flag.String("config", "config/config.yaml", "配置文件路径")
	animeID    = flag.Int64("id", 37439, "动漫ID")
	url        = flag.String("url", "", "直接指定URL")
	delay      = flag.Int("delay", 0, "请求间隔(毫秒)")
	output     = flag.String("output", "", "输出目录")
	resume     = flag.Bool("resume", false, "启用断点续爬")
	incremental = flag.Bool("incremental", false, "增量更新")
	stats      = flag.Bool("stats", true, "显示统计信息")
	debug      = flag.Bool("debug", false, "调试模式")
)

// ============================================================================
// 主程序
// ============================================================================

func main() {
	// 解析命令行参数
	flag.Parse()
	
	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 配置加载失败: %v\n", err)
		os.Exit(1)
	}
	
	// 应用命令行参数覆盖
	applyArgs(cfg)
	
	// 初始化日志
	initLog(cfg)
	
	// 创建上下文和取消信号
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// 信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("收到中断信号，正在优雅退出...")
		cancel()
	}()
	
	// 创建爬虫
	spider, err := NewSpider(cfg)
	if err != nil {
		log.Fatal("创建爬虫失败: %v", err)
	}
	
	// 显示欢迎信息
	showBanner(cfg)
	
	// 执行爬取
	startTime := time.Now()
	if err := spider.Run(ctx); err != nil {
		log.Error("爬取失败: %v", err)
		os.Exit(1)
	}
	
	// 显示统计
	if *stats {
		showStats(spider, startTime)
	}
	
	log.Info("爬取完成!")
}

// ============================================================================
// 爬虫
// ============================================================================

// Spider 爬虫实例
type Spider struct {
	config   *model.Config
	fetcher  *crawler.Fetcher
	parser   *parser.Parser
	storage  *storage.MemoryStorage
	stats    *crawler.StatsCollector
	seenURLs sync.Map
	status   *model.CrawlStatus
}

// NewSpider 创建爬虫
func NewSpider(cfg *model.Config) (*Spider, error) {
	// 创建统计收集器
	stats := crawler.NewStatsCollector()
	
	// 创建请求器
	fetcher := crawler.NewFetcher(cfg, stats)
	
	// 创建解析器
	parse := parser.NewParser(cfg.Spider.BaseURL)
	
	// 创建存储
	jsonStorage, err := storage.NewJSONStorage(cfg.Storage.OutputDir)
	if err != nil {
		return nil, err
	}
	memStorage := storage.NewMemoryStorage(jsonStorage)
	
	return &Spider{
		config:  cfg,
		fetcher: fetcher,
		parser:  parse,
		storage: memStorage,
		stats:   stats,
		status: &model.CrawlStatus{
			AnimeID: cfg.Crawl.AnimeID,
			Status:  "pending",
		},
	}, nil
}

// Run 运行爬虫
func (s *Spider) Run(ctx context.Context) error {
	log.Info("开始爬取动漫 ID: %d", s.config.Crawl.AnimeID)
	
	// 构建URL
	url := s.buildURL()
	
	// 获取详情页HTML
	html, err := s.fetcher.Fetch(ctx, url)
	if err != nil {
		return fmt.Errorf("获取页面失败: %v", err)
	}
	
	// 解析动漫信息
	anime, err := s.parser.ParseAnimeDetail(string(html))
	if err != nil {
		s.stats.IncrParseFail()
		return fmt.Errorf("解析失败: %v", err)
	}
	
	anime.ID = s.config.Crawl.AnimeID
	anime.DetailURL = url
	
	s.stats.IncrParse()
	log.Info("解析成功: %s, 共 %d 集", anime.Title, anime.EpisodeNum)
	
	// 保存结果
	if err := s.storage.Save(anime); err != nil {
		log.Error("保存失败: %v", err)
	}
	
	// 显示剧集预览
	s.showEpisodePreview(anime)
	
	return nil
}

// buildURL 构建URL
func (s *Spider) buildURL() string {
	if *url != "" {
		return *url
	}
	return fmt.Sprintf("%s/acgdetail/%d.html", s.config.Spider.BaseURL, *animeID)
}

// showEpisodePreview 显示剧集预览
func (s *Spider) showEpisodePreview(anime *model.AnimeInfo) {
	log.Info("📺 标题: %s", anime.Title)
	if anime.Year != "" {
		log.Info("📅 年份: %s", anime.Year)
	}
	if anime.Region != "" {
		log.Info("🌍 地区: %s", anime.Region)
	}
	if len(anime.Tags) > 0 {
		log.Info("🏷️ 标签: %s", strings.Join(anime.Tags, ", "))
	}
	if anime.UpdateDate != "" {
		log.Info("📅 更新: %s", anime.UpdateDate)
	}
	log.Info("🎞️ 集数: %d集", anime.EpisodeNum)
	if anime.CoverImage != "" {
		log.Info("🖼️ 封面: %s", anime.CoverImage)
	}
	if anime.Description != "" {
		desc := anime.Description
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		log.Info("📝 简介: %s", desc)
	}
	
	// 显示前5集
	if len(anime.Episodes) > 0 {
		log.Info("")
		log.Info("📋 剧集列表 (前5集):")
		for i := 0; i < 5 && i < len(anime.Episodes); i++ {
			ep := anime.Episodes[i]
			vipTag := ""
			if ep.IsVIP {
				vipTag = " [VIP]"
			}
			log.Info("   [%02d] %s%s", ep.Number, ep.Title, vipTag)
		}
		if anime.EpisodeNum > 5 {
			log.Info("   ... 共 %d 集", anime.EpisodeNum)
		}
	}
}

// ============================================================================
// 配置与初始化
// ============================================================================

func loadConfig() (*model.Config, error) {
	// 加载配置文件
	loader := config.NewLoader(*configPath)
	cfg, err := loader.Load()
	if err != nil {
		// 如果配置文件不存在，使用默认配置
		log.Warn("配置文件不存在，使用默认配置: %v", err)
		cfg = &model.Config{
			Spider: model.SpiderConfig{
				BaseURL:     "https://www.jciyuan.com",
				Delay:       1000,
				Timeout:     10,
				MaxRetry:    3,
				Concurrency: 3,
			},
			Anticrawler: model.AnticrawlerConfig{
				RandomUA:   true,
				KeepCookie: true,
				UserAgents: []string{
					"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				},
			},
			Crawl: model.CrawlConfig{
				AnimeID: 37439,
				Resume:  true,
			},
			Storage: model.StorageConfig{
				OutputDir: "./output",
				SaveJSON: true,
			},
			Log: model.LogConfig{
				Level:   "info",
				Console: true,
			},
		}
	}
	
	// 从环境变量加载
	envCfg := config.LoadFromEnv()
	if envCfg != nil {
		if envCfg.Spider.BaseURL != "" {
			cfg.Spider.BaseURL = envCfg.Spider.BaseURL
		}
		if envCfg.Spider.Delay > 0 {
			cfg.Spider.Delay = envCfg.Spider.Delay
		}
	}
	
	return cfg, nil
}

func applyArgs(cfg *model.Config) {
	// 命令行参数覆盖
	if *animeID > 0 {
		cfg.Crawl.AnimeID = *animeID
	}
	if *delay > 0 {
		cfg.Spider.Delay = *delay
	}
	if *output != "" {
		cfg.Storage.OutputDir = *output
	}
	if *resume {
		cfg.Crawl.Resume = true
	}
	if *incremental {
		cfg.Crawl.Incremental = true
	}
	if *debug {
		cfg.Log.Level = "debug"
	}
}

func initLog(cfg *model.Config) {
	log.SetLevel(cfg.Log.Level)
	
	if cfg.Log.File != "" {
		if err := log.SetFile(cfg.Log.File); err != nil {
			fmt.Fprintf(os.Stderr, "设置日志文件失败: %v\n", err)
		}
	}
}

func showBanner(cfg *model.Config) {
	fmt.Println("")
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                            ║")
	fmt.Println("║   🎬 jciyuan-spider v2.0 企业级动漫爬虫                   ║")
	fmt.Println("║                                                            ║")
	fmt.Println("║   ⚡ SUPERSpider 风格 · 企业级标准                         ║")
	fmt.Println("║   🛡️ 抗反爬 · 限流 · 重试                                  ║")
	fmt.Println("║   📦 断点续爬 · 增量更新                                   ║")
	fmt.Println("║                                                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Printf("📌 目标站点: %s\n", cfg.Spider.BaseURL)
	fmt.Printf("📌 动漫ID: %d\n", cfg.Crawl.AnimeID)
	fmt.Printf("⏱️  请求间隔: %dms\n", cfg.Spider.Delay)
	fmt.Printf("🔄 最大重试: %d次\n", cfg.Spider.MaxRetry)
	fmt.Printf("⚡ 并发数: %d\n", cfg.Spider.Concurrency)
	fmt.Println("")
}

func showStats(spider *Spider, startTime time.Time) {
	stats := spider.stats.GetStats()
	duration := time.Since(startTime)
	
	fmt.Println("")
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     📊 爬取统计                              ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ ⏱️  耗时: %s\n", duration.Round(time.Second))
	fmt.Printf("║ 🌐 总请求: %d\n", stats.TotalRequests)
	fmt.Printf("║ ✅ 成功: %d | ❌ 失败: %d | 🔄 重试: %d\n", 
		stats.SuccessCount, stats.FailCount, stats.RetryCount)
	fmt.Printf("║ 📝 解析: %d | ❌ 解析失败: %d\n", stats.ParseCount, stats.ParseFailCount)
	fmt.Printf("║ 💾 流量: %s\n", formatBytes(stats.Bandwidth))
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
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

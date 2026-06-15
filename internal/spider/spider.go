// Package spider - 爬虫核心编排层
package spider

import (
	"context"
	"fmt"
	"strings"

	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/logger"
	"jciyuan-spider-v2/internal/metrics"
	"jciyuan-spider-v2/internal/model"
	"jciyuan-spider-v2/internal/parser"
	"jciyuan-spider-v2/internal/resume"
	"jciyuan-spider-v2/internal/storage"
)

// Spider 爬虫实例，编排各子模块协同工作
type Spider struct {
	config    *model.Config
	fetcher   fetcher.Fetcher
	parser    parser.Parser
	storage   storage.Storage
	metrics   *metrics.Collector
	resume    *resume.Manager
	log       *logger.Logger
}

// NewSpider 创建爬虫实例
func NewSpider(cfg *model.Config) (*Spider, error) {
	// 创建指标收集器
	m := metrics.NewCollector()

	// 创建 HTTP 请求器
	httpFetcher, err := fetcher.NewHTTPFetcher(cfg, m)
	if err != nil {
		return nil, fmt.Errorf("创建请求器失败: %w", err)
	}

	// 创建 HTML 解析器
	htmlParser := parser.NewHTMLParser(cfg.Spider.BaseURL)

	// 创建 JSON 持久化存储
	jsonStore, err := storage.NewJSONStorage(cfg.Storage.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("创建存储失败: %w", err)
	}

	// 创建内存缓存存储（装饰器模式）
	memStore := storage.NewMemoryStorage(jsonStore, jsonStore)

	// 创建断点续爬管理器
	resumeMgr := resume.NewManager(jsonStore)

	return &Spider{
		config:  cfg,
		fetcher: httpFetcher,
		parser:  htmlParser,
		storage: memStore,
		metrics: m,
		resume:  resumeMgr,
		log:     logger.GetLogger("spider"),
	}, nil
}

// Run 执行爬取任务
func (s *Spider) Run(ctx context.Context) error {
	s.log.Info("开始爬取动漫 ID: %d", s.config.Crawl.AnimeID)

	// 断点续爬检查
	if s.config.Crawl.Resume && s.resume.IsCompleted(s.config.Crawl.AnimeID) {
		s.log.Info("动漫 %d 已完成爬取，跳过", s.config.Crawl.AnimeID)
		return nil
	}

	// 标记运行状态
	if s.config.Crawl.Resume {
		if err := s.resume.MarkRunning(s.config.Crawl.AnimeID); err != nil {
			s.log.Warn("标记运行状态失败: %v", err)
		}
	}

	// 构建详情页 URL
	url := s.buildURL()

	// 获取详情页 HTML
	html, err := s.fetcher.Fetch(ctx, url)
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	// 解析动漫信息
	anime, err := s.parser.ParseAnimeDetail(string(html))
	if err != nil {
		s.metrics.IncrParseFail()
		return fmt.Errorf("解析失败: %w", err)
	}

	anime.ID = s.config.Crawl.AnimeID
	anime.DetailURL = url

	s.metrics.IncrParse()
	s.log.Info("解析成功: %s, 共 %d 集", anime.Title, anime.EpisodeNum)

	// 保存结果
	if err := s.storage.Save(anime); err != nil {
		s.log.Error("保存失败: %v", err)
	}

	// 显示摘要
	s.showSummary(anime)

	// 标记完成
	if s.config.Crawl.Resume {
		if err := s.resume.MarkCompleted(s.config.Crawl.AnimeID); err != nil {
			s.log.Warn("标记完成状态失败: %v", err)
		}
	}

	return nil
}

// Close 释放资源
func (s *Spider) Close() {
	if s.storage != nil {
		_ = s.storage.Close()
	}
}

// GetStats 获取统计快照
func (s *Spider) GetStats() model.Stats {
	return s.metrics.GetStats()
}

// buildURL 构建详情页 URL
func (s *Spider) buildURL() string {
	return fmt.Sprintf("%s/acgdetail/%d.html",
		s.config.Spider.BaseURL, s.config.Crawl.AnimeID)
}

// showSummary 显示爬取摘要
func (s *Spider) showSummary(anime *model.AnimeInfo) {
	s.log.Info("标题: %s", anime.Title)
	if anime.Year != "" {
		s.log.Info("年份: %s", anime.Year)
	}
	if anime.Region != "" {
		s.log.Info("地区: %s", anime.Region)
	}
	if len(anime.Tags) > 0 {
		s.log.Info("标签: %s", strings.Join(anime.Tags, ", "))
	}
	s.log.Info("集数: %d集", anime.EpisodeNum)

	if len(anime.Episodes) > 0 {
		s.log.Info("剧集列表 (前5集):")
		for i := 0; i < 5 && i < len(anime.Episodes); i++ {
			ep := anime.Episodes[i]
			vipTag := ""
			if ep.IsVIP {
				vipTag = " [VIP]"
			}
			s.log.Info("  [%02d] %s%s", ep.Number, ep.Title, vipTag)
		}
		if anime.EpisodeNum > 5 {
			s.log.Info("  ... 共 %d 集", anime.EpisodeNum)
		}
	}
}

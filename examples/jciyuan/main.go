// 示例：将内部 Fetcher + Parser 组件包装为框架的 Crawler 接口实现，
// 通过通用 Engine 调度，演示「单一站点爬虫」如何接入「通用爬虫框架」。
//
// 运行（在仓库根目录）：
//
//	go run ./examples/jciyuan -id 37439
//	go run ./examples/jciyuan -url https://www.jciyuan.com/acgdetail/37439.html -output ./output/jciyuan
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"jciyuan-spider/crawler"
	"jciyuan-spider/internal/config"
	"jciyuan-spider/internal/di"
	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/logger"
	"jciyuan-spider/internal/metrics"
	"jciyuan-spider/internal/model"
	"jciyuan-spider/internal/parser"
	"jciyuan-spider/internal/version"

	// 副作用导入，触发 SPI 注册
	_ "jciyuan-spider/internal/fetcher/http"
	_ "jciyuan-spider/internal/parser/html"
)

var (
	configPath  = flag.String("config", "config/config.yaml", "配置文件路径（缺失时使用内置默认配置）")
	animeIDFlag = flag.Int64("id", 0, "动漫 ID（配置模板自动生成详情页 URL）")
	urlFlag     = flag.String("url", "", "直接指定详情页 URL（优先于 -id）")
	outputFlag  = flag.String("output", "", "输出目录（默认 ./output/jciyuan）")
	concurrency = flag.Int("concurrency", 3, "并发数")
	maxRetry    = flag.Int("max-retry", 3, "失败重试次数")
	quietFlag   = flag.Bool("quiet", false, "安静模式")
	versionFlag = flag.Bool("version", false, "打印版本信息")
)

// jciyuanCrawler 将内部 Fetcher/Parser 适配为框架的 Crawler 接口。
type jciyuanCrawler struct {
	fetcher fetcher.Fetcher
	parser  parser.Parser
}

// Name 返回爬虫名称。
func (c *jciyuanCrawler) Name() string { return "jciyuan" }

// Fetch 通过内部 HTTP Fetcher（含中间件链、抗反爬）抓取页面。
func (c *jciyuanCrawler) Fetch(ctx context.Context, url string) (*crawler.Page, error) {
	resp, err := c.fetcher.Fetch(ctx, &fetcher.Request{URL: url, Method: "GET"})
	if err != nil {
		return nil, err
	}
	return &crawler.Page{
		URL:        resp.URL,
		StatusCode: resp.StatusCode,
		Headers:    resp.Headers,
		Body:       resp.Body,
		Text:       string(resp.Body),
		Meta:       resp.Meta,
	}, nil
}

// Parse 复用配置驱动的 HTML Pipeline 解析器。
func (c *jciyuanCrawler) Parse(ctx context.Context, page *crawler.Page) (any, error) {
	resp := &fetcher.Response{
		URL:        page.URL,
		StatusCode: page.StatusCode,
		Headers:    page.Headers,
		Body:       page.Body,
		Meta:       page.Meta,
	}
	return c.parser.Parse(ctx, resp)
}

// Extract 将解析结果映射为通用 Item。
func (c *jciyuanCrawler) Extract(_ context.Context, parsed any) ([]crawler.Item, error) {
	pr, ok := parsed.(*parser.ParseResult)
	if !ok {
		return nil, fmt.Errorf("解析结果类型异常: %T", parsed)
	}
	if pr.Anime == nil {
		return nil, nil
	}

	episodes := make([]crawler.Item, 0, len(pr.Episodes))
	for _, ep := range pr.Episodes {
		episodes = append(episodes, crawler.Item{
			"number":     ep.Number,
			"title":      ep.Title,
			"url":        ep.URL,
			"m3u8_url":   ep.M3U8URL,
			"is_vip":     ep.IsVIP,
			"is_crawled": ep.IsCrawled,
		})
	}

	item := crawler.Item{
		"id":          pr.Anime.ID,
		"title":       pr.Anime.Title,
		"year":        pr.Anime.Year,
		"region":      pr.Anime.Region,
		"tags":        pr.Anime.Tags,
		"cover_image": pr.Anime.CoverImage,
		"description": pr.Anime.Description,
		"update_date": pr.Anime.UpdateDate,
		"episode_num": pr.Anime.EpisodeNum,
		"detail_url":  pr.Anime.DetailURL,
		"episodes":    episodes,
	}
	return []crawler.Item{item}, nil
}

// Close 释放底层资源。
func (c *jciyuanCrawler) Close() {
	if c.fetcher != nil {
		_ = c.fetcher.Close()
	}
}

// newJCIYUANCrawler 按配置装配内部组件，构造 Crawler 实现。
func newJCIYUANCrawler(cfg *model.Config) (*jciyuanCrawler, error) {
	log := logger.GetLogger("examples/jciyuan")
	m := metrics.NewMemoryMetrics()

	f, err := fetcher.Build(cfg.Fetcher, cfg.Anticrawler, cfg.Spider, cfg.Middlewares, m, log)
	if err != nil {
		return nil, fmt.Errorf("构建 Fetcher 失败: %w", err)
	}

	p, err := parser.Build(cfg.Parser)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("构建 Parser 失败: %w", err)
	}

	return &jciyuanCrawler{fetcher: f, parser: p}, nil
}

// buildURL 根据配置模板生成详情页 URL（与内部 spider 的规则一致）。
func buildURL(cfg *model.Config, id int64) string {
	pattern := cfg.Spider.DetailURLPattern
	if pattern == "" {
		pattern = "{{base_url}}/acgdetail/{{id}}.html"
	}
	pattern = strings.ReplaceAll(pattern, "{{base_url}}", cfg.Spider.BaseURL)
	return strings.ReplaceAll(pattern, "{{id}}", fmt.Sprintf("%d", id))
}

func main() {
	flag.Parse()
	if *versionFlag {
		fmt.Printf("jciyuan-spider jciyuan example %s\n", version.Version)
		os.Exit(0)
	}

	cfg := &model.Config{}
	if c, err := config.NewLoader(*configPath).Load(); err == nil {
		cfg = c
		fmt.Fprintf(os.Stderr, "已加载配置: %s\n", *configPath)
	} else {
		cfg = di.DefaultConfig()
		fmt.Fprintf(os.Stderr, "配置加载失败（%v），使用内置默认配置\n", err)
	}

	// 演示 SPI：按名称注册并构建 Crawler。
	crawler.Register("jciyuan", func(m map[string]any) (crawler.Crawler, error) {
		c, err := newJCIYUANCrawler(cfg)
		return c, err
	})
	c, err := crawler.Build("jciyuan", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "构建爬虫失败: %v\n", err)
		os.Exit(1)
	}
	jc := c.(*jciyuanCrawler)
	defer jc.Close()

	url := *urlFlag
	if url == "" {
		id := *animeIDFlag
		if id == 0 {
			id = cfg.Crawl.AnimeID
		}
		if id == 0 {
			fmt.Fprintln(os.Stderr, "请通过 -id 或 -url 指定要抓取的动漫")
			os.Exit(2)
		}
		url = buildURL(cfg, id)
	}

	out := *outputFlag
	if out == "" {
		out = "./output/jciyuan"
	}

	ctx := context.Background()
	eng := crawler.NewEngine(c, crawler.Options{
		Concurrency: *concurrency,
		Delay:       time.Duration(cfg.Spider.Delay) * time.Millisecond,
		MaxRetry:    *maxRetry,
		Timeout:     time.Duration(cfg.Spider.Timeout) * time.Second,
		OutputDir:   out,
		Quiet:       *quietFlag,
	})

	fmt.Printf("抓取: %s（%s 示例，Engine 并发=%d 重试=%d）\n", url, c.Name(), *concurrency, *maxRetry)
	result, err := eng.Run(ctx, []string{url})
	if err != nil {
		fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
		os.Exit(1)
	}

	// 打印第一条结果的摘要（JSONL 已写入输出目录）。
	if len(result.Items) > 0 {
		pretty, _ := json.MarshalIndent(result.Items[0], "", "  ")
		fmt.Println(string(pretty))
	}
	fmt.Printf("统计: 成功=%d 失败=%d 重试=%d 条目=%d 耗时=%s\n",
		result.Stats.Success, result.Stats.Fail, result.Stats.Retry,
		result.Stats.Items, result.Stats.Elapsed.Round(time.Millisecond))
	if len(result.Failures) > 0 {
		for _, f := range result.Failures {
			fmt.Fprintf(os.Stderr, "失败 %s: %s\n", f.URL, f.Error)
		}
	}
}

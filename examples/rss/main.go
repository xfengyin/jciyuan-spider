// 示例：用 crawler 框架抓取并解析 RSS/XML 源。
// Fetch 同时支持远程 URL 与本地文件（离线演示/测试），Parse 用 encoding/xml
// 解析 RSS 2.0，Extract 输出 {title, link, description, pub_date} 条目列表。
//
// 运行（在仓库根目录）：
//
//	go run ./examples/rss -url examples/rss/sample.xml        # 本地文件（离线演示）
//	go run ./examples/rss -url https://example.org/feed.xml   # 远程 RSS 源
//	go run ./examples/rss -url https://httpbin.org/xml        # httpbin 的 XML 响应
package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"jciyuan-spider/crawler"
)

// rssFeed RSS 2.0 文档结构（仅覆盖常用字段）。
type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel channel  `xml:"channel"`
}

type channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []item `xml:"item"`
}

type item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// rssCrawler 实现 crawler.Crawler：Fetch 支持本地文件与远程 URL。
type rssCrawler struct {
	fetcher *crawler.HTTPFetcher
}

func newRSSCrawler() *rssCrawler {
	return &rssCrawler{fetcher: crawler.NewHTTPFetcher()}
}

func (c *rssCrawler) Name() string { return "rss" }

func (c *rssCrawler) Fetch(ctx context.Context, url string) (*crawler.Page, error) {
	// 本地文件（file:// 或普通路径）直接读取，便于离线演示与测试。
	if path, ok := localPath(url); ok {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("读取本地文件失败: %w", err)
		}
		return &crawler.Page{URL: url, StatusCode: 200, Body: body, Text: string(body)}, nil
	}
	return c.fetcher.Fetch(ctx, url)
}

func (c *rssCrawler) Parse(_ context.Context, page *crawler.Page) (any, error) {
	var feed rssFeed
	if err := xml.Unmarshal(page.Body, &feed); err != nil {
		return nil, fmt.Errorf("解析 RSS/XML 失败: %w", err)
	}
	if feed.Channel.Title == "" && len(feed.Channel.Items) == 0 {
		return nil, fmt.Errorf("不是有效的 RSS 2.0 文档（缺少 channel.title / item）")
	}
	return &feed, nil
}

func (c *rssCrawler) Extract(_ context.Context, parsed any) ([]crawler.Item, error) {
	feed, ok := parsed.(*rssFeed)
	if !ok {
		return nil, fmt.Errorf("解析结果类型异常: %T", parsed)
	}
	items := make([]crawler.Item, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		items = append(items, crawler.Item{
			"title":       it.Title,
			"link":        it.Link,
			"description": it.Description,
			"pub_date":    it.PubDate,
		})
	}
	return items, nil
}

// localPath 判断 url 是否为本地文件路径；是则返回去掉 file:// 前缀的路径。
func localPath(url string) (string, bool) {
	u := strings.TrimPrefix(url, "file://")
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return "", false
	}
	if _, err := os.Stat(u); err == nil {
		return u, true
	}
	return "", false
}

var (
	urlFlag    = flag.String("url", "", "RSS/XML 源地址：http(s):// 远程 URL 或本地文件路径")
	outputFlag = flag.String("output", "", "输出目录（默认 ./output/rss）")
	quietFlag  = flag.Bool("quiet", false, "安静模式")
)

func main() {
	flag.Parse()
	if *urlFlag == "" {
		fmt.Fprintln(os.Stderr, "请通过 -url 指定 RSS/XML 源，例如：\n"+
			"  go run ./examples/rss -url examples/rss/sample.xml\n"+
			"  go run ./examples/rss -url https://example.org/feed.xml")
		os.Exit(2)
	}

	out := *outputFlag
	if out == "" {
		out = "./output/rss"
	}

	eng := crawler.NewEngine(newRSSCrawler(), crawler.Options{
		Concurrency: 1, // RSS 源串行抓取即可
		MaxRetry:    2,
		Timeout:     10 * time.Second,
		OutputDir:   out,
		Quiet:       *quietFlag,
	})

	fmt.Printf("抓取 RSS 源: %s（%s 示例）\n", *urlFlag, "rss")
	result, err := eng.Run(context.Background(), []string{*urlFlag})
	if err != nil {
		fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
		os.Exit(1)
	}

	for i, it := range result.Items {
		pretty, _ := json.MarshalIndent(it, "", "  ")
		fmt.Printf("--- item #%d ---\n%s\n", i+1, pretty)
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

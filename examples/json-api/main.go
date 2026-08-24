// 示例：用 crawler 框架抓取 JSON API 并输出结构化 item。
// Fetch 用默认 HTTP 抓取器；Parse 用 encoding/json 解析响应（优先匹配
// slideshow 结构，不匹配时回退为原始 map）；Extract 把每条 slide 展开为
// 一个结构化 item（演示 JSON 数组 → 多条 Item 的映射）。
//
// 运行（在仓库根目录）：
//
//	go run ./examples/json-api -url https://httpbin.org/json
//	go run ./examples/json-api -url https://httpbin.org/json -output ./output/json-api
//	go run ./examples/json-api -url https://api.example.com/v1/items
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
)

// slideshowResp 对应 httpbin.org/json 的响应结构。
type slideshowResp struct {
	Slideshow struct {
		Author string `json:"author"`
		Date   string `json:"date"`
		Title  string `json:"title"`
		Slides []struct {
			Title string `json:"title"`
			Type  string `json:"type"`
		} `json:"slides"`
	} `json:"slideshow"`
}

// jsonAPICrawler 实现 crawler.Crawler，抓取并解析任意 JSON API。
type jsonAPICrawler struct {
	fetcher *crawler.HTTPFetcher
}

func newJSONAPICrawler() *jsonAPICrawler {
	return &jsonAPICrawler{fetcher: crawler.NewHTTPFetcher()}
}

func (c *jsonAPICrawler) Name() string { return "json-api" }

func (c *jsonAPICrawler) Fetch(ctx context.Context, url string) (*crawler.Page, error) {
	return c.fetcher.Fetch(ctx, url)
}

// Parse 优先按 slideshow 结构解析；不匹配时回退为原始 map，保证任意 JSON API 可用。
func (c *jsonAPICrawler) Parse(_ context.Context, page *crawler.Page) (any, error) {
	var resp slideshowResp
	if err := json.Unmarshal(page.Body, &resp); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	if resp.Slideshow.Title != "" || len(resp.Slideshow.Slides) > 0 {
		return &resp, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(page.Body, &raw); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	return raw, nil
}

// Extract 将 slideshow 的每条 slide 展开为一条结构化 item；回退场景输出整包 JSON。
func (c *jsonAPICrawler) Extract(_ context.Context, parsed any) ([]crawler.Item, error) {
	switch p := parsed.(type) {
	case *slideshowResp:
		items := make([]crawler.Item, 0, len(p.Slideshow.Slides))
		for _, sl := range p.Slideshow.Slides {
			items = append(items, crawler.Item{
				"show_title":  p.Slideshow.Title,
				"author":      p.Slideshow.Author,
				"date":        p.Slideshow.Date,
				"slide_title": sl.Title,
				"type":        sl.Type,
			})
		}
		return items, nil
	case map[string]any:
		return []crawler.Item{{"json": p}}, nil
	default:
		return nil, fmt.Errorf("解析结果类型异常: %T", parsed)
	}
}

var (
	urlFlag   = flag.String("url", "https://httpbin.org/json", "JSON API 地址")
	outFlag   = flag.String("output", "", "输出目录（默认 ./output/json-api）")
	concFlag  = flag.Int("concurrency", 3, "并发数")
	retryFlag = flag.Int("max-retry", 2, "失败重试次数")
	quietFlag = flag.Bool("quiet", false, "安静模式")
)

func main() {
	flag.Parse()

	out := *outFlag
	if out == "" {
		out = "./output/json-api"
	}

	eng := crawler.NewEngine(newJSONAPICrawler(), crawler.Options{
		Concurrency: *concFlag,
		MaxRetry:    *retryFlag,
		Timeout:     10 * time.Second,
		OutputDir:   out,
		Quiet:       *quietFlag,
	})

	urls := strings.Split(*urlFlag, ",")
	fmt.Printf("抓取 JSON API: %v（%s 示例）\n", urls, "json-api")
	result, err := eng.Run(context.Background(), urls)
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

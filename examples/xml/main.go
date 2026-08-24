// 示例：抓取 XML/RSS 源并导出 CSV（复合场景：XML 输入 → 结构化 item → CSV 输出）。
// 与 examples/rss（JSONL 输出）区分：本示例把抓取结果直接写成 CSV 表格。
// Parse 用 encoding/xml 同时支持 RSS 2.0 与 Atom；Fetch 支持远程 URL 与本地文件。
//
// 运行（在仓库根目录）：
//
//	go run ./examples/xml -url examples/xml/sample.xml              # 本地文件（离线演示）
//	go run ./examples/xml -url https://example.org/feed.xml         # 远程 RSS/Atom
//	go run ./examples/xml -url examples/xml/sample.xml -output out.csv
package main

import (
	"context"
	"encoding/csv"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"jciyuan-spider/crawler"
	"jciyuan-spider/internal/version"
)

// rssDoc RSS 2.0 文档结构。
type rssDoc struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title string `xml:"title"`
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

// atomDoc Atom 文档结构。
type atomDoc struct {
	XMLName xml.Name `xml:"feed"`
	Title   string   `xml:"title"`
	Entries []struct {
		Title string `xml:"title"`
		Link  struct {
			Href string `xml:"href,attr"`
		} `xml:"link"`
		Summary   string `xml:"summary"`
		Content   string `xml:"content"`
		Updated   string `xml:"updated"`
		Published string `xml:"published"`
	} `xml:"entry"`
}

// xmlCrawler 实现 crawler.Crawler：Fetch 支持本地文件与远程 URL。
type xmlCrawler struct {
	fetcher *crawler.HTTPFetcher
}

func newXMLCrawler() *xmlCrawler {
	return &xmlCrawler{fetcher: crawler.NewHTTPFetcher()}
}

func (c *xmlCrawler) Name() string { return "xml" }

func (c *xmlCrawler) Fetch(ctx context.Context, url string) (*crawler.Page, error) {
	u := strings.TrimPrefix(url, "file://")
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		body, err := os.ReadFile(u)
		if err != nil {
			return nil, fmt.Errorf("读取本地文件失败: %w", err)
		}
		return &crawler.Page{URL: url, StatusCode: 200, Body: body, Text: string(body)}, nil
	}
	return c.fetcher.Fetch(ctx, url)
}

// Parse 尝试 RSS 2.0，失败后尝试 Atom；两者都不是则报错。
func (c *xmlCrawler) Parse(_ context.Context, page *crawler.Page) (any, error) {
	var rss rssDoc
	if err := xml.Unmarshal(page.Body, &rss); err == nil && rss.Channel.Title != "" {
		return &rss, nil
	}
	var atom atomDoc
	if err := xml.Unmarshal(page.Body, &atom); err == nil && atom.Title != "" {
		return &atom, nil
	}
	return nil, fmt.Errorf("不是有效的 RSS 2.0 或 Atom 文档")
}

// Extract 输出统一字段的条目：title / link / description / pub_date。
func (c *xmlCrawler) Extract(_ context.Context, parsed any) ([]crawler.Item, error) {
	switch p := parsed.(type) {
	case *rssDoc:
		items := make([]crawler.Item, 0, len(p.Channel.Items))
		for _, it := range p.Channel.Items {
			items = append(items, crawler.Item{
				"title":       it.Title,
				"link":        it.Link,
				"description": it.Description,
				"pub_date":    it.PubDate,
			})
		}
		return items, nil
	case *atomDoc:
		items := make([]crawler.Item, 0, len(p.Entries))
		for _, e := range p.Entries {
			desc := e.Summary
			if desc == "" {
				desc = e.Content
			}
			date := e.Updated
			if date == "" {
				date = e.Published
			}
			items = append(items, crawler.Item{
				"title":       e.Title,
				"link":        e.Link.Href,
				"description": desc,
				"pub_date":    date,
			})
		}
		return items, nil
	default:
		return nil, fmt.Errorf("解析结果类型异常: %T", parsed)
	}
}

// ItemsToCSV 将 Item 列表写成 CSV（字段并集排序表头，encoding/csv 自动转义）。
// 与 examples/csv 保持一致的实现，便于示例间对照。
func ItemsToCSV(items []crawler.Item, w io.Writer) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	fieldSet := map[string]bool{}
	for _, it := range items {
		for k := range it {
			fieldSet[k] = true
		}
	}
	fields := make([]string, 0, len(fieldSet))
	for k := range fieldSet {
		fields = append(fields, k)
	}
	sort.Strings(fields)
	if len(fields) == 0 {
		return nil
	}

	if err := cw.Write(fields); err != nil {
		return err
	}
	for _, it := range items {
		row := make([]string, len(fields))
		for i, f := range fields {
			row[i] = csvString(it[f])
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return cw.Error()
}

func csvString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []string:
		return strings.Join(t, "; ")
	default:
		return fmt.Sprint(t)
	}
}

var (
	urlFlag     = flag.String("url", "", "XML/RSS/Atom 源：http(s):// 远程 URL 或本地文件路径")
	outFlag     = flag.String("output", "", "CSV 输出路径（默认 ./output/xml/feed.csv）")
	quietFlag   = flag.Bool("quiet", false, "安静模式")
	versionFlag = flag.Bool("version", false, "打印版本信息")
)

func main() {
	flag.Parse()
	if *versionFlag {
		fmt.Printf("jciyuan-spider xml example %s\n", version.Version)
		os.Exit(0)
	}
	if *urlFlag == "" {
		fmt.Fprintln(os.Stderr, "请通过 -url 指定 XML/RSS 源，例如：\n"+
			"  go run ./examples/xml -url examples/xml/sample.xml\n"+
			"  go run ./examples/xml -url https://example.org/feed.xml")
		os.Exit(2)
	}

	out := *outFlag
	if out == "" {
		out = "./output/xml/feed.csv"
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	result, err := crawler.NewEngine(newXMLCrawler(), crawler.Options{
		Concurrency: 1,
		MaxRetry:    2,
		Timeout:     10 * time.Second,
		Quiet:       *quietFlag,
	}).Run(context.Background(), []string{*urlFlag})
	if err != nil {
		fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 CSV 文件失败: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	if err := ItemsToCSV(result.Items, f); err != nil {
		fmt.Fprintf(os.Stderr, "导出 CSV 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("已导出 %d 条条目到 %s（成功=%d 失败=%d）\n",
		result.Stats.Items, out, result.Stats.Success, result.Stats.Fail)
	if len(result.Failures) > 0 {
		for _, fl := range result.Failures {
			fmt.Fprintf(os.Stderr, "失败 %s: %s\n", fl.URL, fl.Error)
		}
	}
}

// 示例：把 crawler Engine 的抓取结果导出为 CSV。
// 使用默认 HTTP 爬虫（crawler.NewHTTPCrawler）抓取，再调用 ItemsToCSV
// 将结果 Item 列表（map[string]any）转为 CSV：首行为表头（字段名排序），
// 逐行对齐写入，字符串中的逗号/引号/换行由 encoding/csv 自动转义。
//
// 运行（在仓库根目录）：
//
//	go run ./examples/csv -url https://example.com
//	go run ./examples/csv -url https://example.com,https://example.org -output ./output/csv/result.csv
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"jciyuan-spider/crawler"
)

var (
	urlFlag   = flag.String("url", "", "抓取目标 URL（多个用逗号分隔）")
	outFlag   = flag.String("output", "", "CSV 输出路径（默认 ./output/csv/result.csv）")
	concFlag  = flag.Int("concurrency", 3, "并发数")
	retryFlag = flag.Int("max-retry", 2, "失败重试次数")
	quietFlag = flag.Bool("quiet", false, "安静模式")
)

func main() {
	flag.Parse()
	if *urlFlag == "" {
		fmt.Fprintln(os.Stderr, "请通过 -url 指定抓取目标，例如：\n"+
			"  go run ./examples/csv -url https://example.com\n"+
			"  go run ./examples/csv -url https://example.com,https://example.org")
		os.Exit(2)
	}

	out := *outFlag
	if out == "" {
		out = "./output/csv/result.csv"
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	urls := strings.Split(*urlFlag, ",")
	result, err := run(urls, crawler.Options{
		Concurrency: *concFlag,
		MaxRetry:    *retryFlag,
		Timeout:     10 * time.Second,
		Quiet:       *quietFlag,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "抓取运行失败: %v\n", err)
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

	fmt.Printf("已导出 %d 条 Item 到 %s（成功=%d 失败=%d 重试=%d）\n",
		result.Stats.Items, out, result.Stats.Success, result.Stats.Fail, result.Stats.Retry)
}

// run 用默认 HTTP 爬虫抓取 urls 并返回结果（main 与测试复用）。
func run(urls []string, opts crawler.Options) (*crawler.Result, error) {
	return crawler.NewEngine(crawler.NewHTTPCrawler(), opts).Run(context.Background(), urls)
}

// ItemsToCSV 将 Item 列表写成 CSV：字段名为排序后的并集（首行为表头），
// 每行按同一字段顺序对齐；值经 csvString 归一化为字符串。
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

// csvString 将任意字段值归一化为 CSV 单元格字符串。
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

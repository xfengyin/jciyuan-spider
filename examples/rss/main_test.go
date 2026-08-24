package main

import (
	"context"
	"testing"

	"jciyuan-spider/crawler"
)

// TestRSSCrawlerLocalFile 用本地 sample.xml 走完整 Engine 链路（无需网络）。
func TestRSSCrawlerLocalFile(t *testing.T) {
	c := newRSSCrawler()
	res, err := crawler.NewEngine(c, crawler.Options{Quiet: true}).Run(context.Background(), []string{"sample.xml"})
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if res.Stats.Success != 1 || res.Stats.Fail != 0 || res.Stats.Items != 3 {
		t.Fatalf("统计异常: %+v", res.Stats)
	}
	got := map[string]string{}
	for _, it := range res.Items {
		got[it["title"].(string)] = it["link"].(string)
	}
	if got["框架发布 M1：通用 Go 爬虫框架"] == "" {
		t.Fatalf("未解析出第一条 item: %v", got)
	}
	if got["示例：抓取并解析 RSS/XML 源"] == "" {
		t.Fatalf("未解析出第三条 item: %v", got)
	}
}

// TestRSSCrawlerInvalidXML 非法 XML 应返回错误。
func TestRSSCrawlerInvalidXML(t *testing.T) {
	c := newRSSCrawler()
	res, err := crawler.NewEngine(c, crawler.Options{Quiet: true}).Run(context.Background(), []string{"not-a-feed.xml"})
	if err != nil {
		t.Fatalf("Run 不应返回引擎级错误: %v", err)
	}
	if res.Stats.Fail != 1 || len(res.Failures) != 1 {
		t.Fatalf("期望记录 1 条失败: %+v", res.Stats)
	}
}

// TestLocalPath 本地路径判定。
func TestLocalPath(t *testing.T) {
	if p, ok := localPath("sample.xml"); !ok || p != "sample.xml" {
		t.Fatalf("普通路径判定失败: %q %v", p, ok)
	}
	if p, ok := localPath("file://sample.xml"); !ok || p != "sample.xml" {
		t.Fatalf("file:// 前缀判定失败: %q %v", p, ok)
	}
	if _, ok := localPath("https://example.org/feed.xml"); ok {
		t.Fatal("远程 URL 不应判为本地路径")
	}
}

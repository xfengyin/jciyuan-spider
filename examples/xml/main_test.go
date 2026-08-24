package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jciyuan-spider/crawler"
)

const atomSample = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom 示例源</title>
  <entry>
    <title>Atom 条目一</title>
    <link href="https://example.org/atom/1"/>
    <summary>Atom 摘要一</summary>
    <updated>2025-09-20T10:00:00Z</updated>
  </entry>
  <entry>
    <title>Atom 条目二</title>
    <link href="https://example.org/atom/2"/>
    <content>Atom 正文二</content>
    <published>2025-09-21T10:00:00Z</published>
  </entry>
</feed>`

// writeTempFeed 把 feed 内容写入临时文件并返回路径。
func writeTempFeed(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "feed.xml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// runAndCSV 用本地文件跑完整链路并返回 CSV 文本。
func runAndCSV(t *testing.T, feedPath string) string {
	t.Helper()
	res, err := crawler.NewEngine(newXMLCrawler(), crawler.Options{Quiet: true}).Run(context.Background(), []string{feedPath})
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if res.Stats.Success != 1 || res.Stats.Fail != 0 {
		t.Fatalf("统计异常: %+v", res.Stats)
	}
	var sb strings.Builder
	if err := ItemsToCSV(res.Items, &sb); err != nil {
		t.Fatalf("ItemsToCSV 失败: %v", err)
	}
	return sb.String()
}

// TestXMLRSSLocalFile RSS 2.0 本地文件 → CSV（无需网络）。
func TestXMLRSSLocalFile(t *testing.T) {
	feed := writeTempFeed(t, `<?xml version="1.0"?>
<rss version="2.0"><channel>
<title>RSS 源</title>
<item><title>第一条</title><link>https://example.org/1</link><description>描述一</description><pubDate>Mon, 08 Sep 2025 10:00:00 GMT</pubDate></item>
<item><title>第二条, 带逗号</title><link>https://example.org/2</link><description>描述二</description></item>
</channel></rss>`)

	csvText := runAndCSV(t, feed)
	lines := strings.Split(strings.TrimSpace(csvText), "\n")
	if len(lines) != 3 {
		t.Fatalf("期望表头+2 行，实际 %d 行:\n%s", len(lines), csvText)
	}
	if !strings.Contains(lines[0], "title") || !strings.Contains(lines[0], "link") {
		t.Fatalf("表头异常: %s", lines[0])
	}
	// 逗号转义
	if !strings.Contains(lines[2], `"第二条, 带逗号"`) {
		t.Fatalf("逗号未转义: %s", lines[2])
	}
}

// TestXMLAtomLocalFile Atom 本地文件 → CSV。
func TestXMLAtomLocalFile(t *testing.T) {
	feed := writeTempFeed(t, atomSample)
	csvText := runAndCSV(t, feed)
	lines := strings.Split(strings.TrimSpace(csvText), "\n")
	if len(lines) != 3 {
		t.Fatalf("期望表头+2 行，实际 %d 行:\n%s", len(lines), csvText)
	}
	if !strings.Contains(csvText, "Atom 条目一") || !strings.Contains(csvText, "https://example.org/atom/1") {
		t.Fatalf("Atom 条目缺失:\n%s", csvText)
	}
	if !strings.Contains(csvText, "Atom 正文二") || !strings.Contains(csvText, "2025-09-21T10:00:00Z") {
		t.Fatalf("Atom 字段映射异常:\n%s", csvText)
	}
}

// TestXMLInvalid 非法 XML / 非 feed 文档应记录失败。
func TestXMLInvalid(t *testing.T) {
	feed := writeTempFeed(t, "<html><body>not a feed</body></html>")
	res, err := crawler.NewEngine(newXMLCrawler(), crawler.Options{Quiet: true}).Run(context.Background(), []string{feed})
	if err != nil {
		t.Fatalf("Run 不应返回引擎级错误: %v", err)
	}
	if res.Stats.Fail != 1 || len(res.Failures) != 1 {
		t.Fatalf("期望记录 1 条失败: %+v", res.Stats)
	}
}

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jciyuan-spider/crawler"
)

const sampleHTML = `<!doctype html>
<html><head><title>Demo &amp; Page</title></head>
<body>
<h1>Hello <strong>World</strong></h1>
<p>This is a <a href="https://example.org/link">link</a> and an <em>emphasis</em>.</p>
<img src="https://example.org/img.png" alt="an image">
<ul><li>first</li><li>second</li></ul>
<pre><code>code block</code></pre>
<blockquote>quoted text</blockquote>
</body></html>`

func TestHTMLToMarkdown(t *testing.T) {
	md := htmlToMarkdown(sampleHTML)

	cases := []struct {
		name string
		want string
	}{
		{"标题", "# Hello **World**"},
		{"链接", "[link](https://example.org/link)"},
		{"斜体", "*emphasis*"},
		{"图片", "![an image](https://example.org/img.png)"},
		{"列表项", "- first"},
		{"代码块", "```\ncode block\n```"},
		{"引用", "> quoted text"},
		{"实体解码", "Demo & Page"},
	}
	for _, c := range cases {
		if !strings.Contains(md, c.want) {
			t.Errorf("%s: 期望包含 %q，实际:\n%s", c.name, c.want, md)
		}
	}

	if strings.Contains(md, "<") && strings.Contains(md, ">") {
		t.Errorf("仍残留 HTML 标签:\n%s", md)
	}
}

func TestHTMLToMarkdownStripsOtherTags(t *testing.T) {
	md := htmlToMarkdown(`<div><span class="x">plain</span><script>bad()</script></div>`)
	if strings.Contains(md, "<div") || strings.Contains(md, "<span") || strings.Contains(md, "bad()") {
		t.Fatalf("应剥离 div/span/script: %q", md)
	}
	if !strings.Contains(md, "plain") {
		t.Fatalf("应保留文本: %q", md)
	}
}

func TestSlugName(t *testing.T) {
	if got := slugName("https://example.com/foo/bar.html"); got != "example.com-foo-bar.html" {
		t.Fatalf("slug 异常: %s", got)
	}
	if got := slugName("https://example.com"); got != "example.com" {
		t.Fatalf("根路径 slug 异常: %s", got)
	}
}

// TestMarkdownEndToEnd 本地 httptest 服务 → Engine 抓取 → .md 文件落盘（无需网络）。
func TestMarkdownEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html><body><h1>Local Title</h1><p>hello md</p></body></html>"))
	}))
	defer srv.Close()

	out := t.TempDir()
	result, err := crawler.NewEngine(crawler.NewHTTPCrawler(), crawler.Options{Quiet: true}).Run(context.Background(), []string{srv.URL})
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if result.Stats.Success != 1 || len(result.Items) != 1 {
		t.Fatalf("统计异常: %+v", result.Stats)
	}

	text, _ := result.Items[0]["text"].(string)
	name := slugName(srv.URL)
	path := filepath.Join(out, name+".md")
	if err := os.WriteFile(path, []byte(htmlToMarkdown(text)), 0o644); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	s := string(data)
	if !strings.Contains(s, "# Local Title") || !strings.Contains(s, "hello md") {
		t.Fatalf("MD 内容异常:\n%s", s)
	}
}

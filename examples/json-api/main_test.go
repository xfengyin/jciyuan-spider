package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"jciyuan-spider/crawler"
)

// sampleSlideshow 与 httpbin.org/json 结构一致，供本地 mock 服务返回。
const sampleSlideshow = `{
  "slideshow": {
    "author": "Yours Truly",
    "date": "date of publication",
    "slides": [
      { "title": "Wake up to WonderWidgets!", "type": "all" },
      { "title": "Overview", "type": "all" },
      { "title": "Why WonderWidgets are great", "type": "all" }
    ],
    "title": "Sample Slide Show"
  }
}`

// TestJSONAPIMockServer 本地 mock 服务 → Engine 抓取 → slide 展开为多条 item（无需网络）。
func TestJSONAPIMockServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleSlideshow))
	}))
	defer srv.Close()

	res, err := crawler.NewEngine(newJSONAPICrawler(), crawler.Options{Quiet: true}).Run(context.Background(), []string{srv.URL})
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if res.Stats.Success != 1 || res.Stats.Fail != 0 || res.Stats.Items != 3 {
		t.Fatalf("统计异常: %+v", res.Stats)
	}
	for _, it := range res.Items {
		if it["show_title"] != "Sample Slide Show" {
			t.Fatalf("show_title 异常: %+v", it)
		}
		if it["author"] != "Yours Truly" {
			t.Fatalf("author 异常: %+v", it)
		}
		if it["slide_title"] == "" || it["type"] == "" {
			t.Fatalf("slide 字段缺失: %+v", it)
		}
	}
}

// TestJSONAPIFallback 非 slideshow 结构回退为原始 map item。
func TestJSONAPIFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"hello": "world", "count": 42}`))
	}))
	defer srv.Close()

	res, err := crawler.NewEngine(newJSONAPICrawler(), crawler.Options{Quiet: true}).Run(context.Background(), []string{srv.URL})
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if res.Stats.Success != 1 || res.Stats.Items != 1 {
		t.Fatalf("统计异常: %+v", res.Stats)
	}
	raw, ok := res.Items[0]["json"].(map[string]any)
	if !ok || raw["hello"] != "world" {
		t.Fatalf("回退 item 异常: %+v", res.Items[0])
	}
}

// TestJSONAPIInvalid 非法 JSON 应记录失败。
func TestJSONAPIInvalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("this is not json"))
	}))
	defer srv.Close()

	res, err := crawler.NewEngine(newJSONAPICrawler(), crawler.Options{Quiet: true}).Run(context.Background(), []string{srv.URL})
	if err != nil {
		t.Fatalf("Run 不应返回引擎级错误: %v", err)
	}
	if res.Stats.Fail != 1 || len(res.Failures) != 1 {
		t.Fatalf("期望记录 1 条失败: %+v", res.Stats)
	}
}

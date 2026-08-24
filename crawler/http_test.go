package crawler

import (
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPFetcherGzipDecompress(t *testing.T) {
	// 回归：不手动设置 Accept-Encoding，交由传输层透明解压 gzip。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte("<h1>gzipped</h1>"))
		_ = gz.Close()
	}))
	defer srv.Close()

	page, err := NewHTTPFetcher().Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Fetch 失败: %v", err)
	}
	if page.Text != "<h1>gzipped</h1>" {
		t.Fatalf("gzip 未透明解压: %q", page.Text)
	}
}

func TestHTTPFetcherFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<h1>hello</h1>"))
	}))
	defer srv.Close()

	page, err := NewHTTPFetcher().Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Fetch 失败: %v", err)
	}
	if page.StatusCode != 200 || page.Text != "<h1>hello</h1>" {
		t.Fatalf("Page 内容异常: %+v", page)
	}
	if string(page.Body) != "<h1>hello</h1>" {
		t.Fatalf("Body 异常: %s", page.Body)
	}
}

func TestHTTPFetcherUserAgentAndHeaders(t *testing.T) {
	var gotUA, gotX string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotX = r.Header.Get("X-Custom")
	}))
	defer srv.Close()

	f := NewHTTPFetcher(HTTPOptions{
		UserAgent: "my-agent/1.0",
		Headers:   map[string]string{"X-Custom": "abc"},
	})
	if _, err := f.Fetch(context.Background(), srv.URL); err != nil {
		t.Fatalf("Fetch 失败: %v", err)
	}
	if gotUA != "my-agent/1.0" {
		t.Fatalf("UA 异常: %q", gotUA)
	}
	if gotX != "abc" {
		t.Fatalf("自定义头异常: %q", gotX)
	}
}

func TestHTTPFetcherStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := NewHTTPFetcher().Fetch(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("期望 500 报错")
	}
	var se *StatusError
	if !errors.As(err, &se) || se.Code != 500 {
		t.Fatalf("期望 *StatusError(500)，实际: %v", err)
	}
	if !IsStatusError(err) {
		t.Fatal("IsStatusError 应为 true")
	}
}

func TestHTTPFetcherMaxBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer srv.Close()

	f := NewHTTPFetcher(HTTPOptions{MaxBodySize: 5})
	if _, err := f.Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("期望超出体积上限报错")
	}
}

func TestHTTPFetcherTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	f := NewHTTPFetcher(HTTPOptions{Timeout: 50 * time.Millisecond})
	start := time.Now()
	if _, err := f.Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("期望超时报错")
	}
	if time.Since(start) > 1500*time.Millisecond {
		t.Fatalf("超时未生效: 耗时 %s", time.Since(start))
	}
}

func TestHTTPFetcherNoRedirect(t *testing.T) {
	redirected := &atomic.Bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/final" {
			redirected.Store(true)
			return
		}
		http.Redirect(w, r, "/final", http.StatusFound)
	}))
	defer srv.Close()

	// 默认跟随重定向
	page, err := NewHTTPFetcher().Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("跟随重定向失败: %v", err)
	}
	if page.StatusCode != 200 || !redirected.Load() {
		t.Fatalf("重定向未跟随: %+v", page)
	}

	// 关闭重定向
	redirected.Store(false)
	page, err = NewHTTPFetcher(HTTPOptions{FollowRedirects: false}).Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("不跟随重定向也应成功返回: %v", err)
	}
	if page.StatusCode != 302 || redirected.Load() {
		t.Fatalf("重定向应被关闭: %+v", page)
	}
}

func TestPackageFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pkg-fetch"))
	}))
	defer srv.Close()

	page, err := Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Fetch 失败: %v", err)
	}
	if page.Text != "pkg-fetch" {
		t.Fatalf("内容异常: %q", page.Text)
	}
}

func TestHTTPCrawlerFullRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("framework works"))
	}))
	defer srv.Close()

	c := NewHTTPCrawler()
	if c.Name() != "http" {
		t.Fatalf("名称异常: %s", c.Name())
	}
	res, err := NewEngine(c, Options{Quiet: true}).Run(context.Background(), []string{srv.URL + "/a", srv.URL + "/b"})
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if res.Stats.Success != 2 || res.Stats.Fail != 0 || res.Stats.Items != 2 {
		t.Fatalf("统计异常: %+v", res.Stats)
	}
	for _, it := range res.Items {
		if it["text"] != "framework works" {
			t.Fatalf("条目内容异常: %+v", it)
		}
	}
}

func TestHTTPCrawlerDefaultOpts(t *testing.T) {
	o := DefaultHTTPOptions()
	if o.Timeout != 30*time.Second || o.MaxBodySize != defaultHTTPMaxBody || !o.FollowRedirects {
		t.Fatalf("默认选项异常: %+v", o)
	}
	if !strings.HasPrefix(DefaultHTTPUserAgent, "jciyuan-spider") {
		t.Fatalf("默认 UA 异常: %s", DefaultHTTPUserAgent)
	}
}

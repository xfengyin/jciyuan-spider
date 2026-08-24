package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jciyuan-spider/crawler"
)

// TestItemsToCSV 校验表头、行对齐与 CSV 转义（逗号/引号）。
func TestItemsToCSV(t *testing.T) {
	items := []crawler.Item{
		{"title": "hello, world", "url": "https://a.example", "count": 1},
		{"title": `say "hi"`, "url": "https://b.example", "tags": []string{"x", "y"}},
	}
	var buf bytes.Buffer
	if err := ItemsToCSV(items, &buf); err != nil {
		t.Fatalf("ItemsToCSV 失败: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("期望 1 表头 + 2 行，实际 %d 行:\n%s", len(lines), buf.String())
	}
	if !strings.Contains(lines[0], "title") || !strings.Contains(lines[0], "url") || !strings.Contains(lines[0], "count") {
		t.Fatalf("表头异常: %s", lines[0])
	}
	// 逗号与引号应被 encoding/csv 转义
	if !strings.Contains(lines[1], `"hello, world"`) {
		t.Fatalf("逗号未转义: %s", lines[1])
	}
	if !strings.Contains(lines[2], `"say ""hi"""`) {
		t.Fatalf("引号未转义: %s", lines[2])
	}
	// []string 值应合并为 "; " 分隔
	if !strings.Contains(lines[2], "x; y") {
		t.Fatalf("切片值未合并: %s", lines[2])
	}
}

// TestItemsToCSVEmpty 空列表输出空文件（仅可能无表头）。
func TestItemsToCSVEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := ItemsToCSV(nil, &buf); err != nil {
		t.Fatalf("空列表应无错误: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("空列表应无输出: %q", buf.String())
	}
}

// TestRunEndToEnd 本地 httptest 服务 → Engine 抓取 → CSV 落盘（无需网络）。
func TestRunEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("csv example page"))
	}))
	defer srv.Close()

	result, err := run([]string{srv.URL}, crawler.Options{Quiet: true})
	if err != nil {
		t.Fatalf("run 失败: %v", err)
	}
	if result.Stats.Success != 1 || len(result.Items) != 1 {
		t.Fatalf("统计异常: %+v", result.Stats)
	}

	path := filepath.Join(t.TempDir(), "result.csv")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = ItemsToCSV(result.Items, f)
	_ = f.Close()
	if err != nil {
		t.Fatalf("ItemsToCSV 失败: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "text") || !strings.Contains(string(data), "csv example page") {
		t.Fatalf("CSV 内容异常:\n%s", data)
	}
}

# Example: demo

最小的「通用爬虫」示例：纯标准库（`net/http`）+ goquery 实现 `crawler.Crawler`，
通过 YAML 配置描述抓取任务（起始 URL 列表 + CSS/正则抽取规则），完全与站点解耦。

## 运行

```bash
# 在仓库根目录
go run ./examples/demo -config examples/demo/config.yaml
go run ./examples/demo -config examples/demo/config.yaml -url https://httpbin.org/html -output ./output/demo
```

## 配置

见 [`config.yaml`](./config.yaml)。核心结构：

```yaml
start_urls: ["https://example.com"]
rules:
  - field: title          # 抽取字段名
    selector:
      type: css           # css | regex
      value: "h1"         # CSS 选择器或正则表达式
    multiple: false       # true 返回列表，false 取首个匹配
concurrency: 2
max_retry: 2
timeout: 10s
delay: 500ms
output_dir: ./output/demo
```

## 实现要点

- `Fetch`：`net/http` + 超时 + UA，仅标准库。
- `Parse`：保留 HTML 原文，直接透传。
- `Extract`：按配置规则用 goquery（CSS）或 `regexp` 抽取字段。
- 由通用 `crawler.Engine` 统一调度并发、重试、限速与 JSONL 输出。

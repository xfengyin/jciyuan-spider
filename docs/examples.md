# 示例详解

所有示例均在仓库根目录运行；输出默认写入 `./output/<示例名>/items.jsonl`（JSON Lines）。

## 示例总览

| 示例 | 说明 | 运行 |
|------|------|------|
| [`examples/demo`](../examples/demo) | 配置驱动通用 HTML 爬虫：YAML 定义起始 URL + CSS/正则抽取规则，纯标准库 + goquery | `go run ./examples/demo -config examples/demo/config.yaml` |
| [`examples/jciyuan`](../examples/jciyuan) | 复用内部企业级 Fetcher/Parser 的框架化实现，抓取 jciyuan.com 动漫详情页 | `go run ./examples/jciyuan -id 37439` |
| [`examples/rss`](../examples/rss) | 用框架抓取并解析 RSS/XML 源，支持远程 URL 与本地文件（离线演示） | `go run ./examples/rss -url examples/rss/sample.xml` |
| [`examples/csv`](../examples/csv) | 把 crawler Engine 抓取结果导出为 CSV | `go run ./examples/csv -url https://example.com` |

## 1. examples/demo —— 配置驱动通用爬虫

演示「任意站点」的最小接入：`Fetch` 用 `net/http`，`Extract` 用配置规则。

```yaml
# examples/demo/config.yaml
start_urls: ["https://example.com"]
rules:
  - field: title
    selector: { type: css, value: "h1" }
  - field: paragraph
    selector: { type: css, value: "p" }
  - field: links
    selector: { type: css, value: "a", attr: href }
    multiple: true
concurrency: 2
max_retry: 2
timeout: 10s
delay: 500ms
output_dir: ./output/demo
```

- 选择器 `type` 支持 `css`（goquery）与 `regex`。
- `multiple: true` 返回列表，否则取首个匹配。

```bash
go run ./examples/demo -config examples/demo/config.yaml
go run ./examples/demo -config examples/demo/config.yaml -url https://httpbin.org/html
```

## 2. examples/jciyuan —— 企业级组件适配

演示「单一站点爬虫」如何接入「通用框架」：把内部 `fetcher.Fetcher`（中间件链 + 抗反爬）
与 `parser.Parser`（配置驱动 HTML Pipeline）包装为 `Crawler` 接口，并通过
`crawler.Register("jciyuan", ...)` 注册、`crawler.Build` 构建。

```bash
go run ./examples/jciyuan -id 37439
go run ./examples/jciyuan -url https://www.jciyuan.com/acgdetail/37439.html -output ./output/jciyuan
```

| 参数 | 默认 | 说明 |
|------|------|------|
| `-config` | config/config.yaml | 配置文件路径（缺失时回退内置默认配置） |
| `-id` | 0 | 动漫 ID，按配置模板生成详情页 URL |
| `-url` | - | 直接指定详情页 URL（优先于 `-id`） |
| `-output` | ./output/jciyuan | 输出目录 |
| `-concurrency` | 3 | Engine 并发数 |
| `-max-retry` | 3 | 失败重试次数 |
| `-quiet` | false | 安静模式 |

## 3. examples/rss —— RSS/XML 源解析

演示用框架解析订阅源：`Fetch` 支持 `http(s)://` 远程 URL 与本地文件路径（离线演示/测试），
`Parse` 用 `encoding/xml` 解析 RSS 2.0，`Extract` 输出 `{title, link, description, pub_date}` 条目。

```bash
# 本地文件离线演示（仓库内置 sample.xml，3 条 item）
go run ./examples/rss -url examples/rss/sample.xml

# 远程 RSS 源
go run ./examples/rss -url https://example.org/feed.xml

# httpbin 的 XML 响应
go run ./examples/rss -url https://httpbin.org/xml
```

| 参数 | 默认 | 说明 |
|------|------|------|
| `-url` | - | RSS/XML 源：http(s):// 远程 URL 或本地文件路径（必填） |
| `-output` | ./output/rss | 输出目录 |
| `-quiet` | false | 安静模式 |

实现要点：

- `localPath()` 识别本地文件（含 `file://` 前缀），直接 `os.ReadFile`，全程无网络。
- RSS 结构体仅覆盖常用字段（channel.title/link/description + item.title/link/description/pubDate）。
- 非 RSS 2.0 / 非法 XML 会返回明确错误并记录到 `Result.Failures`。

## 4. examples/csv —— 导出抓取结果为 CSV

演示把 `crawler.Engine` 的抓取结果（`[]crawler.Item`）转成 CSV 落地：
`run()` 用默认 HTTP 爬虫抓取，`ItemsToCSV()` 生成 CSV（字段并集排序作表头，
`encoding/csv` 自动处理逗号/引号/换行转义）。

```bash
# 单站点
go run ./examples/csv -url https://example.com

# 多站点（逗号分隔）+ 自定义输出路径
go run ./examples/csv -url https://example.com,https://example.org -output ./output/csv/result.csv
```

| 参数 | 默认 | 说明 |
|------|------|------|
| `-url` | - | 抓取目标 URL，多个用逗号分隔（必填） |
| `-output` | ./output/csv/result.csv | CSV 输出路径 |
| `-concurrency` | 3 | Engine 并发数 |
| `-max-retry` | 2 | 失败重试次数 |
| `-quiet` | false | 安静模式 |

实现要点：

- `run()` 封装「Engine 抓取」；`ItemsToCSV()` 独立成函数便于复用与单测。
- 表头 = 所有 Item 字段名的排序并集；缺失字段留空，`[]string` 值以 `; ` 合并。
- 嵌套结构（map/[]Item）以 `fmt.Sprint` 兜底，适合扁平化结果。

## 输出格式

所有示例的输出均为 JSON Lines（每行一条 Item），例如：

```json
{"title":"框架发布 M1：通用 Go 爬虫框架","link":"https://example.org/posts/framework-m1","description":"从单一站点爬虫转型为通用 Crawler 接口 + Engine 调度。","pub_date":"Mon, 18 Aug 2025 10:00:00 GMT"}
```

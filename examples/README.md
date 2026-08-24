# Examples

本目录演示如何基于 `crawler` 框架包实现具体爬虫。

| 示例 | 说明 | 运行方式 |
|------|------|----------|
| [`demo`](./demo) | 纯标准库 + goquery 的通用 HTML 爬虫，配置驱动（CSS/正则选择器），抓取任意站点 | `go run ./examples/demo -config examples/demo/config.yaml` |
| [`jciyuan`](./jciyuan) | 复用仓库内置的企业级 Fetcher/Parser 组件，抓取 [jciyuan.com](https://www.jciyuan.com) 动漫详情页 | `go run ./examples/jciyuan -id 37439` |
| [`rss`](./rss) | 抓取并解析 RSS/XML 源，支持远程 URL 与本地文件（离线演示） | `go run ./examples/rss -url examples/rss/sample.xml` |
| [`csv`](./csv) | 把 crawler Engine 抓取结果导出为 CSV（字段并集表头 + 自动转义） | `go run ./examples/csv -url https://example.com` |

所有示例均在仓库根目录下运行。输出默认为 `./output/` 下的 JSON Lines 文件（`items.jsonl`）。
各示例详解见 [docs/examples.md](../docs/examples.md)。

# Examples

本目录演示如何基于 `crawler` 框架包实现具体爬虫。

| 示例 | 说明 | 运行方式 |
|------|------|----------|
| [`jciyuan`](./jciyuan) | 复用仓库内置的企业级 Fetcher/Parser 组件，抓取 [jciyuan.com](https://www.jciyuan.com) 动漫详情页 | `go run ./examples/jciyuan -id 37439` |
| [`demo`](./demo) | 纯标准库 + goquery 的通用 HTML 爬虫，配置驱动（CSS/正则选择器），抓取任意站点 | `go run ./examples/demo -config examples/demo/config.yaml` |

所有示例均在仓库根目录下运行。输出默认为 `./output/` 下的 JSON Lines 文件（`items.jsonl`）。

# jciyuan-spider 文档站

框架文档从 README 拆分而来；README 保留速览，深入内容见本目录。

| 文档 | 内容 |
|------|------|
| [getting-started.md](getting-started.md) | 快速开始：三行代码抓取、运行内置示例、自定义爬虫、配置与命令行 |
| [architecture.md](architecture.md) | 架构设计：Crawler 接口、Engine、SPI、默认实现、内部企业级组件、扩展点 |
| [examples.md](examples.md) | 示例详解：`examples/demo` / `examples/jciyuan` / `examples/rss` |

## 速览

- **定位**：轻量通用 Go 爬虫框架 —— `Crawler` 接口（`Fetch → Parse → Extract`）+ `Engine` 调度。
- **公开 API**：`crawler` 包（[GoDoc](https://pkg.go.dev/jciyuan-spider/crawler)），仅依赖标准库。
- **内置默认实现**：`HTTPFetcher` / `HTTPCrawler` / `crawler.Fetch`。
- **示例**：`examples/demo`（配置驱动通用爬虫）、`examples/jciyuan`（企业级组件适配）、
  `examples/rss`（RSS/XML 解析）、`examples/csv`（CSV 导出）、`examples/markdown`（HTML→Markdown）、
  `examples/json-api`（JSON API 抓取）。

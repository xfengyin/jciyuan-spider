# jciyuan-spider - 轻量通用 Go 爬虫框架

<p align="center">
  <strong>Go</strong> · <strong>Generic Crawler Interface</strong> · <strong>Engine</strong> · <strong>SPI Plugins</strong>
  <br/><br/>
  <a href="https://pkg.go.dev/github.com/xfengyin/jciyuan-spider/crawler"><img src="https://pkg.go.dev/badge/github.com/xfengyin/jciyuan-spider/crawler.svg" alt="Go Reference"></a>
  <a href="https://github.com/xfengyin/jciyuan-spider/releases"><img src="https://img.shields.io/github/v/release/xfengyin/jciyuan-spider?style=flat-square" alt="Release"></a>
</p>

---

**jciyuan-spider** 是一个轻量通用的 Go 爬虫框架：定义 `Crawler` 接口（`Fetch → Parse → Extract`），
由通用 `Engine` 统一调度并发、重试、限速与结果输出。任何站点/数据源只需实现该接口即可接入框架；
仓库内置的企业级组件（Fetcher 中间件链、配置驱动 HTML 解析器、多存储后端）作为参考实现，
原有 [jciyuan.com](https://www.jciyuan.com) 动漫爬虫保留为 [Example](#examples-示例)。

> M1 里程碑：从「单一站点爬虫」转型为「通用爬虫框架」——新增公开框架包 [`crawler/`](crawler)，原功能保持可用。
>
> M2 里程碑：框架工程化与可发现性——CI（vet/build/test）、默认 HTTP 抓取实现
> （`HTTPFetcher` / `HTTPCrawler` / `crawler.Fetch`）、3 行快速开始与框架能力表格。
>
> M3 里程碑：文档站（[`docs/`](docs/README.md)）+ RSS 示例（[`examples/rss`](examples/rss)）+ GoDoc 完善。
>
> M4 里程碑：Release 流水线（v* 标签 → go build 矩阵 → softprops 发布，权限最小化）、
> pkg.go.dev / Release badge、CSV 导出示例（[`examples/csv`](examples/csv)）。
>
> M5 里程碑：Markdown 导出示例（[`examples/markdown`](examples/markdown)）、文档完善，
> v0.2.0 发布准备（当前 tag：v0.1.0）。
>
> M6 里程碑：JSON API 示例（[`examples/json-api`](examples/json-api)）、文档同步、
> [CHANGELOG](CHANGELOG.md) v0.3.0 候选条目（当前 tag：v0.2.0）。
>
> M7 里程碑：社区推广包（[`docs/PROMOTION.md`](docs/PROMOTION.md)）、
> XML/RSS→CSV 复合示例（[`examples/xml`](examples/xml)）（当前 tag：v0.3.0）。
>
> M8 里程碑：根目录 [Makefile](Makefile)（build/test/vet/fmt/clean）、全部示例 CLI
> 支持 `--version`、[CHANGELOG](CHANGELOG.md) v0.5.0 候选条目（当前 tag：v0.4.0）。

## 功能特性

| 特性 | 描述 |
|------|------|
| 🧩 **通用 Crawler 接口** | `Fetch / Parse / Extract` 三阶段抽象，任何站点实现即接入 |
| ⚙️ **通用 Engine** | 并发 Worker、失败重试、请求间隔限速、超时控制、JSON Lines 输出 |
| 🔌 **SPI 注册机制** | `crawler.Register` / `crawler.Build`，按名称装配爬虫实现 |
| 🎯 **示例开箱即用** | [`examples/demo`](examples/demo)、[`examples/jciyuan`](examples/jciyuan)、[`examples/rss`](examples/rss) |
| 🧱 **企业级组件（内部）** | Fetcher 中间件链（trace/metrics/限流/重试/熔断/代理轮换）、抗反爬、多存储后端 |
| 📦 **多存储后端** | JSON（默认）、SQLite、MySQL、S3，外加内存缓存装饰器 |
| 🔄 **断点续爬** | 爬取状态持久化，中断后可恢复；支持增量合并保留已抓取的 M3U8 |
| 📊 **可观测性** | 内存 / Prometheus 指标、/healthz 健康检查、traceId 全链路日志 |

## 文档站

详细文档已拆分至 [`docs/`](docs/README.md)：

| 文档 | 内容 |
|------|------|
| [快速开始](docs/getting-started.md) | 三行代码抓取、运行内置示例、自定义爬虫、配置与命令行 |
| [架构设计](docs/architecture.md) | Crawler 接口、Engine、SPI、默认实现、内部组件、扩展点 |
| [示例详解](docs/examples.md) | demo / jciyuan / rss / csv / markdown / json-api / xml |
| [社区推广包](docs/PROMOTION.md) | Awesome / HN / Reddit / 中文社区文案 + tracking 表 |

公开 API 参考 [pkg.go.dev](https://pkg.go.dev/jciyuan-spider/crawler)。

## 快速开始

### 三行代码抓取任意页面

```go
ctx := context.Background()
result, _ := crawler.NewEngine(crawler.NewHTTPCrawler(), crawler.Options{}).Run(ctx, []string{"https://example.com"})
fmt.Println(result.Items[0]["text"]) // 输出页面正文
```

更简的单行抓取：`page, _ := crawler.Fetch(ctx, "https://example.com")`。

### 运行内置示例

```bash
git clone https://github.com/xfengyin/jciyuan-spider.git
cd jciyuan-spider
go run ./examples/demo -config examples/demo/config.yaml      # 通用 HTML 爬虫
go run ./examples/jciyuan -id 37439                           # jciyuan 动漫爬虫
go run ./examples/rss -url examples/rss/sample.xml            # RSS/XML 源（本地离线演示）
```

完整接入方式（自定义 Crawler、CLI、配置）见 [docs/getting-started.md](docs/getting-started.md)。

## Examples 示例

| 示例 | 说明 | 运行 |
|------|------|------|
| [`examples/demo`](examples/demo) | 配置驱动通用 HTML 爬虫：YAML 定义起始 URL + CSS/正则抽取规则 | `go run ./examples/demo -config examples/demo/config.yaml` |
| [`examples/jciyuan`](examples/jciyuan) | 复用内置企业级 Fetcher/Parser 组件抓取 jciyuan.com 动漫详情页 | `go run ./examples/jciyuan -id 37439` |
| [`examples/rss`](examples/rss) | 用框架抓取并解析 RSS/XML 源，支持远程 URL 与本地文件 | `go run ./examples/rss -url examples/rss/sample.xml` |
| [`examples/csv`](examples/csv) | 把 crawler Engine 抓取结果导出为 CSV（字段并集表头 + 自动转义） | `go run ./examples/csv -url https://example.com` |
| [`examples/markdown`](examples/markdown) | 抓取 HTML 并导出 Markdown 文件（仅标准库的轻量 HTML→MD 转换） | `go run ./examples/markdown -url https://example.com` |
| [`examples/json-api`](examples/json-api) | 抓取 JSON API 并输出结构化 item（JSON 数组展开为多条） | `go run ./examples/json-api -url https://httpbin.org/json` |
| [`examples/xml`](examples/xml) | 抓取 XML/RSS/Atom 源并导出 CSV（复合场景：XML→结构化 item→CSV） | `go run ./examples/xml -url examples/xml/sample.xml` |

## 项目结构

```
jciyuan-spider/
├── main.go                     # 原有 CLI 入口（jciyuan 动漫爬虫，保持兼容）
├── crawler/                    # 公开框架包：Crawler 接口 + Engine + SPI + 默认 HTTP 实现
├── config/config.yaml          # 默认配置
├── examples/
│   ├── demo/                   # 配置驱动通用 HTML 爬虫
│   ├── jciyuan/                # 企业级组件 → 框架接口适配
│   └── rss/                    # RSS/XML 源抓取解析
├── docs/                       # 文档站（快速开始/架构/示例）
└── internal/                   # 内部企业级组件（参考实现）
```

## 开发

推荐使用根目录 [`Makefile`](Makefile) 统一命令：

```bash
make build          # 编译根 CLI 到 ./jciyuan-spider（VERSION 可覆盖：make build VERSION=v0.5.0）
make test           # 运行全部单元测试
make vet            # 静态检查
make fmt            # gofmt 格式化
make clean          # 清理构建产物
make run-demo       # 运行任一示例（run-jciyuan / run-rss / run-csv / run-markdown / run-json-api / run-xml）
```

等价的原生命令：

```bash
go build ./...
go vet ./...
go test ./...
```

所有示例 CLI 均支持 `--version` 统一输出版本（如 `jciyuan-spider demo example 0.5.0`）。

## 注意事项

⚠️ **免责声明**
- 本工具仅供学习研究使用
- 请遵守目标网站的 robots.txt 和使用条款
- 爬取行为需自行承担法律责任
- 建议设置合理的请求间隔（>=1000ms）

## License

MIT License - see [LICENSE](LICENSE) for details.

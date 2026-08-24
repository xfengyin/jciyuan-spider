# 架构设计

## 1. 设计目标

- **通用**：任何站点/数据源只需实现 `Crawler` 接口即可接入框架，由 `Engine` 统一调度。
- **轻量**：公开框架包 `crawler/` 仅依赖标准库，易于阅读、审计与嵌入。
- **可扩展**：`Register` / `Build` 按名称装配实现；三阶段（Fetch/Parse/Extract）可独立替换。
- **兼容**：原有 jciyuan 动漫爬虫 CLI 与内部企业级组件保持可用，作为参考实现与示例。

## 2. 核心抽象：Crawler 接口

```go
type Crawler interface {
	Name() string                                        // 爬虫名称（注册/日志/配置识别）
	Fetch(ctx context.Context, url string) (*Page, error) // 抓取原始页面
	Parse(ctx context.Context, page *Page) (any, error)   // 解析为中间表示（DOM/文档/对象）
	Extract(ctx context.Context, parsed any) ([]Item, error) // 抽取结构化条目，可返回多条
}
```

### 数据流

```
  URL ──Fetch──▶ Page ──Parse──▶ any(中间表示) ──Extract──▶ []Item
                  │                │                        │
             原始页面            DOM/文档/对象            结构化结果
```

三个阶段相互独立：可只替换 Fetch（换代理/浏览器内核）、只替换 Parse（换 DOM 引擎）、
只替换 Extract（换字段规则），互不影响。

### 类型

- **Page**：一次抓取的原始页面，含 `URL / StatusCode / Headers / Body / Text / Meta`。
- **Item**：`map[string]any`，一条结构化抽取结果。

## 3. Engine 调度器

```go
eng := crawler.NewEngine(c, crawler.Options{
	Concurrency: 4,                       // 并发 Worker 数（默认 1）
	Delay:       500 * time.Millisecond,  // 每次 Fetch 前请求间隔（限速）
	MaxRetry:    2,                       // 失败重试次数（默认 0）
	Timeout:     10 * time.Second,        // 单次 Fetch 超时（默认 30s）
	OutputDir:   "./output",              // 结果目录（JSON Lines: items.jsonl）
	Quiet:       false,                   // 关闭进度日志
})
result, err := eng.Run(ctx, urls)        // *Result{Stats, Items, Failures}
```

行为约定：

- 固定数量 Worker 从任务队列取 URL，逐个执行 `Fetch → Parse → Extract`。
- 失败按指数退避重试（500ms × 次数）；单个 URL 失败不中断整个运行。
- 每个 URL 的失败记录在 `Result.Failures`，统计在 `Result.Stats`（total/success/fail/retry/items/耗时）。
- `OutputDir` 非空时，每条 Item 追加写入 `items.jsonl`（JSON Lines）。
- 引擎级错误（如 nil Crawler、输出目录不可写）才返回非 nil error。

## 4. SPI 注册机制

```go
crawler.Register("my", func(cfg map[string]any) (crawler.Crawler, error) {
	return &myCrawler{}, nil
})
c, err := crawler.Build("my", cfg) // 按名称构建；未注册时返回错误并列出可用名称
```

`Register` 适合在 `init()` 中调用；`Build` 支持配置驱动装配。

## 5. 默认 HTTP 实现

[crawler/http.go](../crawler/http.go) 提供仅依赖标准库的默认实现：

| 类型 | 说明 |
|------|------|
| `HTTPFetcher` | 只实现 Fetch：超时、UA、自定义请求头、响应体上限、重定向开关、`*StatusError` 状态错误、gzip 透明解压 |
| `HTTPCrawler` | 完整 Crawler：Fetch 用 HTTPFetcher，Parse 透传原文，Extract 输出 `{text}` 条目 |
| `crawler.Fetch` | 包级一行式便捷函数（共享默认抓取器） |
| `HTTPOptions` | 配置结构（Timeout / UserAgent / Headers / MaxBodySize / FollowRedirects） |

```go
page, err := crawler.NewHTTPFetcher(crawler.HTTPOptions{
	Timeout: 10 * time.Second, UserAgent: "my-agent/1.0",
}).Fetch(ctx, "https://example.com")
```

需要代理池、中间件链、反爬等企业级能力时，参考 [examples/jciyuan](../examples/jciyuan)
换用仓库内置的 `internal/fetcher`。

## 6. 内部企业级组件（internal/，参考实现）

| 包 | 职责 |
|----|------|
| `internal/fetcher` | Fetcher 接口 + SPI；`http` 实现（连接池/白名单/robots/UA/gzip）；`middleware`（trace/metrics/logging/rate_limit/retry/circuit_breaker/proxy_rotate） |
| `internal/parser` | Parser 接口 + SPI；`extractor`（CSS/XPath/Regex）、`processor`（clean_text/trim/split/lower 等）、`html` 配置驱动 Pipeline |
| `internal/storage` | Storage 接口 + SPI：json / sqlite / mysql / s3 / memory 缓存装饰器 |
| `internal/spider` | 核心编排：任务调度、状态机、增量合并、M3U8 抓取 |
| `internal/worker` | goroutine 池；`internal/resume` 断点续爬；`internal/metrics`、`internal/health`、`internal/logger`、`internal/di` 等 |

> `internal/` 包只能在本仓库内引用；外部使用者应将其中模式复制到自己的代码，或通过
> `examples/jciyuan` 的适配方式接入公开框架包。

## 7. 项目结构

```
jciyuan-spider/
├── main.go                     # 原有 CLI 入口（jciyuan 动漫爬虫，保持兼容）
├── crawler/                    # 公开框架包：Crawler 接口 + Engine + SPI + 默认 HTTP 实现
├── config/config.yaml          # 默认配置
├── examples/
│   ├── demo/                   # 配置驱动通用 HTML 爬虫
│   ├── jciyuan/                # 企业级组件 → 框架接口适配
│   └── rss/                    # RSS/XML 源抓取解析
├── docs/                       # 文档站（本目录）
└── internal/                   # 内部企业级组件（参考实现）
```

## 8. 扩展点

| 场景 | 做法 |
|------|------|
| 新站点 | 实现 `Crawler`（复用 `HTTPFetcher` 或自写 Fetch），注册后由 Engine 调度 |
| 换传输层 | 替换 Fetch（如 Selenium/Playwright 渲染），接口不变 |
| 换解析引擎 | 替换 Parse/Extract（如 goquery、encoding/json、encoding/xml） |
| 加存储 | 自定义 Item 落盘方式，或用 `OutputDir` 的 JSONL |
| 企业级能力 | 参考 `internal/fetcher/middleware` 在 Fetch 内组合中间件 |

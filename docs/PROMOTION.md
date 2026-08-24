# 社区推广包（PROMOTION）

发布 v0.1.0+ 后的社区推广物料：面向英文社区（Awesome / Hacker News / Reddit）与
中文社区（掘金 / 知乎 / V2EX / 公众号）的投稿文案与发布 tracking。

## 1. 项目定位与核心卖点

**一句话定位**：jciyuan-spider 是一个仅依赖标准库的轻量通用 Go 爬虫框架——实现
`Crawler` 接口（`Fetch → Parse → Extract`）即可接入，由 `Engine` 统一调度并发、
重试、限速与输出。

**核心卖点（对社区）**

| # | 卖点 | 对应证据 |
|---|------|----------|
| 1 | **极简 API**：3 行代码跑通抓取 | `crawler.NewEngine(crawler.NewHTTPCrawler(), …)` |
| 2 | **零第三方依赖**：公开框架包仅用标准库 | `crawler/` 包 import 检查 |
| 3 | **开箱即用示例**：demo / jciyuan / rss / csv / markdown / json-api / xml 共 7 个 | `examples/` |
| 4 | **配置驱动**：YAML 规则抓任意站点 | `examples/demo` |
| 5 | **工程完备**：CI、Release 流水线、GoDoc、文档站 | `.github/`、`docs/` |
| 6 | **真实落地**：源自生产级动漫爬虫（中间件链/抗反爬/多存储），非 toy project | `internal/` |

## 2. Awesome 投稿文案

### 2.1 Awesome Go（英文）

> **Title**: jciyuan-spider — A lightweight general-purpose Go crawler framework (stdlib-only)
>
> **Body**:
> [jciyuan-spider](https://github.com/xfengyin/jciyuan-spider) is a lightweight,
> general-purpose web crawler framework in Go. Implement the `Crawler` interface
> (`Fetch → Parse → Extract`) and the built-in `Engine` handles concurrency, retries,
> rate limiting, timeouts and JSON Lines output for you.
>
> - Public `crawler` package depends **only on the standard library**.
> - 3-line quick start: `crawler.NewEngine(crawler.NewHTTPCrawler(), …)`.
> - 7 runnable examples: config-driven HTML, jciyuan anime site, RSS/XML→CSV, JSON API,
>   HTML→Markdown, CSV export.
> - Fully documented (docs site, GoDoc) with CI + release pipeline.
>
> Links: [GitHub](https://github.com/xfengyin/jciyuan-spider) ·
> [GoDoc](https://pkg.go.dev/github.com/xfengyin/jciyuan-spider/crawler)

### 2.2 中文 Awesome 列表（如 awesome-go-cn 类）

> **标题**：jciyuan-spider — 轻量通用 Go 爬虫框架（仅标准库）
>
> **正文**：一个轻量、通用的 Go 爬虫框架：实现 `Crawler` 接口（`Fetch/Parse/Extract`），
> 内置 `Engine` 自动处理并发、重试、限速、超时与 JSON Lines 输出。公开框架包零第三方依赖；
> 附带 7 个可运行示例（配置驱动 HTML、动漫站点、RSS/XML→CSV、JSON API、HTML→Markdown、
> CSV 导出）；文档站 + GoDoc + CI + Release 流水线齐备。

## 3. Hacker News 帖子草稿（英文）

> **Title**: Show HN: A stdlib-only Go crawler framework — 3 lines to crawl anything
>
> **Text**:
> I built [jciyuan-spider](https://github.com/xfengyin/jciyuan-spider), a lightweight
> general-purpose crawler framework in Go.
>
> The core idea: you implement a `Crawler` interface with three methods
> (`Fetch → Parse → Extract`), and the built-in `Engine` gives you concurrency,
> retries with backoff, rate limiting, timeouts and JSON Lines output — for free.
>
> ```go
> result, _ := crawler.NewEngine(crawler.NewHTTPCrawler(), crawler.Options{}).
>     Run(ctx, []string{"https://example.com"})
> fmt.Println(result.Items[0]["text"])
> ```
>
> Why I like it: the public package has **zero third-party dependencies** (stdlib only),
> it grew out of a production anime-site crawler (middleware chain, anti-crawl, multiple
> storage backends live under `internal/`), and it ships with 7 runnable examples —
> config-driven HTML scraping, RSS/Atom → CSV, JSON API → items, HTML → Markdown, CSV export.
>
> Docs: pkg.go.dev · docs/ (architecture, getting-started, examples) · CI + release pipeline.
>
> Feedback welcome! Especially on the interface design and what examples you'd want next.

## 4. Reddit 帖子草稿（英文）

### r/golang 自荐帖

> **Title**: [P] jciyuan-spider — stdlib-only Go crawler framework with 7 examples
>
> **Body**:
> I wrote a small crawler framework while productionizing an anime-site scraper, and
> extracted the generic part into a public package that only uses the standard library.
>
> - `Crawler` interface: `Fetch(ctx, url) (*Page, error)`, `Parse(...)`, `Extract(...)`
> - `Engine`: worker pool, retry + backoff, per-request delay, timeout, JSONL output,
>   per-URL failure isolation
> - Default impl: `HTTPFetcher` / `HTTPCrawler` / one-liner `crawler.Fetch`
> - 7 examples: demo (config-driven CSS/regex), jciyuan anime site, RSS/Atom→CSV,
>   JSON API, HTML→Markdown, CSV export
>
> Questions for the community: would you prefer a config-file-first framework (like
> scrapy-style pipelines) vs. code-first (this one)? What's the most useful example
> you'd like to see?

## 5. 中文社区文案

### 掘金（短版）

> **标题**：写了一个零依赖的通用 Go 爬虫框架：3 行代码爬任意站点
>
> **正文**：最近把生产环境的动漫爬虫重构成通用框架并开源。核心就一个接口：
> `Fetch / Parse / Extract`，剩下的并发、重试、限速、超时、JSONL 输出全交给 Engine。
> 公开包只依赖标准库；内置默认 HTTP 实现，3 行代码即可抓取：
> `crawler.NewEngine(crawler.NewHTTPCrawler(), …).Run(ctx, []string{url})`。
> 仓库带 7 个可运行示例（配置驱动 HTML / 动漫站点 / RSS→CSV / JSON API / HTML→Markdown /
> CSV 导出），文档站 + GoDoc + CI + Release 流水线齐全。欢迎 Star 与 PR！

### 知乎 / V2EX（长版）

> **标题**：轻量通用 Go 爬虫框架实践：从单一站点到 7 个示例
>
> **正文**：为什么要做？单一站点爬虫难以复用；做成框架后，站点差异被收敛到
> `Crawler` 接口的三阶段里。设计取舍：① 公开包零第三方依赖（可审计、可嵌入）；
> ② Engine 独立于站点逻辑（并发/重试/限速/输出）；③ 配置驱动与代码驱动双路径
> （demo 用 YAML 规则，其余示例用代码）。生产级能力（中间件链、抗反爬、多存储、
> 断点续爬）保留在 internal/ 作为参考实现。GitHub：xfengyin/jciyuan-spider。

### 公众号（一句话 + 配图）

> **标题**：Go 零依赖爬虫框架，3 行代码跑通
> **配图建议**：`examples/` 目录树 + 3 行代码截图 + badges。

## 6. 发布渠道 Tracking

| 渠道 | 链接/目标 | 状态 | 提交日期 | 结果/反馈 | 负责人 |
|------|-----------|------|----------|-----------|--------|
| GitHub Release（v0.1.0） | github.com/xfengyin/jciyuan-spider/releases | ✅ 已发布 | 2026-08-24 | - | xfengyin |
| GitHub Release（v0.2.0） | 同上 | ✅ 已发布 | 2026-08-24 | - | xfengyin |
| GitHub Release（v0.3.0） | 同上 | ⏳ 待打 tag | - | - | xfengyin |
| Awesome Go | github.com/avelino/awesome-go | 📝 待投稿 | - | - | - |
| Hacker News（Show HN） | news.ycombinator.com | 📝 草稿就绪（§3） | - | - | - |
| Reddit r/golang | reddit.com/r/golang | 📝 草稿就绪（§4） | - | - | - |
| 掘金 | juejin.cn | 📝 草稿就绪（§5） | - | - | - |
| 知乎 | zhihu.com | 📝 草稿就绪（§5） | - | - | - |
| V2EX | v2ex.com | 📝 草稿就绪（§5） | - | - | - |
| 公众号 | 微信 | 📝 草稿就绪（§5） | - | - | - |

状态图例：✅ 已完成 · ⏳ 待办 · 📝 草稿就绪 · 📤 已投 · ⏸ 被拒 · ↩️ 待重投

## 7. 发布节奏与注意事项

- **节奏**：每个里程碑合入 main 并打 tag（release 流水线自动出包）后，在对应渠道
  各发一轮；重大特性（如 v1.0）配长文，小版本仅更新 Release 说明。
- **合规**：README 免责声明已声明仅限学习研究；示例默认限速 >=500ms；推广文案避免
  引导绕过目标站反爬。
- **不刷星**：不做互刷/付费推广；靠内容与示例质量自然增长。
- **数据**：投稿后更新 tracking 表；7 天后复盘各渠道引流（Star/Issue/PR）。

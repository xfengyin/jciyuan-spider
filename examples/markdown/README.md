# Example: markdown

抓取 HTML 并导出 Markdown 文件：复用 `crawler.Engine` + 默认 HTTP 爬虫抓取，
再用内置的轻量 HTML→MD 转换器（仅标准库 `regexp`/`html`，不引入第三方库）生成 `.md` 文件。

## 运行

```bash
# 在仓库根目录
go run ./examples/markdown -url https://example.com
go run ./examples/markdown -url https://example.com,https://example.org -output ./output/markdown
```

## 参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `-url` | - | 抓取目标 URL，多个用逗号分隔 |
| `-output` | ./output/markdown | 输出目录（每页一个 `<slug>.md`） |
| `-concurrency` | 3 | Engine 并发数 |
| `-max-retry` | 2 | 失败重试次数 |
| `-quiet` | false | 安静模式 |

## 转换覆盖

标题 h1-h6、链接、图片、列表、粗体/斜体、行内代码、代码块（占位保护）、引用、
`<br>`/`<hr>`；`<script>`/`<style>` 整块剔除；HTML 实体解码；其余标签剥离。
详见 [docs/examples.md](../../docs/examples.md)。

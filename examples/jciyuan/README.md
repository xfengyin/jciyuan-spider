# Example: jciyuan

将仓库内置的企业级组件（Fetcher 中间件链 / 配置驱动 HTML Pipeline 解析器）包装为框架
[`crawler.Crawler`](../../crawler) 接口实现的示例，演示「单一站点爬虫」如何接入「通用爬虫框架」。

## 运行

```bash
# 在仓库根目录
go run ./examples/jciyuan -id 37439
go run ./examples/jciyuan -url https://www.jciyuan.com/acgdetail/37439.html -output ./output/jciyuan
```

## 参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `-config` | config/config.yaml | 配置文件路径（缺失时回退内置默认配置） |
| `-id` | 0 | 动漫 ID，按配置模板生成详情页 URL |
| `-url` | - | 直接指定详情页 URL（优先于 `-id`） |
| `-output` | ./output/jciyuan | 输出目录（JSON Lines） |
| `-concurrency` | 3 | Engine 并发数 |
| `-max-retry` | 3 | 失败重试次数 |
| `-quiet` | false | 安静模式 |

## 实现要点

- `Fetch` 委托内部 `fetcher.Fetcher`（含限流/重试/熔断/代理等中间件链与抗反爬）。
- `Parse` 复用配置驱动的 `parser.Parser`（`config/config.yaml` 中的 extractors）。
- `Extract` 将 `parser.ParseResult` 映射为通用 `crawler.Item`。
- 通过 `crawler.Register("jciyuan", ...)` 注册，使用 `crawler.Build` 按名称构建。
- 由通用 `crawler.Engine` 统一调度并发、重试、限速与输出。

# jciyuan-spider - 轻量通用 Go 爬虫框架

<p align="center">
  <strong>Go</strong> · <strong>Generic Crawler Interface</strong> · <strong>Engine</strong> · <strong>SPI Plugins</strong>
</p>

---

**jciyuan-spider** 是一个轻量通用的 Go 爬虫框架：定义 `Crawler` 接口（`Fetch → Parse → Extract`），
由通用 `Engine` 统一调度并发、重试、限速与结果输出。任何站点/数据源只需实现该接口即可接入框架；
仓库内置的企业级组件（Fetcher 中间件链、配置驱动 HTML 解析器、多存储后端）作为参考实现，
原有 [jciyuan.com](https://www.jciyuan.com) 动漫爬虫保留为 [Example](#examples-示例)。

> M1 里程碑：从「单一站点爬虫」转型为「通用爬虫框架」——新增公开框架包 [`crawler/`](crawler)，
> 原功能保持可用（根目录 CLI 不变）。

## 功能特性

| 特性 | 描述 |
|------|------|
| 🧩 **通用 Crawler 接口** | `Fetch / Parse / Extract` 三阶段抽象，任何站点实现即接入 |
| ⚙️ **通用 Engine** | 并发 Worker、失败重试、请求间隔限速、超时控制、JSON Lines 输出 |
| 🔌 **SPI 注册机制** | `crawler.Register` / `crawler.Build`，按名称装配爬虫实现 |
| 🎯 **示例开箱即用** | [`examples/jciyuan`](examples/jciyuan)（复用企业级组件）、[`examples/demo`](examples/demo)（纯标准库通用爬虫） |
| 🧱 **企业级组件（内部）** | Fetcher 中间件链（trace/metrics/限流/重试/熔断/代理轮换）、抗反爬、多存储后端 |
| ⛓️ **中间件链** | trace → metrics → logging → rate_limit → retry → circuit_breaker → proxy_rotate，顺序可配置 |
| 🛡️ **抗反爬** | Random UA、Referer、Cookie 保持、URL 白名单、robots.txt 检查 |
| 📦 **多存储后端** | JSON（默认）、SQLite、MySQL、S3，外加内存缓存装饰器 |
| 🔄 **断点续爬** | 爬取状态持久化，中断后可恢复；支持增量合并保留已抓取的 M3U8 |
| 📊 **可观测性** | 内存 / Prometheus 指标、/healthz 健康检查、traceId 全链路日志 |

## 快速开始

### 方式一：运行内置示例

```bash
git clone https://github.com/xfengyin/jciyuan-spider.git
cd jciyuan-spider
go run ./examples/demo -config examples/demo/config.yaml                    # 通用示例：抓取 example.com
go run ./examples/jciyuan -id 37439                                         # jciyuan 示例：抓取动漫详情页
```

### 方式二：实现自己的爬虫（约 30 行）

```go
package main

import (
"context"
"fmt"
"io"
"net/http"

"jciyuan-spider/crawler"
)

type myCrawler struct{}

func (c *myCrawler) Name() string { return "my" }

func (c *myCrawler) Fetch(ctx context.Context, url string) (*crawler.Page, error) {
resp, err := http.Get(url)
if err != nil {
return nil, err
}
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
return &crawler.Page{URL: url, StatusCode: resp.StatusCode, Body: body, Text: string(body)}, nil
}

func (c *myCrawler) Parse(_ context.Context, page *crawler.Page) (any, error) {
return page.Text, nil // 中间表示：原样透传
}

func (c *myCrawler) Extract(_ context.Context, parsed any) ([]crawler.Item, error) {
return []crawler.Item{{"text": parsed}}, nil
}

func main() {
eng := crawler.NewEngine(&myCrawler{}, crawler.Options{
Concurrency: 4, MaxRetry: 2, OutputDir: "./output/my",
})
result, err := eng.Run(context.Background(), []string{"https://example.com"})
if err != nil {
panic(err)
}
fmt.Printf("成功=%d 失败=%d 条目=%d\n", result.Stats.Success, result.Stats.Fail, result.Stats.Items)
}
```

### 方式三：原有 CLI（jciyuan 动漫爬虫，保持兼容）

```bash
go build -o jciyuan-spider .
./jciyuan-spider -id 37439 -incremental -debug
```

## 接口说明

### Crawler 接口

```go
type Crawler interface {
Name() string                                        // 爬虫名称（注册/日志/配置识别）
Fetch(ctx context.Context, url string) (*Page, error) // 抓取原始页面
Parse(ctx context.Context, page *Page) (any, error)   // 解析为中间表示（DOM/文档/对象）
Extract(ctx context.Context, parsed any) ([]Item, error) // 抽取结构化条目，可返回多条
}
```

- `Page`：原始页面，含 `URL / StatusCode / Headers / Body / Text / Meta`。
- `Item`：`map[string]any`，一条结构化结果。
- 数据流：`Fetch → Parse → Extract`，三个阶段相互独立，可单独替换实现。

### Engine

```go
eng := crawler.NewEngine(c, crawler.Options{
Concurrency: 4,            // 并发 Worker 数
Delay:       500 * time.Millisecond, // 请求间隔限速
MaxRetry:    2,            // 失败重试次数
Timeout:     10 * time.Second,       // 单次 Fetch 超时
OutputDir:   "./output",   // 结果目录（JSON Lines: items.jsonl）
})
result, err := eng.Run(ctx, urls) // *Result{Stats, Items, Failures}
```

单个 URL 失败（含重试后）不会中断整个运行，失败明细记录在 `Result.Failures`。

### SPI 注册

```go
crawler.Register("my", func(cfg map[string]any) (crawler.Crawler, error) {
return &myCrawler{}, nil
})
c, err := crawler.Build("my", cfg) // 按名称构建
```

## Examples 示例

| 示例 | 说明 | 运行 |
|------|------|------|
| [`examples/jciyuan`](examples/jciyuan) | 复用内置企业级 Fetcher/Parser 组件抓取 jciyuan.com 动漫详情页（原站点爬虫的框架化实现） | `go run ./examples/jciyuan -id 37439` |
| [`examples/demo`](examples/demo) | 纯标准库 + goquery 的通用 HTML 爬虫，YAML 配置驱动（CSS/正则选择器），抓取任意站点 | `go run ./examples/demo -config examples/demo/config.yaml` |

## 项目结构

```
jciyuan-spider/
├── main.go                     # 原有 CLI 入口（jciyuan 动漫爬虫，保持兼容）
├── crawler/                    # 🆕 通用框架包：Crawler 接口 + Engine + SPI（仅依赖标准库）
├── config/config.yaml          # 默认配置
├── examples/
│   ├── jciyuan/                # 🆕 示例：企业级组件 → 框架接口适配
│   └── demo/                   # 🆕 示例：通用配置驱动爬虫
└── internal/                   # 内部实现（可作为扩展参考）
    ├── config/                 # YAML 加载、默认值、校验、环境变量覆盖
    ├── di/                     # 依赖注入容器（含各插件的副作用导入）
    ├── errors/                 # 错误分类（network/parse/storage/...）+ Retry 标记
    ├── fetcher/                # Fetcher 接口与中间件
    │   ├── http/               # HTTP 实现：白名单、robots、UA、gzip
    │   └── middleware/         # 限流/重试/熔断/代理轮换/trace/指标/日志
    ├── parser/
    │   ├── extractor/          # CSS / XPath / Regex 提取器（配置驱动）
    │   ├── html/               # HTML 解析器 + 剧集链接解析
    │   └── processor/          # 字段后处理器（去重、排序等）
    ├── storage/                # Storage 接口 + json/sqlite/mysql/s3/内存缓存
    ├── spider/                 # 核心编排：任务调度、状态机、增量合并
    ├── worker/                 # goroutine 池
    ├── resume/                 # 断点续爬状态机
    ├── metrics/                # 指标（memory / prometheus）
    ├── health/                 # /healthz 健康检查
    ├── logger/                 # zap + lumberjack 封装
    └── model/                  # 配置与数据模型
```

## 配置说明

完整示例见 [config/config.yaml](config/config.yaml)，主要段落：

```yaml
spider:          # 站点、并发、超时、重试
crawl:           # anime_id、resume、incremental、max_episodes
fetcher:         # HTTP 传输参数、代理策略
anticrawler:     # random_ua、referer_policy、robots_txt_check
parser:          # html 编码 + extractors（配置驱动的字段提取）
storage:         # type: json|sqlite|mysql|s3，output 开关（save_json/save_m3u8/save_raw_html）
middlewares:     # 中间件链及顺序
metrics:         # memory|prometheus（prometheus 时暴露 /metrics 与 /healthz）
log:             # 级别、格式、文件轮转
```

配置可被 `JCIYUAN_*` 环境变量与命令行 flag 覆盖；配置文件缺失时回退到内置默认配置。
框架示例（`examples/demo`）使用自己的 YAML 任务配置，见 [examples/demo/config.yaml](examples/demo/config.yaml)。

## 开发

```bash
go build ./...
go vet ./...
go test ./...
```

## 注意事项

⚠️ **免责声明**
- 本工具仅供学习研究使用
- 请遵守目标网站的 robots.txt 和使用条款
- 爬取行为需自行承担法律责任
- 建议设置合理的请求间隔（>=1000ms）

## License

MIT License - see [LICENSE](LICENSE) for details.

# 快速开始

> 架构与设计见 [architecture.md](architecture.md)，示例详解见 [examples.md](examples.md)。

## 环境要求

- Go 1.25+（`go.mod` 声明 `go 1.25.0`）

## 1. 三行代码抓取任意页面

```go
ctx := context.Background()
result, _ := crawler.NewEngine(crawler.NewHTTPCrawler(), crawler.Options{}).Run(ctx, []string{"https://example.com"})
fmt.Println(result.Items[0]["text"]) // 输出页面正文
```

更简的单行抓取：

```go
page, _ := crawler.Fetch(ctx, "https://example.com")
```

## 2. 运行内置示例

```bash
git clone https://github.com/xfengyin/jciyuan-spider.git
cd jciyuan-spider

# 通用 HTML 爬虫（配置驱动，抓取 example.com）
go run ./examples/demo -config examples/demo/config.yaml

# jciyuan 动漫爬虫（复用内部企业级组件）
go run ./examples/jciyuan -id 37439

# RSS/XML 源解析（本地文件离线演示）
go run ./examples/rss -url examples/rss/sample.xml
```

结果默认以 JSON Lines 写入 `./output/<示例名>/items.jsonl`。

## 3. 实现自己的爬虫（约 30 行）

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

只关心 `Fetch` 阶段？直接用内置的 `crawler.NewHTTPFetcher()`（详见 [architecture.md](architecture.md)）。

## 4. 原有 CLI（jciyuan 动漫爬虫，保持兼容）

```bash
go build -o jciyuan-spider .
./jciyuan-spider -id 37439 -incremental -debug
```

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-config` | config/config.yaml | 配置文件路径 |
| `-id` | 37439 | 动漫 ID |
| `-delay` | 1000 | 请求间隔 (ms) |
| `-output` | ./output | 输出目录（JSON 存储） |
| `-resume` | false | 启用断点续爬 |
| `-incremental` | false | 增量更新（与旧数据合并，保留已抓取的 M3U8） |
| `-stats` | true | 结束时显示统计信息 |
| `-debug` | false | 调试模式（日志级别设为 debug） |
| `-version` | - | 打印版本信息 |

## 5. 配置说明

完整示例见 [config/config.yaml](../config/config.yaml)，主要段落：

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
框架示例（`examples/demo`）使用自己的 YAML 任务配置，见 [examples/demo/config.yaml](../examples/demo/config.yaml)。

## 6. 开发

推荐使用根目录 [`Makefile`](../Makefile) 统一命令：

```bash
make build          # 编译根 CLI 到 ./jciyuan-spider（VERSION 可覆盖：make build VERSION=v0.5.0）
make test           # 运行全部单元测试
make vet            # 静态检查
make fmt            # gofmt 格式化
make fmt-check      # 检查 gofmt（CI 用）
make clean          # 清理构建产物
make run-demo       # 运行任一示例（run-jciyuan / run-rss / run-csv / run-markdown / run-json-api / run-xml）
make version        # 打印构建版本信息
```

等价的原生命令：`go build ./...` / `go vet ./...` / `go test ./...`。

### 版本输出

- 全部 7 个示例 CLI（demo / jciyuan / rss / csv / markdown / json-api / xml）
  支持 `--version`，统一输出 `jciyuan-spider <name> example <version>`。
- 版本号集中维护于 [`internal/version`](../internal/version)（`const Version`），
  与 [CHANGELOG](../CHANGELOG.md) 同步。
- 根 CLI `./jciyuan-spider -version` 输出注入的版本/commit/构建时间
  （`make build VERSION=v0.5.0` 可指定）。

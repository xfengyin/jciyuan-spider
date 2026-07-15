# jciyuan-spider-v3 企业级重构 PRD

> **文档版本**：v1.0
> **日期**：2026-07-15
> **状态**：待评审
> **目标读者**：后端架构师、Go 开发工程师、SRE、QA
> **范围**：在 jciyuan-spider-v2 已完成接口化重构的基础上，进一步升级为生产可用、高可用、可扩展、可观测、安全、可维护、可测试的企业级爬虫平台。

---

## 1. 项目背景与目标

### 1.1 背景

`jciyuan-spider-v2` 已完成初步企业化改造：

- 模块按 `internal/fetcher`、`internal/parser`、`internal/storage`、`internal/spider` 分层；
- 核心模块已抽象为接口；
- 引入中间件链处理限流与重试；
- 支持 JSON 持久化、断点续爬、基础指标统计。

但距离真正生产可用仍有明显差距：配置项与实际能力不匹配（如 `concurrency`、`save_sqlite`、`enable_proxy` 未实现）、依赖硬编码、缺乏并发 Worker 池、缺少熔断降级、metrics 仅内存计数、无全链路 Trace、测试覆盖为零。

### 1.2 目标

将 `jciyuan-spider-v2` 重构为 **jciyuan-spider-v3**，满足以下标准：

| 维度 | 目标 |
|------|------|
| 高可用 | 超时、指数退避重试、熔断、限流、降级、多代理/多 UA 兜底 |
| 可扩展 | 配置驱动、SPI 插件化、Pipeline 解析、存储多后端 |
| 可观测 | traceId 全链路、结构化日志、Prometheus metrics、健康检查 |
| 安全 | URL 白名单、敏感信息脱敏、robots 合规、风险拦截 |
| 可维护 | 单一职责、依赖倒置、接口隔离、开闭原则 |
| 可测试 | Mock 实现、httptest、fixture、沙箱测试、≥80% 单元测试覆盖率 |

### 1.3 非目标

- 不实现验证码破解、签名伪造等违反目标站点 ToS 的能力；
- 不追求通用多站点爬虫（仍聚焦囧次元），但架构需支持通过配置扩展新站点；
- 本次不涉及前端管理后台，仅保留命令行入口与可嵌入的 SDK。

---

## 2. 现状分析

### 2.1 目录结构现状

```text
/workspace
├── main.go
├── config/config.yaml
├── internal/
│   ├── config/loader.go
│   ├── fetcher/{interface.go,http_fetcher.go,middleware.go}
│   ├── logger/logger.go
│   ├── metrics/collector.go
│   ├── model/model.go
│   ├── parser/{html_parser.go,interface.go,utils.go}
│   ├── resume/manager.go
│   ├── spider/spider.go
│   └── storage/{interface.go,json_storage.go,memory_storage.go}
```

### 2.2 关键问题清单

| 模块 | 现状问题 | 风险等级 |
|------|---------|---------|
| **Spider 编排** | `NewSpider` 硬编码 `HTTPFetcher`、`HTMLParser`、`JSONStorage`；主流程无法替换实现 | P0 |
| **配置** | `concurrency`、`save_sqlite`、`save_m3u8`、`enable_proxy`、`stats.interval` 等字段未消费 | P0 |
| **并发** | 仍是单页串行，未使用配置的并发能力 | P0 |
| **Fetcher** | 仅有限流+固定退避重试；无熔断、无代理池、无连接池调优 | P1 |
| **Parser** | 单体 `ParseAnimeDetail`，新增字段/站点需改主流程；正则硬编码 | P1 |
| **Storage** | 仅 JSON 实现；无事务、无批量、无 Upsert | P1 |
| **Metrics** | 内存计数器，进程退出即丢失；无 Prometheus 输出 | P1 |
| **Logger** | 自定义 logger，无 traceId、无结构化 JSON | P1 |
| **错误处理** | `BlockedError` 用类型断言；无错误分类体系 | P1 |
| **测试** | 无 `tests/` 目录，无可 Mock 的测试桩 | P2 |
| **安全** | 无 URL 白名单、无敏感信息脱敏、无 robots 检查 | P2 |

---

## 3. 设计原则

严格遵循以下工程原则：

1. **开闭原则**：新增能力通过扩展（新插件、新配置）实现，不修改主流程；
2. **依赖倒置**：`Spider` 只依赖接口，不依赖具体实现；
3. **单一职责**：每个包、每个函数只做一件事；
4. **接口隔离**：轻量、细粒度接口，按需组合；
5. **配置驱动**：提示词、解析规则、中间件、存储后端全部配置化；
6. **插件化/SPI**：Fetcher、Parser、Storage、Middleware 支持动态注册；
7. **幂等一致性**：重复执行不产生脏数据，增量更新幂等；
8. **可观测**：traceId 全链路、结构化日志、指标暴露；
9. **防御式编程**：外部输入（URL、HTML、配置）一律校验。

---

## 4. 总体架构

### 4.1 架构分层

```text
┌─────────────────────────────────────────────────────────────┐
│                       Entry Layer                            │
│  cmd/spider/main.go  |  cmd/server/main.go (future)         │
├─────────────────────────────────────────────────────────────┤
│                      Config Layer                            │
│  internal/config (YAML/Env/Flag 统一加载 + 校验 + 默认值)     │
├─────────────────────────────────────────────────────────────┤
│                       DI / Factory                           │
│  internal/di (根据配置装配 Fetcher/Parser/Storage/Middleware) │
├─────────────────────────────────────────────────────────────┤
│                      Core Engine                             │
│  internal/spider (任务调度、WorkerPool、Pipeline、状态机)     │
├─────────────────────────────────────────────────────────────┤
│  Fetcher    │   Parser      │   Storage     │   Resume      │
│  (接口+插件) │   (接口+插件)  │   (接口+插件)  │   (状态持久化) │
├─────────────────────────────────────────────────────────────┤
│  Middleware Chain (RateLimit / Retry / CircuitBreaker /      │
│  ProxyRotate / Logging / Metrics / Signature)                │
├─────────────────────────────────────────────────────────────┤
│  Observability (Logger with traceId / Metrics / HealthCheck) │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 核心接口关系

```go
// 核心引擎只持有三大抽象
Spider:
  fetcher Fetcher
  parser  Parser
  storage Storage
  resume  *resume.Manager
  pool    *worker.Pool
  metrics Metrics
  logger  Logger

// Fetcher 接口
Fetcher.Fetch(ctx context.Context, req *Request) (*Response, error)

// Parser 接口
Parser.Parse(ctx context.Context, raw *Response) (*ParseResult, error)

// Storage 接口
Storage.Save(ctx context.Context, anime *AnimeInfo) error
Storage.Load(ctx context.Context, animeID int64) (*AnimeInfo, error)
Storage.SaveBatch(ctx context.Context, animes []*AnimeInfo) error

// Middleware 接口
Middleware.Handle(ctx context.Context, req *Request, next Handler) (*Response, error)
```

---

## 5. 模块详细设计

### 5.1 配置层（internal/config）

#### 5.1.1 配置结构

```yaml
# config/config.yaml
app:
  name: jciyuan-spider-v3
  mode: cli          # cli | server
  trace_id_header: X-Request-ID

spider:
  base_url: "https://www.jciyuan.com"
  detail_url_pattern: "{{base_url}}/acgdetail/{{id}}.html"
  delay: 1000                     # ms
  timeout: 10                     # s
  max_retry: 3
  concurrency: 3                  # 全局并发 Worker 数
  queue_size: 100                 # 任务队列长度
  crawl_interval: 0               # 连续 Crawl 任务间隔 ms

fetcher:
  type: http                      # http | selenium | playwright
  http:
    transport:
      max_idle_conns: 100
      max_conns_per_host: 10
      idle_conn_timeout: 90s
      disable_keep_alives: false
    follow_redirects: true
    max_body_size: 52428800       # 50MB
  proxy:
    enable: false
    strategy: round_robin         # round_robin | random | least_used
    proxies:
      - "http://user:pass@proxy1:8080"
      - "http://proxy2:8080"

anticrawler:
  random_ua: true
  user_agents:
    - "Mozilla/5.0 ..."
  keep_cookie: true
  referer_policy: origin
  robots_txt_check: true

parser:
  type: html
  html:
    encoding: auto                # auto | utf-8 | gbk
    extractors:
      - field: title
        selector: regex:<title[^>]*>([^<]+)</title>
        processors: [clean_text, split_by_-]
      - field: episodes
        selector: regex:/acgplay/(\d+)-(\d+)-(\d+)\.html
        deduplicate: true
        sort_by: number

storage:
  type: json                      # json | sqlite | mysql | s3
  json:
    output_dir: "./output"
  sqlite:
    dsn: "./data/spider.db"
  output:
    save_json: true
    save_sqlite: false
    save_m3u8: false
    save_raw_html: false          # 失败/审计时保存原始 HTML

crawl:
  anime_id: 37439
  resume: true
  incremental: false
  max_episodes: 0                 # 0 = 全部
  max_pages_per_run: 0            # 0 = 无限制

middlewares:
  - name: trace
  - name: metrics
  - name: logging
  - name: rate_limit
  - name: retry
  - name: circuit_breaker
  - name: proxy_rotate

metrics:
  enabled: true
  backend: memory                 # memory | prometheus
  prometheus:
    port: 9090
    path: /metrics

log:
  level: info
  format: text                    # text | json
  file: "./logs/spider.log"
  console: true
  max_size: 10
  max_backups: 5
```

#### 5.1.2 配置加载顺序

1. 加载 `config.yaml`；
2. 应用默认值；
3. 校验（使用 `go-playground/validator`）；
4. 环境变量覆盖（前缀 `JCIYUAN_`）；
5. 命令行 Flag 覆盖。

### 5.2 依赖注入与工厂（internal/di）

#### 5.2.1 DI 容器职责

- 根据 `fetcher.type`、`storage.type`、`parser.type` 从 `PluginRegistry` 查找实现；
- 装配 Middleware 链；
- 初始化 WorkerPool、Metrics、Logger；
- 返回一个完全装配好的 `*spider.Spider`。

#### 5.2.2 示例代码

```go
// internal/di/container.go
package di

import (
    "context"
    "fmt"

    "jciyuan-spider-v2/internal/config"
    "jciyuan-spider-v2/internal/fetcher"
    "jciyuan-spider-v2/internal/logger"
    "jciyuan-spider-v2/internal/metrics"
    "jciyuan-spider-v2/internal/parser"
    "jciyuan-spider-v2/internal/resume"
    "jciyuan-spider-v2/internal/spider"
    "jciyuan-spider-v2/internal/storage"
    "jciyuan-spider-v2/internal/worker"
)

// Container 依赖注入容器
type Container struct {
    cfg *config.Config
}

// NewContainer 创建容器
func NewContainer(cfg *config.Config) *Container {
    return &Container{cfg: cfg}
}

// BuildSpider 构建爬虫实例
func (c *Container) BuildSpider(ctx context.Context) (*spider.Spider, error) {
    log := logger.New(c.cfg.Log)
    m := metrics.New(c.cfg.Metrics)

    fetcherImpl, err := fetcher.Build(c.cfg.Fetcher, c.cfg.Anticrawler, m, log)
    if err != nil {
        return nil, fmt.Errorf("build fetcher: %w", err)
    }

    parserImpl, err := parser.Build(c.cfg.Parser)
    if err != nil {
        return nil, fmt.Errorf("build parser: %w", err)
    }

    storageImpl, err := storage.Build(c.cfg.Storage)
    if err != nil {
        return nil, fmt.Errorf("build storage: %w", err)
    }

    // 断点续爬状态存储复用 storage 的 StatusStorage 能力
    statusStore, ok := storageImpl.(storage.StatusStorage)
    if !ok {
        return nil, fmt.Errorf("storage backend must implement StatusStorage")
    }

    pool := worker.NewPool(c.cfg.Spider.Concurrency, c.cfg.Spider.QueueSize)
    resumeMgr := resume.NewManager(statusStore)

    s := spider.New(spider.Options{
        Config:  c.cfg,
        Fetcher: fetcherImpl,
        Parser:  parserImpl,
        Storage: storageImpl,
        Resume:  resumeMgr,
        Pool:    pool,
        Metrics: m,
        Logger:  log,
    })

    return s, nil
}
```

### 5.3 Fetcher 层（internal/fetcher）

#### 5.3.1 接口定义

```go
// internal/fetcher/fetcher.go
package fetcher

import "context"

// Request 请求对象
type Request struct {
    URL     string
    Method  string
    Headers map[string]string
    Body    []byte
    Meta    map[string]interface{} // 透传 traceId、attempt 等
}

// Response 响应对象
type Response struct {
    URL        string
    StatusCode int
    Headers    map[string][]string
    Body       []byte
    Meta       map[string]interface{}
    Duration   int64 // ms
}

// Fetcher 请求器接口
type Fetcher interface {
    Fetch(ctx context.Context, req *Request) (*Response, error)
    Close() error
}

// Builder 构造器函数签名
type Builder func(cfg config.FetcherConfig, anti config.AnticrawlerConfig, m metrics.Metrics, l logger.Logger) (Fetcher, error)

var registry = make(map[string]Builder)

// Register 注册实现
func Register(name string, b Builder) { registry[name] = b }

// Build 按配置构建
func Build(cfg config.FetcherConfig, anti config.AnticrawlerConfig, m metrics.Metrics, l logger.Logger) (Fetcher, error) {
    b, ok := registry[cfg.Type]
    if !ok {
        return nil, fmt.Errorf("unknown fetcher type: %s", cfg.Type)
    }
    return b(cfg, anti, m, l)
}
```

#### 5.3.2 HTTPFetcher 设计

- 使用 `http.Transport` 连接池参数可配置；
- 支持 gzip/deflate 自动解压；
- 请求头集中管理：UA、Referer、Accept、Accept-Encoding、Cookie；
- 集成 Middleware 链。

#### 5.3.3 Middleware 链

```go
// internal/fetcher/middleware.go
type Handler func(ctx context.Context, req *Request) (*Response, error)
type Middleware func(next Handler) Handler

// 核心中间件
- TraceMiddleware      // 注入/透传 traceId
- MetricsMiddleware    // 记录 QPS/延迟/状态码
- LoggingMiddleware    // 请求/响应日志
- RateLimitMiddleware  // 令牌桶/固定窗口限流
- RetryMiddleware      // 指数退避 + 抖动
- CircuitBreakerMiddleware // 熔断器
- ProxyRotateMiddleware    // 代理切换
```

#### 5.3.4 熔断器设计

使用标准熔断三态：Closed / Open / HalfOpen。

- 触发条件：连续失败 5 次 或 60 秒内错误率 > 50%；
- Open 后冷却 30s 进入 HalfOpen；
- HalfOpen 放行 1 个探测请求，成功则 Closed，失败则重新 Open。

#### 5.3.5 指数退避重试

```go
backoff := baseDelay * math.Pow(2, attempt) + jitter
if backoff > maxBackoff {
    backoff = maxBackoff
}
```

### 5.4 Parser 层（internal/parser）

#### 5.4.1 Pipeline 解析模型

```go
// internal/parser/parser.go
package parser

import "context"

// ParseResult 解析结果
type ParseResult struct {
    Anime    *model.AnimeInfo
    Episodes []*model.Episode
    RawHTML  []byte // 审计用
}

// Parser 解析器接口
type Parser interface {
    Parse(ctx context.Context, resp *fetcher.Response) (*ParseResult, error)
}

// Extractor 字段提取器接口
type Extractor interface {
    Name() string
    Extract(ctx context.Context, doc *Document) (interface{}, error)
}

// Document 解析上下文
type Document struct {
    URL      string
    HTML     string
    Encoding string
    Meta     map[string]interface{}
}
```

#### 5.4.2 HTMLParser 实现

- 使用 `goquery` 或 `htmlquery` 做 DOM 解析，减少脆弱的正则；
- 内置 XPath/CSS/Regex 三类 Selector；
- 字段映射从配置读取；
- 未命中字段不报错，写入空值并记录 warning。

#### 5.4.3 字段提取配置示例

```yaml
parser:
  html:
    extractors:
      - field: title
        selector:
          type: css
          value: "h1.title"
        processors:
          - type: clean_text
          - type: split
            separator: "_"
      - field: year
        selector:
          type: regex
          value: "(19|20)\\d{2}"
      - field: tags
        selector:
          type: css
          value: "a.tag"
        multiple: true
        processors:
          - type: trim
      - field: episodes
        selector:
          type: regex
          value: "/acgplay/(\\d+)-(\\d+)-(\\d+)\\.html"
        multiple: true
        deduplicate: true
        sort_by: number
```

### 5.5 Storage 层（internal/storage）

#### 5.5.1 接口定义

```go
// internal/storage/storage.go
package storage

import "context"

type Storage interface {
    Save(ctx context.Context, anime *model.AnimeInfo) error
    Load(ctx context.Context, animeID int64) (*model.AnimeInfo, error)
    Exists(ctx context.Context, animeID int64) (bool, error)
    SaveBatch(ctx context.Context, animes []*model.AnimeInfo) error
    Close() error
}

type StatusStorage interface {
    SaveStatus(ctx context.Context, status *model.CrawlStatus) error
    LoadStatus(ctx context.Context, animeID int64) (*model.CrawlStatus, error)
}
```

#### 5.5.2 实现清单

| 实现 | 文件 | 说明 |
|------|------|------|
| JSONStorage | `storage/json/json_storage.go` | 单文件 JSON，兼容 v2 |
| SQLiteStorage | `storage/sqlite/sqlite_storage.go` | 支持 Upsert、事务、索引 |
| MySQLStorage | `storage/mysql/mysql_storage.go` | 生产持久化 |
| S3Storage | `storage/s3/s3_storage.go` | 云原生归档 |
| MemoryStorage | `storage/memory/memory_storage.go` | L1 缓存装饰器 |

#### 5.5.3 SQLite Schema

```sql
CREATE TABLE IF NOT EXISTS anime (
    id INTEGER PRIMARY KEY,
    title TEXT,
    year TEXT,
    region TEXT,
    tags TEXT,          -- JSON 数组
    cover_image TEXT,
    description TEXT,
    update_date TEXT,
    episode_num INTEGER,
    update_num INTEGER,
    douban_url TEXT,
    detail_url TEXT,
    status INTEGER,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS episode (
    anime_id INTEGER,
    number INTEGER,
    title TEXT,
    url TEXT,
    m3u8_url TEXT,
    is_vip BOOLEAN,
    is_crawled BOOLEAN,
    created_at DATETIME,
    PRIMARY KEY (anime_id, number)
);

CREATE TABLE IF NOT EXISTS crawl_status (
    anime_id INTEGER PRIMARY KEY,
    status TEXT,
    current_index INTEGER,
    total_count INTEGER,
    success_count INTEGER,
    fail_count INTEGER,
    retry_count INTEGER,
    error_msg TEXT,
    last_crawl_at DATETIME
);
```

### 5.6 核心引擎（internal/spider）

#### 5.6.1 状态机

```text
Idle -> Running -> Completed
              -> Failed
              -> Paused  (信号中断)
```

#### 5.6.2 任务类型

```go
const (
    TaskTypeDetail TaskType = "detail"   // 详情页
    TaskTypeEpisode TaskType = "episode" // 单集页面（M3U8）
)
```

#### 5.6.3 调度流程

1. 检查 `CrawlStatus`，已 `completed` 且非增量则跳过；
2. 标记 `running`；
3. 生成详情页任务，投递 WorkerPool；
4. Worker 执行 `Fetch` → `Parse` → `Storage.Save`；
5. 解析出剧集后，如开启 M3U8 抓取，生成 `episode` 任务并行处理；
6. 所有任务完成后合并结果；
7. 标记 `completed` / `failed`；
8. 输出统计与摘要。

#### 5.6.4 WorkerPool

```go
// internal/worker/pool.go
package worker

type Pool struct {
    workers int
    queue   chan Task
    wg      sync.WaitGroup
}

func (p *Pool) Submit(ctx context.Context, task Task) error
func (p *Pool) Stop() error
```

使用 `semaphore.Weighted` 控制并发，支持优雅关闭。

### 5.7 可观测性

#### 5.7.1 日志

- 使用 `uber-go/zap` 或 `sirupsen/logrus`；
- 输出格式支持 `text` / `json`；
- 每条日志携带 `traceId`、`module`、`caller`；
- 日志文件按大小轮转（`lumberjack`）。

#### 5.7.2 Metrics

| 指标 | 类型 | 说明 |
|------|------|------|
| spider_requests_total | Counter | 请求总数，按 status 分标签 |
| spider_request_duration_seconds | Histogram | 请求延迟 |
| spider_parse_total | Counter | 解析次数 |
| spider_parse_fail_total | Counter | 解析失败次数 |
| spider_storage_save_total | Counter | 保存次数 |
| spider_worker_queue_size | Gauge | 队列长度 |
| spider_circuit_breaker_state | Gauge | 熔断器状态 |

#### 5.7.3 Trace

- 入口 `main` 生成 `traceId`；
- 通过 `context.WithValue` 传递；
- Fetcher / Parser / Storage / Middleware 统一从 ctx 读取并打印。

### 5.8 安全设计

1. **URL 白名单**：只允许 `base_url` 对应域名，拒绝相对路径跳转/重定向到外部域名；
2. **敏感信息脱敏**：日志中隐藏 Cookie、`Authorization`、代理密码；
3. **Robots.txt 合规**：启动前拉取并解析，拒绝 Disallow 路径；
4. **限速保护**：默认最小 `delay >= 100ms`，强制遵守；
5. **内容安全**：不存储/输出恶意脚本，保存前做 HTML 转义；
6. **风险拦截**：遇到验证码页面立即停止并告警，不尝试绕过。

---

## 6. 关键数据流

```text
main
  │
  ▼
Config.Load ──► DI.BuildSpider
  │
  ▼
Spider.Run(ctx)
  │
  ├─► resume.LoadStatus
  ├─► WorkerPool.Submit(detailTask)
  │       │
  │       ▼
  │   Fetcher.Fetch(req)
  │       │
  │       ▼
  │   Middleware Chain
  │   (trace → metrics → logging → rate_limit → retry → circuit_breaker → proxy)
  │       │
  │       ▼
  │   HTTP Client
  │       │
  │       ▼
  │   Parser.Parse(resp)
  │       │
  │       ▼
  │   Pipeline Extractors
  │       │
  │       ▼
  │   Storage.Save(ctx, anime)
  │       │
  │       ▼
  │   resume.MarkCompleted / Failed
  │
  ▼
showStats
```

---

## 7. 接口与插件注册规范

### 7.1 SPI 注册示例

```go
// internal/fetcher/http/http.go
package http

import "jciyuan-spider-v2/internal/fetcher"

func init() {
    fetcher.Register("http", NewHTTPFetcher)
}
```

### 7.2 新增后端步骤

1. 在 `internal/storage/xxx/` 实现 `Storage` + `StatusStorage`；
2. `init()` 中调用 `storage.Register("xxx", builder)`；
3. 修改 `config.yaml`：`storage.type: xxx`；
4. 无需修改 `internal/spider`。

---

## 8. 测试策略

### 8.1 测试目录

```text
internal/
  fetcher/..._test.go
  parser/..._test.go
  storage/..._test.go
  spider/spider_test.go
testdata/
  detail_37439.html
  blocked.html
  captcha.html
```

### 8.2 Mock 实现

```go
// internal/mocks/fetcher.go
type MockFetcher struct {
    Responses map[string]*fetcher.Response
    Err       error
}

func (m *MockFetcher) Fetch(ctx context.Context, req *fetcher.Request) (*fetcher.Response, error) {
    if m.Err != nil {
        return nil, m.Err
    }
    resp, ok := m.Responses[req.URL]
    if !ok {
        return nil, fetcher.ErrNotFound
    }
    return resp, nil
}
```

### 8.3 测试目标

- 单元测试覆盖率 ≥ 80%；
- Fetcher：重试、熔断、代理切换、超时；
- Parser：各字段提取、失败降级、编码处理；
- Storage：Upsert、批量、并发安全；
- Spider：断点续爬、增量更新、上下文取消。

---

## 9. 实施计划

### Phase 1：基础架构（Week 1）

- [ ] 新建 `internal/di`、`internal/worker`、`internal/errors`；
- [ ] 重构 `Fetcher/Parser/Storage` 接口为 SPI + Builder 模式；
- [ ] 升级配置结构，补充缺失字段并全部消费；
- [ ] 引入 `zap` + `lumberjack` 替换自定义 logger。

### Phase 2：高可用与并发（Week 2）

- [ ] 实现 WorkerPool 与任务调度；
- [ ] 实现指数退避 + 抖动重试、熔断器、代理池；
- [ ] 实现 SQLiteStorage、MySQLStorage；
- [ ] 实现 MemoryStorage 缓存装饰器。

### Phase 3：可观测与安全（Week 3）

- [ ] traceId 全链路注入；
- [ ] Prometheus metrics 输出；
- [ ] URL 白名单、敏感信息脱敏、robots.txt 检查；
- [ ] 健康检查端点 `/healthz`。

### Phase 4：Pipeline 解析器与测试（Week 4）

- [ ] 实现 Extractor Pipeline + 配置化 selector；
- [ ] 编写 Mock 与单元测试；
- [ ] 集成测试与沙箱测试；
- [ ] 覆盖率 ≥ 80%。

### Phase 5：验收与文档（Week 5）

- [ ] 性能基准测试；
- [ ] 更新 README、架构图、操作手册；
- [ ] CI/CD 接入（build / vet / test / lint / docker）。

---

## 10. 风险与应对

| 风险 | 影响 | 应对 |
|------|------|------|
| 目标站点反爬升级 | 高 | 熔断+代理池+多 UA，失败即停止并告警 |
| 解析规则失效 | 高 | 配置化 selector，支持热更新，失败保存原始 HTML |
| 并发导致 IP 被封 | 中 | 全局限流、单域名并发限制、随机 jitter |
| 存储单点故障 | 中 | 多存储后端、本地 JSON 兜底 |
| 测试依赖外部网络 | 中 | 全部使用 httptest + fixture |

---

## 11. 附录

### 11.1 推荐依赖

| 用途 | 库 |
|------|-----|
| 配置校验 | github.com/go-playground/validator/v10 |
| 日志 | go.uber.org/zap + gopkg.in/natefinch/lumberjack.v2 |
| 测试 | github.com/stretchr/testify |
| HTTP Mock | net/http/httptest |
| DOM 解析 | github.com/PuerkitoBio/goquery |
| SQLite | github.com/mattn/go-sqlite3 |
| MySQL | github.com/go-sql-driver/mysql |
| Prometheus | github.com/prometheus/client_golang |

### 11.2 命名规范

- 包名：小写单数，如 `fetcher`、`parser`、`storage`；
- 接口名：`Fetcher`、`Parser`、`Storage`；
- 实现名：`HTTPFetcher`、`HTMLParser`、`JSONStorage`；
- 错误变量：`ErrXXX`，使用 `errors.Is` / `errors.As`。

### 11.3 文档产出物

- 本 PRD；
- `docs/architecture.md`（架构图与模块说明）；
- `docs/operation.md`（部署与运维手册）；
- `docs/testing.md`（测试与 Mock 指南）。

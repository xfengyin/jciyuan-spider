# jciyuan-spider - 企业级动漫爬虫

<p align="center">
  <strong>Go</strong> · <strong>Plugin Architecture</strong> · <strong>Middleware Chain</strong>
</p>

---

**jciyuan-spider** 是一款面向 [www.jciyuan.com](https://www.jciyuan.com) 的动漫信息爬虫：抓取动漫详情页，解析标题与剧集列表，可选抓取每集的 M3U8 播放地址，并以多种后端持久化。基于接口 + SPI 插件架构设计，Fetcher / Parser / Storage 均可替换实现。

## 功能特性

| 特性 | 描述 |
|------|------|
| 🧩 **插件架构** | Fetcher / Parser / Storage 接口 + 注册式 SPI，按配置装配 |
| ⛓️ **中间件链** | trace → metrics → logging → rate_limit → retry → circuit_breaker → proxy_rotate，顺序可配置 |
| 🛡️ **抗反爬** | Random UA、Referer、Cookie 保持、URL 白名单、robots.txt 检查 |
| ⚡ **并发控制** | WorkerPool、信号量限流、指数退避重试、熔断器 |
| 📦 **多存储后端** | JSON（默认）、SQLite、MySQL、S3，外加内存缓存装饰器 |
| 🔄 **断点续爬** | 爬取状态持久化，中断后可恢复；支持增量合并保留已抓取的 M3U8 |
| 📊 **可观测性** | 内存 / Prometheus 指标、/healthz 健康检查、traceId 全链路日志 |
| ⚙️ **配置化** | YAML 配置 + `JCIYUAN_*` 环境变量覆盖 + 命令行 flag |
| 📝 **日志** | zap + lumberjack，分级输出、文件轮转 |

## 快速开始

### 构建与运行

```bash
git clone https://github.com/xfengyin/jciyuan-spider.git
cd jciyuan-spider
go mod tidy
go build -o jciyuan-spider .

# 使用默认配置运行（config/config.yaml）
./jciyuan-spider

# 指定动漫 ID、开启增量更新与调试日志
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

## 项目结构

```
jciyuan-spider/
├── main.go                     # 命令行入口：flag、信号、健康服务
├── config/config.yaml          # 默认配置
└── internal/
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

## 输出示例

默认（JSON 后端）输出到 `./output/`，字段与配置的 extractor 一致（默认提取 title 与 episodes）：

```json
{
  "id": 37439,
  "title": "一人之下第六季",
  "episode_num": 17,
  "episodes": [
    {
      "number": 1,
      "title": "第01集",
      "url": "https://www.jciyuan.com/acgplay/37439-4-1.html",
      "is_crawled": false
    }
  ]
}
```

开启 `crawl.incremental` 后重复运行会与旧数据合并；开启 `storage.output.save_m3u8` 后会并发抓取每集播放页并填充 `m3u8_url`。

## 开发

```bash
go build ./...
go vet ./...
go test ./...
```

## 注意事项

⚠️ **免责声明**
- 本工具仅供学习研究使用
- 请遵守网站的 robots.txt 和使用条款
- 爬取行为需自行承担法律责任
- 建议设置合理的请求间隔（>=1000ms）

## License

MIT License - see [LICENSE](LICENSE) for details.

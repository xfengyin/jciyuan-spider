# jciyuan-spider v2.0 - 企业级动漫爬虫

<p align="center">
  <strong>Go</strong> · <strong>Enterprise Grade</strong> · <strong>SUPERSpider</strong>
</p>

---

**jciyuan-spider v2.0** 是一款企业级动漫爬虫，基于 SUPERSpider 理念设计，支持抗反爬、并发控制、断点续爬、统计监控等功能。

## 功能特性

| 特性 | 描述 |
|------|------|
| 🛡️ **抗反爬** | Random UA、Referer、Cookie保持、403检测 |
| ⚡ **并发控制** | goroutine池、限流、重试机制 |
| 📦 **存储** | JSON文件、SQLite（可选）、M3U8播放列表 |
| 🔄 **断点续爬** | 保存爬取状态、中断后可恢复 |
| 📊 **统计监控** | 请求统计、成功率、带宽监控 |
| 🏗️ **分层架构** | Fetcher/Parser/Storage 完全解耦 |
| ⚙️ **配置化** | YAML配置、环境变量覆盖 |
| 📝 **日志系统** | 分级日志、文件输出、优雅日志 |

## 快速开始

### 安装

```bash
git clone https://github.com/xfengyin/jciyuan-spider.git
cd jciyuan-spider-v2
go mod tidy
go build -o jciyuan-spider main.go
```

### 基本用法

```bash
# 爬取默认动漫
./jciyuan-spider

# 指定动漫ID
./jciyuan-spider -id 37439

# 指定URL
./jciyuan-spider -url "https://www.jciyuan.com/acgdetail/37439.html"

# 设置请求间隔
./jciyuan-spider -delay 2000

# 启用断点续爬
./jciyuan-spider -resume

# 增量更新
./jciyuan-spider -incremental

# 调试模式
./jciyuan-spider -debug
```

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-config` | config/config.yaml | 配置文件路径 |
| `-id` | 37439 | 动漫ID |
| `-url` | - | 直接指定URL |
| `-delay` | 1000 | 请求间隔(ms) |
| `-output` | ./output | 输出目录 |
| `-resume` | false | 启用断点续爬 |
| `-incremental` | false | 增量更新 |
| `-stats` | true | 显示统计 |
| `-debug` | false | 调试模式 |

## 项目结构

```
jciyuan-spider-v2/
├── main.go              # 主程序入口
├── config/
│   └── config.yaml      # 配置文件
├── cmd/
│   └── config/
│       └── config.go    # 配置加载
├── crawler/
│   └── fetcher.go       # HTTP请求器
├── parser/
│   └── parser.go        # HTML解析器
├── model/
│   └── model.go         # 数据结构
├── storage/
│   └── storage.go       # 存储层
├── log/
│   └── logger.go        # 日志系统
├── utils/
│   └── utils.go         # 工具函数
└── README.md
```

## 配置说明

```yaml
# config/config.yaml
spider:
  base_url: "https://www.jciyuan.com"
  delay: 1000        # 请求间隔(ms)
  timeout: 10        # 超时(s)
  max_retry: 3       # 最大重试
  concurrency: 3      # 并发数

anticrawler:
  enable_proxy: false
  random_ua: true
  keep_cookie: true
  user_agents:
    - "Mozilla/5.0 ..."

crawl:
  anime_id: 37439
  resume: true
  incremental: false

storage:
  output_dir: "./output"
  save_json: true
  save_sqlite: false
  db_path: "./data/spider.db"
  save_m3u8: false

log:
  level: "info"
  file: "./logs/spider.log"
  console: true
```

## 反爬措施

| 措施 | 说明 |
|------|------|
| Random UA | 随机选择User-Agent |
| Referer | 模拟页面跳转 |
| Cookie保持 | 保持会话Cookie |
| 限流 | 控制请求频率 |
| 重试 | 失败自动重试 |
| 403检测 | 检测访问被禁 |
| 验证码检测 | 识别验证码页面 |

## 输出示例

```json
{
  "id": 37439,
  "title": "一人之下第六季",
  "year": "2026",
  "region": "大陆",
  "tags": ["热血", "冒险", "爆笑", "国产动漫"],
  "cover_image": "https://...",
  "description": "张楚岚在碧游村事件后...",
  "update_date": "2026-04-19",
  "episode_num": 17,
  "episodes": [
    {
      "number": 1,
      "title": "第01集",
      "url": "https://www.jciyuan.com/acgplay/37439-4-1.html",
      "is_vip": false
    }
  ]
}
```

## 注意事项

⚠️ **免责声明**
- 本工具仅供学习研究使用
- 请遵守网站的 robots.txt 和使用条款
- 爬取行为需自行承担法律责任
- 建议设置合理的请求间隔（>=1000ms）

## License

MIT License - see [LICENSE](LICENSE) for details.

---

**Made with ❤️ by Kongming Agent**

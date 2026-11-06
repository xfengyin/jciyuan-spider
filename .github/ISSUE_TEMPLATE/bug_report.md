---
name: Bug 反馈
about: 报告抓取、解析或引擎调度中的问题
title: "[Bug] "
labels: bug
assignees: xfengyin
---

## 问题描述

<!-- 一句话说清哪里不对 -->

## 复现方式

```go
// 最小可复现代码；如果是配置驱动，请贴 YAML
```

## 目标页面与 Crawler

- Crawler 类型：<!-- 内置 demo/jciyuan/rss，或自定义实现的名字 -->
- 目标 URL 特征：<!-- 不需要贴完整 URL，说清站点类型即可（HTML / RSS / JSON API） -->
- 并发数 / 重试次数：<!-- Options 里的 Concurrency、MaxRetry -->

## 期望行为

## 实际行为

```text
（粘贴完整报错或异常输出）
```

## 环境信息

- 版本 / commit：<!-- git rev-parse --short HEAD -->
- Go 版本：<!-- go version -->
- 操作系统：
- 是否命中熔断或限速：<!-- 日志里有没有 circuit breaker / retry 相关行 -->

---

<!-- 如果问题与某个具体站点强相关，请说明该站点的 robots.txt 与 ToS 是否允许抓取 —— 本框架不用于绕过访问控制。 -->

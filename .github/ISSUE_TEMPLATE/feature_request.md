---
name: 功能建议
about: 提出新的 Crawler、中间件或引擎能力
title: "[Feature] "
labels: enhancement
assignees: xfengyin
---

## 需求场景

<!-- 你现在的做法是什么，哪里别扭 -->

## 期望方案

<!-- 新增一个 Crawler 实现？一个 fetcher 中间件？还是引擎选项？ -->

## 属于哪一层

- [ ] `Crawler` 接口的实现（站点适配）
- [ ] `Engine` 调度（并发 / 重试 / 限速 / 输出）
- [ ] fetcher 中间件（重试、熔断、trace、脱敏）
- [ ] 存储 / 断点续爬
- [ ] 文档与示例

## 备选方案

<!-- 考虑过但不合适的方式，以及为什么不合适 -->

## 是否愿意提 PR

- [ ] 愿意，我可以按 `docs/architecture.md` 的扩展点自己实现
- [ ] 只是想先讨论方向

---

<!-- 提醒：本框架的公开 API 刻意只有 Crawler + Engine 两个类型，新增能力的默认位置是 internal/ 下的参考实现，不要为了一个用例扩大公开接口。 -->

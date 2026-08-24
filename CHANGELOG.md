# Changelog

本仓库所有值得注意的变更均记录于此。格式参考
[Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循
[语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased] — v0.3.0 候选

### Added

- 新增 [`examples/json-api`](examples/json-api)：抓取 JSON API（如
  `httpbin.org/json` 或任意 JSON 接口），Parse 用 `encoding/json` 解析
  （优先匹配 slideshow 结构，不匹配回退为原始 map），Extract 将 JSON 数组
  展开为多条结构化 item；含本地 mock server 单测（无需网络）。
- 文档：`docs/examples.md` 新增 json-api 详解；`examples/README.md`、
  `README.md`/`README_zh.md` 示例表同步。

## [v0.2.0] - 2026-08-24

### Added

- 新增 [`examples/markdown`](examples/markdown)：抓取 HTML 并导出 Markdown
  文件（仅标准库 `regexp`/`html` 的轻量 HTML→MD 转换器，不引入第三方库）。
- 文档完善：`docs/examples.md` 新增 markdown 详解；示例索引与 README 同步；
  M5 里程碑注记与 v0.2.0 发布准备说明。

## [v0.1.0] - 2026-08-24

### Added

- Release 流水线（`.github/workflows/release.yml`）：tag `v*` 触发，go build
  矩阵（linux-amd64 / windows-amd64 / darwin-arm64），zip/tar.gz 压缩 +
  sha256sum 校验和，`softprops/action-gh-release@v2` 创建 Release，权限最小化。
- README/README_zh 新增 pkg.go.dev badge（`crawler` 包）与 Release badge。
- 新增 [`examples/csv`](examples/csv)：将 crawler Engine 抓取结果导出为 CSV
  （字段并集表头 + `encoding/csv` 自动转义），含单测。

---

> 历史里程碑（M1-M3，含在 v0.1.0 之前的迭代）：
> M1 通用 Crawler 接口 + Engine + SPI（`crawler/`）；M2 默认 HTTP 实现与 CI；
> M3 文档站（`docs/`）+ RSS 示例 + GoDoc。

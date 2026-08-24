# Example: csv

把 `crawler.Engine` 的抓取结果导出为 CSV：默认 HTTP 爬虫抓取 → `ItemsToCSV` 生成 CSV
（字段并集排序作表头，`encoding/csv` 自动转义逗号/引号/换行）。

## 运行

```bash
# 在仓库根目录
go run ./examples/csv -url https://example.com
go run ./examples/csv -url https://example.com,https://example.org -output ./output/csv/result.csv
```

## 参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `-url` | - | 抓取目标 URL，多个用逗号分隔 |
| `-output` | ./output/csv/result.csv | CSV 输出路径 |
| `-concurrency` | 3 | Engine 并发数 |
| `-max-retry` | 2 | 失败重试次数 |
| `-quiet` | false | 安静模式 |

## 实现要点

- `run()` 封装「Engine 抓取」，`ItemsToCSV()` 独立成函数便于复用与单测。
- 表头 = 所有 Item 字段名的排序并集；缺失字段留空，`[]string` 值以 `; ` 合并。
- 详见 [docs/examples.md](../../docs/examples.md)。

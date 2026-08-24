# Example: xml

抓取 XML/RSS/Atom 源并导出 CSV（复合场景：XML 输入 → 结构化 item → CSV 输出）。
与 `examples/rss`（JSONL 输出）区分：本示例把抓取结果直接写成 CSV 表格。

## 运行

```bash
# 在仓库根目录
go run ./examples/xml -url examples/xml/sample.xml              # 本地文件（离线演示）
go run ./examples/xml -url https://example.org/feed.xml         # 远程 RSS/Atom
go run ./examples/xml -url examples/xml/sample.xml -output out.csv
```

## 参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `-url` | - | XML/RSS/Atom 源：http(s):// 远程 URL 或本地文件路径 |
| `-output` | ./output/xml/feed.csv | CSV 输出路径 |
| `-quiet` | false | 安静模式 |

## 实现要点

- Parse 用 `encoding/xml` 同时支持 RSS 2.0 与 Atom，Extract 输出统一字段
  `{title, link, description, pub_date}`。
- Fetch 识别本地文件（含 `file://`）与远程 URL，离线演示无需网络。
- `ItemsToCSV` 与 `examples/csv` 保持一致：字段并集排序表头 + `encoding/csv` 自动转义。
- 详见 [docs/examples.md](../../docs/examples.md)。

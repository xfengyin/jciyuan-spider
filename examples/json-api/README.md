# Example: json-api

抓取 JSON API 并输出结构化 item：Fetch 用默认 HTTP 抓取器，Parse 用
`encoding/json` 解析（优先匹配 slideshow 结构，不匹配回退为原始 map），
Extract 把 JSON 数组（slides）展开为多条结构化 item。

## 运行

```bash
# 在仓库根目录
go run ./examples/json-api -url https://httpbin.org/json
go run ./examples/json-api -url https://httpbin.org/json -output ./output/json-api
go run ./examples/json-api -url https://api.example.com/v1/items   # 任意 JSON 接口（回退为整包 item）
```

## 参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `-url` | https://httpbin.org/json | JSON API 地址，多个用逗号分隔 |
| `-output` | ./output/json-api | 输出目录（JSON Lines） |
| `-concurrency` | 3 | Engine 并发数 |
| `-max-retry` | 2 | 失败重试次数 |
| `-quiet` | false | 安静模式 |

## 实现要点

- `slideshowResp` 对应 httpbin.org/json 结构；字段用 `json` tag 绑定。
- 结构不匹配时回退 `map[string]any`，任意 JSON API 均可输出 item。
- 本地 mock server 单测：slideshow 展开 / 回退 / 非法 JSON。
- 详见 [docs/examples.md](../../docs/examples.md)。

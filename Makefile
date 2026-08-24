# jciyuan-spider 开发常用命令（make build / make test / make vet / make fmt / make clean）
.PHONY: build test vet fmt fmt-check clean tidy version run-demo run-jciyuan run-rss run-csv run-markdown run-json-api run-xml

BIN     := jciyuan-spider
VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build: ## 编译根 CLI 到 ./$(BIN)（VERSION 可覆盖，如 make build VERSION=v0.5.0）
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) .

test: ## 运行全部单元测试
	go test ./...

vet: ## 静态检查
	go vet ./...

fmt: ## gofmt 格式化全部 Go 源码
	gofmt -w .

fmt-check: ## 检查 gofmt（框架包/示例/新增代码；有未格式化文件则失败）
	@test -z "$$(gofmt -l crawler examples internal/version main.go)" || { gofmt -l crawler examples internal/version main.go; echo "存在未格式化文件，请运行 make fmt"; exit 1; }

tidy: ## 整理 go.mod / go.sum
	go mod tidy

clean: ## 清理构建产物与运行时输出
	rm -rf $(BIN) dist output

version: ## 打印构建版本信息
	@echo "jciyuan-spider $(VERSION) (commit: $(COMMIT), built: $(DATE))"

# ---------- 示例快速运行 ----------
run-demo: ## 运行 demo 示例（配置驱动通用爬虫）
	go run ./examples/demo -config examples/demo/config.yaml

run-jciyuan: ## 运行 jciyuan 示例（动漫爬虫）
	go run ./examples/jciyuan -id 37439

run-rss: ## 运行 rss 示例（RSS/XML → JSONL）
	go run ./examples/rss -url examples/rss/sample.xml

run-csv: ## 运行 csv 示例（抓取结果 → CSV）
	go run ./examples/csv -url https://example.com

run-markdown: ## 运行 markdown 示例（HTML → Markdown）
	go run ./examples/markdown -url https://example.com

run-json-api: ## 运行 json-api 示例（JSON API → item）
	go run ./examples/json-api -url https://httpbin.org/json

run-xml: ## 运行 xml 示例（XML/RSS → CSV）
	go run ./examples/xml -url examples/xml/sample.xml

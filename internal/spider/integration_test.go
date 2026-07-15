// Package spider 的集成测试，使用 httptest 本地服务器端到端跑通 Fetch -> Parse -> Save。
package spider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	httpfetcher "jciyuan-spider-v2/internal/fetcher/http"
	"jciyuan-spider-v2/internal/logger"
	"jciyuan-spider-v2/internal/metrics"
	"jciyuan-spider-v2/internal/model"
	htmlparser "jciyuan-spider-v2/internal/parser/html"
	"jciyuan-spider-v2/internal/resume"
	jsonstorage "jciyuan-spider-v2/internal/storage/json"
	"jciyuan-spider-v2/internal/worker"
)

// newIntegrationServer 启动本地测试服务器，返回服务端点。
func newIntegrationServer(t *testing.T) string {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /admin/\nAllow: /\n"))
		case "/acgdetail/37439.html":
			serveFixture(t, w, "detail_37439.html")
		case "/acgplay/37439-1-1.html":
			serveFixture(t, w, "episode_play.html")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)
	return ts.URL
}

// serveFixture 从 testdata 目录读取指定 HTML 文件并写入响应。
func serveFixture(t *testing.T, w http.ResponseWriter, name string) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// integrationConfig 返回针对本地测试服务器的 Spider 配置。
func integrationConfig(t *testing.T, baseURL string) *model.Config {
	return &model.Config{
		App: model.AppConfig{
			Name:          "jciyuan-spider-v3",
			Mode:          "cli",
			TraceIDHeader: "X-Request-ID",
		},
		Spider: model.SpiderConfig{
			BaseURL:          baseURL,
			DetailURLPattern: "{{base_url}}/acgdetail/{{id}}.html",
			Concurrency:      2,
			QueueSize:        10,
			Timeout:          5,
			MaxRetry:         1,
		},
		Anticrawler: model.AnticrawlerConfig{
			RandomUA:       false,
			KeepCookie:     false,
			UserAgents:     []string{"jciyuan-spider-v3-test"},
			RefererPolicy:  "origin",
			RobotsTxtCheck: true,
		},
		Fetcher: model.FetcherConfig{
			Type: "http",
			HTTP: model.HTTPFetcherConfig{
				Timeout:         5,
				MaxRetry:        1,
				FollowRedirects: true,
				MaxBodySize:     10 * 1024 * 1024,
			},
		},
		Parser: model.ParserConfig{
			Type: "html",
			HTML: model.HTMLParserConfig{
				Encoding: "utf-8",
				Extractors: []model.ExtractorConfig{
					{
						Field:    "title",
						Selector: model.SelectorConfig{Type: "css", Value: "h1.title"},
						Processors: []model.ProcessorConfig{
							{Type: "trim"},
						},
					},
					{
						Field:       "tags",
						Selector:    model.SelectorConfig{Type: "css", Value: "span.tag"},
						Multiple:    true,
						Deduplicate: true,
					},
					{
						Field:       "episodes",
						Selector:    model.SelectorConfig{Type: "regex", Value: `/acgplay/(\d+)-(\d+)-(\d+)\.html`},
						Multiple:    true,
						Deduplicate: true,
					},
				},
			},
		},
		Storage: model.StorageConfig{
			Type: "json",
			JSON: model.JSONStorageConfig{OutputDir: t.TempDir()},
			Output: model.OutputConfig{
				SaveJSON:    true,
				SaveM3U8:    true,
				SaveRawHTML: false,
			},
		},
		Crawl: model.CrawlConfig{
			AnimeID:     37439,
			Resume:      false,
			Incremental: false,
			MaxEpisodes: 1,
		},
		Middlewares: []model.MiddlewareItem{
			{Name: "trace"},
			{Name: "metrics"},
			{Name: "logging"},
			{Name: "retry"},
		},
		Metrics: model.MetricsConfig{
			Enabled: true,
			Backend: "memory",
		},
	}
}

// buildIntegrationSpider 手动装配 Spider，避免引入 di 包导致循环依赖。
func buildIntegrationSpider(t *testing.T, cfg *model.Config) *Spider {
	log := logger.NewFromZap(zap.NewNop())
	m := metrics.NewMemoryMetrics()

	fetcherImpl, err := httpfetcher.NewHTTPFetcher(cfg.Fetcher, cfg.Anticrawler, cfg.Spider, cfg.Middlewares, m, log)
	require.NoError(t, err)

	parserImpl, err := htmlparser.New(cfg.Parser)
	require.NoError(t, err)

	storageImpl, err := jsonstorage.NewJSONStorage(cfg.Storage.JSON.OutputDir)
	require.NoError(t, err)

	pool := worker.NewPool(cfg.Spider.Concurrency, cfg.Spider.QueueSize)
	resumeMgr := resume.NewManager(storageImpl)

	return New(Options{
		Config:  cfg,
		Fetcher: fetcherImpl,
		Parser:  parserImpl,
		Storage: storageImpl,
		Resume:  resumeMgr,
		Pool:    pool,
		Metrics: m,
		Logger:  log,
	})
}

// TestSpiderIntegrationFetchParseSave 验证端到端 Fetch -> Parse -> Save 流程。
func TestSpiderIntegrationFetchParseSave(t *testing.T) {
	baseURL := newIntegrationServer(t)
	cfg := integrationConfig(t, baseURL)

	s := buildIntegrationSpider(t, cfg)
	defer s.Close()

	require.NoError(t, s.Run(context.Background()))
	assert.Equal(t, StateCompleted, s.State())

	stats := s.GetStats()
	assert.Equal(t, int64(1), stats.StorageSaveCount)

	outputFile := filepath.Join(cfg.Storage.JSON.OutputDir, "37439.json")
	_, err := os.Stat(outputFile)
	require.NoError(t, err)
}

// TestSpiderIntegrationRobotsDisallowed 验证 robots.txt 禁止路径会被拦截。
func TestSpiderIntegrationRobotsDisallowed(t *testing.T) {
	baseURL := newIntegrationServer(t)
	cfg := integrationConfig(t, baseURL)
	cfg.Crawl.AnimeID = 37439
	cfg.Spider.DetailURLPattern = "{{base_url}}/admin/detail/{{id}}.html"

	s := buildIntegrationSpider(t, cfg)
	defer s.Close()

	err := s.Run(context.Background())
	require.Error(t, err)
}

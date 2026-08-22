// Package health 提供爬虫健康检查端点，支持独立暴露 /healthz，
// 并在配置 Prometheus 端口时与 /metrics 共用一个 HTTP server。
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"jciyuan-spider/internal/logger"
	"jciyuan-spider/internal/metrics"
	"jciyuan-spider/internal/model"
)

// Checker 健康检查服务，负责启动 HTTP server 并响应 /healthz。
type Checker struct {
	cfg       *model.Config
	log       logger.Logger
	metrics   metrics.Metrics
	startTime time.Time
	stateFunc func() string
	server    *http.Server
	mu        sync.Mutex
	started   bool
}

// NewChecker 创建健康检查服务实例。
// stateFunc 用于动态获取当前爬虫状态字符串。
func NewChecker(
	cfg *model.Config,
	log logger.Logger,
	m metrics.Metrics,
	stateFunc func() string,
	startTime time.Time,
) *Checker {
	if stateFunc == nil {
		stateFunc = func() string { return "unknown" }
	}
	return &Checker{
		cfg:       cfg,
		log:       log,
		metrics:   m,
		startTime: startTime,
		stateFunc: stateFunc,
	}
}

// Start 启动 HTTP 健康检查服务。
// 监听端口优先使用 MetricsConfig.Prometheus.Port；当 backend 为 prometheus 时，
// 同时在该 server 上挂载 /metrics，实现 health 与 metrics 共端口。
func (c *Checker) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return nil
	}

	port := c.cfg.Metrics.Prometheus.Port
	if port <= 0 {
		port = 9090
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", c.healthzHandler)
	if c.cfg.Metrics.Backend == "prometheus" {
		path := c.cfg.Metrics.Prometheus.Path
		if path == "" {
			path = "/metrics"
		}
		mux.Handle(path, metrics.PrometheusHandler(c.metrics))
	}

	c.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	c.started = true

	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.log.Error("健康检查服务异常退出", logger.Err(err))
		}
	}()

	c.log.Info("健康检查服务已启动",
		logger.Int("port", port),
		logger.String("metrics_backend", c.cfg.Metrics.Backend),
	)
	return nil
}

// Stop 优雅关闭健康检查服务。
func (c *Checker) Stop(ctx context.Context) error {
	c.mu.Lock()
	s := c.server
	c.server = nil
	c.started = false
	c.mu.Unlock()

	if s == nil {
		return nil
	}
	return s.Shutdown(ctx)
}

// healthResponse 是 /healthz 返回的 JSON 结构。
type healthResponse struct {
	Status      string        `json:"status"`
	Uptime      string        `json:"uptime"`
	StartTime   time.Time     `json:"start_time"`
	SpiderState string        `json:"spider_state"`
	Config      configSummary `json:"config"`
	Stats       model.Stats   `json:"stats"`
}

// configSummary 是配置摘要，避免暴露敏感字段。
type configSummary struct {
	AppName string `json:"app_name"`
	Mode    string `json:"mode"`
	BaseURL string `json:"base_url"`
	Fetcher string `json:"fetcher"`
	Parser  string `json:"parser"`
	Storage string `json:"storage"`
	Metrics string `json:"metrics"`
}

// healthzHandler 处理 /healthz 请求，返回当前状态、uptime、配置摘要与统计信息。
func (c *Checker) healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := c.metrics.GetStats()
	resp := healthResponse{
		Status:      "ok",
		Uptime:      time.Since(c.startTime).Round(time.Second).String(),
		StartTime:   c.startTime,
		SpiderState: c.stateFunc(),
		Config: configSummary{
			AppName: c.cfg.App.Name,
			Mode:    c.cfg.App.Mode,
			BaseURL: c.cfg.Spider.BaseURL,
			Fetcher: c.cfg.Fetcher.Type,
			Parser:  c.cfg.Parser.Type,
			Storage: c.cfg.Storage.Type,
			Metrics: c.cfg.Metrics.Backend,
		},
		Stats: stats,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		c.log.Warn("健康检查响应编码失败", logger.Err(err))
	}
}

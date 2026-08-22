// Package config 负责配置加载、默认值应用、环境变量覆盖与基础校验。
package config

import (
	"fmt"
	"os"
	"time"

	"jciyuan-spider/internal/model"

	"gopkg.in/yaml.v3"
)

// Loader 配置加载器
type Loader struct {
	configPath string
}

// NewLoader 创建配置加载器
func NewLoader(configPath string) *Loader {
	return &Loader{configPath: configPath}
}

// Load 加载配置文件
func (l *Loader) Load() (*model.Config, error) {
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &model.Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	l.applyDefaults(cfg)

	if err := l.validate(cfg); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return cfg, nil
}

// applyDefaults 应用默认值
func (l *Loader) applyDefaults(cfg *model.Config) {
	if cfg.App.Name == "" {
		cfg.App.Name = "jciyuan-spider-v3"
	}
	if cfg.App.Mode == "" {
		cfg.App.Mode = "cli"
	}
	if cfg.App.TraceIDHeader == "" {
		cfg.App.TraceIDHeader = "X-Request-ID"
	}

	if cfg.Spider.BaseURL == "" {
		cfg.Spider.BaseURL = "https://www.jciyuan.com"
	}
	if cfg.Spider.DetailURLPattern == "" {
		cfg.Spider.DetailURLPattern = "{{base_url}}/acgdetail/{{id}}.html"
	}
	if cfg.Spider.Delay == 0 {
		cfg.Spider.Delay = 1000
	}
	if cfg.Spider.Timeout == 0 {
		cfg.Spider.Timeout = 10
	}
	if cfg.Spider.MaxRetry == 0 {
		cfg.Spider.MaxRetry = 3
	}
	if cfg.Spider.Concurrency == 0 {
		cfg.Spider.Concurrency = 3
	}
	if cfg.Spider.QueueSize == 0 {
		cfg.Spider.QueueSize = 100
	}

	if cfg.Fetcher.Type == "" {
		cfg.Fetcher.Type = "http"
	}
	if cfg.Fetcher.HTTP.Timeout == 0 {
		cfg.Fetcher.HTTP.Timeout = cfg.Spider.Timeout
	}
	if cfg.Fetcher.HTTP.MaxRetry == 0 {
		cfg.Fetcher.HTTP.MaxRetry = cfg.Spider.MaxRetry
	}
	if cfg.Fetcher.HTTP.MaxBodySize == 0 {
		cfg.Fetcher.HTTP.MaxBodySize = 50 * 1024 * 1024
	}
	if cfg.Fetcher.HTTP.Transport.MaxIdleConns == 0 {
		cfg.Fetcher.HTTP.Transport.MaxIdleConns = 100
	}
	if cfg.Fetcher.HTTP.Transport.MaxConnsPerHost == 0 {
		cfg.Fetcher.HTTP.Transport.MaxConnsPerHost = 10
	}
	if cfg.Fetcher.HTTP.Transport.IdleConnTimeout == 0 {
		cfg.Fetcher.HTTP.Transport.IdleConnTimeout = 90 * time.Second
	}
	if cfg.Fetcher.HTTP.Transport.TLSHandshakeTimeout == 0 {
		cfg.Fetcher.HTTP.Transport.TLSHandshakeTimeout = 10 * time.Second
	}
	if cfg.Fetcher.Proxy.Strategy == "" {
		cfg.Fetcher.Proxy.Strategy = "round_robin"
	}
	if cfg.Fetcher.CircuitBreaker.FailureThreshold == 0 {
		cfg.Fetcher.CircuitBreaker.FailureThreshold = 5
	}
	if cfg.Fetcher.CircuitBreaker.ErrorRateThreshold == 0 {
		cfg.Fetcher.CircuitBreaker.ErrorRateThreshold = 0.5
	}
	if cfg.Fetcher.CircuitBreaker.WindowSize == 0 {
		cfg.Fetcher.CircuitBreaker.WindowSize = 10
	}
	if cfg.Fetcher.CircuitBreaker.OpenDuration == 0 {
		cfg.Fetcher.CircuitBreaker.OpenDuration = 30 * time.Second
	}
	if cfg.Fetcher.CircuitBreaker.HalfOpenRequests == 0 {
		cfg.Fetcher.CircuitBreaker.HalfOpenRequests = 1
	}

	if cfg.Parser.Type == "" {
		cfg.Parser.Type = "html"
	}
	if cfg.Parser.HTML.Encoding == "" {
		cfg.Parser.HTML.Encoding = "auto"
	}

	if cfg.Storage.Type == "" {
		cfg.Storage.Type = "json"
	}
	if cfg.Storage.JSON.OutputDir == "" {
		cfg.Storage.JSON.OutputDir = "./output"
	}
	if cfg.Storage.SQLite.DSN == "" {
		cfg.Storage.SQLite.DSN = "./data/spider.db"
	}

	if len(cfg.Anticrawler.UserAgents) == 0 {
		cfg.Anticrawler.UserAgents = defaultUserAgents()
	}
	if cfg.Anticrawler.RefererPolicy == "" {
		cfg.Anticrawler.RefererPolicy = cfg.Spider.BaseURL + "/"
	}

	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "text"
	}
	if cfg.Log.MaxSize == 0 {
		cfg.Log.MaxSize = 10
	}
	if cfg.Log.MaxBackups == 0 {
		cfg.Log.MaxBackups = 5
	}

	if cfg.Metrics.Backend == "" {
		cfg.Metrics.Backend = "memory"
	}
	if cfg.Metrics.Prometheus.Port == 0 {
		cfg.Metrics.Prometheus.Port = 9090
	}
	if cfg.Metrics.Prometheus.Path == "" {
		cfg.Metrics.Prometheus.Path = "/metrics"
	}
}

// validate 验证配置
func (l *Loader) validate(cfg *model.Config) error {
	if cfg.Spider.BaseURL == "" {
		return fmt.Errorf("base_url 不能为空")
	}
	if cfg.Spider.Delay < 100 {
		return fmt.Errorf("delay 不能小于 100 毫秒")
	}
	if cfg.Spider.Timeout < 1 {
		return fmt.Errorf("timeout 不能小于 1 秒")
	}
	if cfg.Spider.MaxRetry < 0 {
		return fmt.Errorf("max_retry 不能为负数")
	}
	if cfg.Spider.Concurrency < 1 {
		return fmt.Errorf("concurrency 至少为 1")
	}
	if cfg.Fetcher.Type == "" {
		return fmt.Errorf("fetcher.type 不能为空")
	}
	if cfg.Parser.Type == "" {
		return fmt.Errorf("parser.type 不能为空")
	}
	if cfg.Storage.Type == "" {
		return fmt.Errorf("storage.type 不能为空")
	}
	return nil
}

// defaultUserAgents 默认 UA 列表
func defaultUserAgents() []string {
	return []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// LoadFromEnv 从环境变量加载配置覆盖
func LoadFromEnv(cfg *model.Config) {
	if baseURL := os.Getenv("JCIYUAN_BASE_URL"); baseURL != "" {
		cfg.Spider.BaseURL = baseURL
	}
	if delay := os.Getenv("JCIYUAN_DELAY"); delay != "" {
		var d int
		fmt.Sscanf(delay, "%d", &d)
		if d > 0 {
			cfg.Spider.Delay = d
		}
	}
	if timeout := os.Getenv("JCIYUAN_TIMEOUT"); timeout != "" {
		var t int
		fmt.Sscanf(timeout, "%d", &t)
		if t > 0 {
			cfg.Spider.Timeout = t
			cfg.Fetcher.HTTP.Timeout = t
		}
	}
	if ua := os.Getenv("JCIYUAN_USER_AGENT"); ua != "" {
		cfg.Anticrawler.UserAgents = []string{ua}
	}
	if backend := os.Getenv("JCIYUAN_METRICS_BACKEND"); backend != "" {
		cfg.Metrics.Backend = backend
	}
}

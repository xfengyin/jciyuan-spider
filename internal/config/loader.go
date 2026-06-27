// Package config - 配置加载模块
package config

import (
	"fmt"
	"os"

	"jciyuan-spider-v2/internal/model"

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
	if cfg.Spider.BaseURL == "" {
		cfg.Spider.BaseURL = "https://www.jciyuan.com"
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
	if len(cfg.Anticrawler.UserAgents) == 0 {
		cfg.Anticrawler.UserAgents = defaultUserAgents()
	}
	if cfg.Storage.OutputDir == "" {
		cfg.Storage.OutputDir = "./output"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
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
	if ua := os.Getenv("JCIYUAN_USER_AGENT"); ua != "" {
		cfg.Anticrawler.UserAgents = []string{ua}
	}
}

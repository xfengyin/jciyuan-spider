// Package config - 配置加载模块
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"jciyuan-spider-v2/model"
)

// ============================================================================
// 配置加载器
// ============================================================================

// Loader 配置加载器
type Loader struct {
	configPath string
}

// NewLoader 创建配置加载器
func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
	}
}

// Load 加载配置
func (l *Loader) Load() (*model.Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	
	// 手动解析YAML（简化版）
	config, err := parseYAML(string(data))
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	
	// 应用默认值
	l.applyDefaults(config)
	
	// 验证配置
	if err := l.validate(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}
	
	return config, nil
}

// parseYAML 简化版YAML解析
func parseYAML(content string) (*model.Config, error) {
	config := &model.Config{}
	
	lines := strings.Split(content, "\n")
	var currentSection string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// 检测section
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			currentSection = strings.TrimSuffix(line, ":")
			continue
		}
		
		// 解析键值对
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "-") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			
			// 根据section和key设置值
			switch currentSection {
			case "spider":
				switch key {
				case "base_url":
					config.Spider.BaseURL = value
				case "delay":
					config.Spider.Delay, _ = strconv.Atoi(value)
				case "timeout":
					config.Spider.Timeout, _ = strconv.Atoi(value)
				case "max_retry":
					config.Spider.MaxRetry, _ = strconv.Atoi(value)
				case "concurrency":
					config.Spider.Concurrency, _ = strconv.Atoi(value)
				}
			case "anticrawler":
				switch key {
				case "enable_proxy":
					config.Anticrawler.EnableProxy = value == "true"
				case "random_ua":
					config.Anticrawler.RandomUA = value == "true"
				case "keep_cookie":
					config.Anticrawler.KeepCookie = value == "true"
				}
			case "crawl":
				switch key {
				case "anime_id":
					config.Crawl.AnimeID, _ = strconv.ParseInt(value, 10, 64)
				case "resume":
					config.Crawl.Resume = value == "true"
				case "incremental":
					config.Crawl.Incremental = value == "true"
				case "max_episodes":
					config.Crawl.MaxEpisodes, _ = strconv.Atoi(value)
				}
			case "storage":
				switch key {
				case "output_dir":
					config.Storage.OutputDir = value
				case "save_json":
					config.Storage.SaveJSON = value == "true"
				case "save_sqlite":
					config.Storage.SaveSQLite = value == "true"
				case "db_path":
					config.Storage.DBPath = value
				case "save_m3u8":
					config.Storage.SaveM3U8 = value == "true"
				}
			case "log":
				switch key {
				case "level":
					config.Log.Level = value
				case "file":
					config.Log.File = value
				case "console":
					config.Log.Console = value == "true"
				case "max_size":
					config.Log.MaxSize, _ = strconv.Atoi(value)
				case "max_backups":
					config.Log.MaxBackups, _ = strconv.Atoi(value)
				}
			case "stats":
				switch key {
				case "enabled":
					config.Stats.Enabled = value == "true"
				case "interval":
					config.Stats.Interval, _ = strconv.Atoi(value)
				}
			}
		}
	}
	
	return config, nil
}

// applyDefaults 应用默认值
func (l *Loader) applyDefaults(config *model.Config) {
	// 爬虫配置
	if config.Spider.BaseURL == "" {
		config.Spider.BaseURL = "https://www.jciyuan.com"
	}
	if config.Spider.Delay == 0 {
		config.Spider.Delay = 1000
	}
	if config.Spider.Timeout == 0 {
		config.Spider.Timeout = 10
	}
	if config.Spider.MaxRetry == 0 {
		config.Spider.MaxRetry = 3
	}
	if config.Spider.Concurrency == 0 {
		config.Spider.Concurrency = 3
	}
	
	// 反爬配置
	if len(config.Anticrawler.UserAgents) == 0 {
		config.Anticrawler.UserAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		}
	}
	
	// 存储配置
	if config.Storage.OutputDir == "" {
		config.Storage.OutputDir = "./output"
	}
	if config.Storage.DBPath == "" {
		config.Storage.DBPath = "./data/spider.db"
	}
	
	// 日志配置
	if config.Log.Level == "" {
		config.Log.Level = "info"
	}
	if config.Log.File == "" {
		config.Log.File = "./logs/spider.log"
	}
	if config.Log.MaxSize == 0 {
		config.Log.MaxSize = 10
	}
	if config.Log.MaxBackups == 0 {
		config.Log.MaxBackups = 5
	}
}

// validate 验证配置
func (l *Loader) validate(config *model.Config) error {
	if config.Spider.BaseURL == "" {
		return fmt.Errorf("base_url不能为空")
	}
	
	if config.Spider.Delay < 100 {
		return fmt.Errorf("delay不能小于100毫秒")
	}
	
	if config.Spider.Timeout < 1 {
		return fmt.Errorf("timeout不能小于1秒")
	}
	
	if config.Spider.MaxRetry < 0 {
		return fmt.Errorf("max_retry不能为负数")
	}
	
	if config.Spider.Concurrency < 1 {
		return fmt.Errorf("concurrency不能小于1")
	}
	
	return nil
}

// Save 保存配置
func (l *Loader) Save(config *model.Config) error {
	// 简化实现：直接写入
	content := fmt.Sprintf(`spider:
  base_url: "%s"
  delay: %d
  timeout: %d
  max_retry: %d
  concurrency: %d

anticrawler:
  enable_proxy: %v
  random_ua: %v
  keep_cookie: %v

crawl:
  anime_id: %d
  resume: %v
  incremental: %v

storage:
  output_dir: "%s"
  save_json: %v
  save_sqlite: %v
  db_path: "%s"
  save_m3u8: %v

log:
  level: "%s"
  file: "%s"
  console: %v
`,
		config.Spider.BaseURL, config.Spider.Delay, config.Spider.Timeout,
		config.Spider.MaxRetry, config.Spider.Concurrency,
		config.Anticrawler.EnableProxy, config.Anticrawler.RandomUA,
		config.Anticrawler.KeepCookie,
		config.Crawl.AnimeID, config.Crawl.Resume, config.Crawl.Incremental,
		config.Storage.OutputDir, config.Storage.SaveJSON, config.Storage.SaveSQLite,
		config.Storage.DBPath, config.Storage.SaveM3U8,
		config.Log.Level, config.Log.File, config.Log.Console,
	)
	
	return os.WriteFile(l.configPath, []byte(content), 0644)
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv() *model.Config {
	config := &model.Config{}
	
	// 从环境变量覆盖配置
	if baseURL := os.Getenv("JCIYUAN_BASE_URL"); baseURL != "" {
		config.Spider.BaseURL = baseURL
	}
	
	if delay := os.Getenv("JCIYUAN_DELAY"); delay != "" {
		config.Spider.Delay, _ = strconv.Atoi(delay)
	}
	
	if ua := os.Getenv("JCIYUAN_USER_AGENT"); ua != "" {
		config.Anticrawler.UserAgents = []string{ua}
	}
	
	return config
}

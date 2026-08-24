// 最小示例：演示用框架抓取「任意站点」。
// 实现一个纯标准库 + goquery 的通用 Crawler（Fetch 用 net/http，Extract 用
// 配置驱动的 CSS/正则选择器），并通过 YAML 配置文件描述抓取任务。
//
// 运行（在仓库根目录）：
//
//	go run ./examples/demo -config examples/demo/config.yaml
//	go run ./examples/demo -config examples/demo/config.yaml -url https://example.com -output ./output/demo
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"jciyuan-spider/crawler"
	"jciyuan-spider/internal/version"

	"github.com/PuerkitoBio/goquery"
	"gopkg.in/yaml.v3"
)

// Config 抓取任务配置，对应 examples/demo/config.yaml。
type Config struct {
	Name        string        `yaml:"name"`
	StartURLs   []string      `yaml:"start_urls"`
	Rules       []Rule        `yaml:"rules"`
	Concurrency int           `yaml:"concurrency"`
	MaxRetry    int           `yaml:"max_retry"`
	Timeout     time.Duration `yaml:"timeout"`
	Delay       time.Duration `yaml:"delay"`
	OutputDir   string        `yaml:"output_dir"`
	UserAgent   string        `yaml:"user_agent"`
}

// Rule 一条字段抽取规则。
type Rule struct {
	Field    string   `yaml:"field"`
	Selector Selector `yaml:"selector"`
	Multiple bool     `yaml:"multiple"`
}

// Selector 选择器：type 为 css 或 regex。
type Selector struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
	Attr  string `yaml:"attr,omitempty"`
}

// demoCrawler 通用 HTML 爬虫：Fetch 用 net/http，Extract 用配置驱动的选择器。
type demoCrawler struct {
	cfg Config
	cli *http.Client
}

func newDemoCrawler(cfg Config) *demoCrawler {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &demoCrawler{
		cfg: cfg,
		cli: &http.Client{Timeout: timeout},
	}
}

func (c *demoCrawler) Name() string {
	if c.cfg.Name != "" {
		return c.cfg.Name
	}
	return "demo"
}

func (c *demoCrawler) Fetch(ctx context.Context, url string) (*crawler.Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.cfg.UserAgent != "" {
		req.Header.Set("User-Agent", c.cfg.UserAgent)
	}
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	return &crawler.Page{
		URL:        url,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
		Text:       string(body),
	}, nil
}

func (c *demoCrawler) Parse(_ context.Context, page *crawler.Page) (any, error) {
	return page.Text, nil // 保持原文，Extract 阶段再按规则抽取
}

func (c *demoCrawler) Extract(_ context.Context, parsed any) ([]crawler.Item, error) {
	html, ok := parsed.(string)
	if !ok {
		return nil, fmt.Errorf("解析结果类型异常: %T", parsed)
	}

	item := crawler.Item{}
	for _, rule := range c.cfg.Rules {
		values, err := c.selectValues(html, rule)
		if err != nil {
			return nil, fmt.Errorf("字段 %s 抽取失败: %w", rule.Field, err)
		}
		if len(values) == 0 {
			continue
		}
		if rule.Multiple {
			item[rule.Field] = values
		} else {
			item[rule.Field] = values[0]
		}
	}
	if len(item) == 0 {
		return nil, nil
	}
	return []crawler.Item{item}, nil
}

// selectValues 按规则抽取文本/属性值列表。
func (c *demoCrawler) selectValues(html string, rule Rule) ([]string, error) {
	switch rule.Selector.Type {
	case "css":
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return nil, err
		}
		var out []string
		doc.Find(rule.Selector.Value).Each(func(_ int, sel *goquery.Selection) {
			if rule.Selector.Attr != "" {
				if v, ok := sel.Attr(rule.Selector.Attr); ok {
					out = append(out, v)
				}
			} else {
				out = append(out, strings.TrimSpace(sel.Text()))
			}
		})
		return out, nil
	case "regex":
		re, err := regexp.Compile(rule.Selector.Value)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, m := range re.FindAllStringSubmatch(html, -1) {
			if len(m) > 1 {
				out = append(out, strings.TrimSpace(m[1]))
			} else {
				out = append(out, strings.TrimSpace(m[0]))
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("未知选择器类型: %s", rule.Selector.Type)
	}
}

var (
	configPath  = flag.String("config", "examples/demo/config.yaml", "任务配置文件路径")
	urlFlag     = flag.String("url", "", "覆盖 start_urls，多个 URL 用逗号分隔")
	outputFlag  = flag.String("output", "", "覆盖输出目录")
	versionFlag = flag.Bool("version", false, "打印版本信息")
)

func main() {
	flag.Parse()
	if *versionFlag {
		fmt.Printf("jciyuan-spider demo example %s\n", version.Version)
		os.Exit(0)
	}

	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取配置失败: %v\n", err)
		os.Exit(1)
	}
	cfg := Config{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "解析配置失败: %v\n", err)
		os.Exit(1)
	}
	if len(cfg.StartURLs) == 0 {
		fmt.Fprintln(os.Stderr, "配置中缺少 start_urls")
		os.Exit(1)
	}

	if *urlFlag != "" {
		cfg.StartURLs = strings.Split(*urlFlag, ",")
	}
	if *outputFlag != "" {
		cfg.OutputDir = *outputFlag
	}

	c := newDemoCrawler(cfg)
	eng := crawler.NewEngine(c, crawler.Options{
		Concurrency: cfg.Concurrency,
		Delay:       cfg.Delay,
		MaxRetry:    cfg.MaxRetry,
		Timeout:     cfg.Timeout,
		OutputDir:   cfg.OutputDir,
	})

	fmt.Printf("抓取 %d 个 URL（%s 示例）：%v\n", len(cfg.StartURLs), c.Name(), cfg.StartURLs)
	result, err := eng.Run(context.Background(), cfg.StartURLs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
		os.Exit(1)
	}

	for i, it := range result.Items {
		pretty, _ := json.MarshalIndent(it, "", "  ")
		fmt.Printf("--- item #%d ---\n%s\n", i+1, pretty)
	}
	fmt.Printf("统计: 成功=%d 失败=%d 重试=%d 条目=%d 耗时=%s\n",
		result.Stats.Success, result.Stats.Fail, result.Stats.Retry,
		result.Stats.Items, result.Stats.Elapsed.Round(time.Millisecond))
}

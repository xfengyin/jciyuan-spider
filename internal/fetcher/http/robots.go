// Package http 提供基于 net/http 的 Fetcher 实现。
package http

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jciyuan-spider/internal/logger"
)

// robotsRule 记录 robots.txt 中的单条路径规则。
type robotsRule struct {
	path    string
	allowed bool
}

// RobotsChecker 拉取并解析站点 robots.txt，提供路径级合规检查。
type RobotsChecker struct {
	rules []robotsRule
}

// newRobotsChecker 同步拉取并解析 baseURL 对应的 robots.txt。
func newRobotsChecker(client *http.Client, baseURL string, log logger.Logger) (*RobotsChecker, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("解析 baseURL 失败: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("baseURL 缺少 scheme 或 host")
	}

	robotsURL := u.Scheme + "://" + u.Host + "/robots.txt"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建 robots.txt 请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("拉取 robots.txt 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("robots.txt 返回 HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("读取 robots.txt 失败: %w", err)
	}

	return parseRobots(string(body)), nil
}

// parseRobots 解析 robots.txt 文本，仅处理 User-agent: * 或本站爬虫的分组。
func parseRobots(content string) *RobotsChecker {
	checker := &RobotsChecker{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	inMatchingGroup := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(line[:idx]))
		val := strings.TrimSpace(line[idx+1:])

		switch key {
		case "user-agent":
			// 仅匹配通配符或本站爬虫名；出现新分组时重置匹配状态。
			inMatchingGroup = val == "*" || strings.EqualFold(val, "jciyuan-spider")
		case "disallow":
			if inMatchingGroup && val != "" {
				checker.rules = append(checker.rules, robotsRule{path: val, allowed: false})
			}
		case "allow":
			if inMatchingGroup && val != "" {
				checker.rules = append(checker.rules, robotsRule{path: val, allowed: true})
			}
		}
	}
	return checker
}

// IsAllowed 判断目标路径是否允许抓取；checker 为 nil 时默认允许。
func (c *RobotsChecker) IsAllowed(rawURL string) bool {
	if c == nil {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}

	path := u.Path
	if path == "" {
		path = "/"
	}

	// 优先匹配最长路径；Allow 可覆盖 Disallow。
	matched := false
	allowed := true
	longest := 0
	for _, r := range c.rules {
		if strings.HasPrefix(path, r.path) && len(r.path) > longest {
			longest = len(r.path)
			matched = true
			allowed = r.allowed
		}
	}
	if !matched {
		return true
	}
	return allowed
}

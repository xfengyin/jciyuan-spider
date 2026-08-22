package spider

import (
	"context"
	"fmt"
	"regexp"

	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/model"
)

// m3u8Pattern 用于从播放页 HTML 中简单匹配 M3U8 URL。
var m3u8Pattern = regexp.MustCompile(`(?i)(https?://[^\s"']+\.m3u8[^\s"']*)`)

// fetchM3U8 抓取单集播放页并解析 M3U8 URL。
func (s *Spider) fetchM3U8(ctx context.Context, ep *model.Episode) error {
	resp, err := s.fetcher.Fetch(ctx, &fetcher.Request{
		URL:    ep.URL,
		Method: "GET",
	})
	if err != nil {
		return fmt.Errorf("抓取播放页失败: %w", err)
	}
	if resp != nil && resp.Body != nil {
		s.metrics.AddBytes(ctx, int64(len(resp.Body)))
	}

	m3u8 := parseM3U8URL(string(resp.Body))
	if m3u8 == "" {
		return fmt.Errorf("未解析到 M3U8 URL")
	}
	ep.M3U8URL = m3u8
	ep.IsCrawled = true
	return nil
}

// parseM3U8URL 使用简单正则从 HTML 中提取 M3U8 URL。
func parseM3U8URL(html string) string {
	matches := m3u8Pattern.FindStringSubmatch(html)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

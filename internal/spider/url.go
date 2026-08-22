package spider

import (
	"fmt"
	"strings"
)

// buildURL 根据配置模板构建详情页 URL。
func (s *Spider) buildURL(animeID int64) string {
	pattern := s.config.Spider.DetailURLPattern
	if pattern == "" {
		pattern = "{{base_url}}/acgdetail/{{id}}.html"
	}
	pattern = strings.ReplaceAll(pattern, "{{base_url}}", s.config.Spider.BaseURL)
	pattern = strings.ReplaceAll(pattern, "{{id}}", fmt.Sprintf("%d", animeID))
	return pattern
}

package html

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"jciyuan-spider/internal/model"
)

// episodePattern 解析剧集 URL：/acgplay/{animeID}-{group}-{number}.html。
var episodePattern = regexp.MustCompile(`/acgplay/(\d+)-(\d+)-(\d+)\.html`)

// buildEpisodes 从提取出的 URL/路径列表构造 Episode 切片，按 number 排序去重。
func buildEpisodes(value interface{}, baseURL string) ([]*model.Episode, error) {
	paths, ok := value.([]string)
	if !ok {
		return nil, fmt.Errorf("episodes 字段期望 []string，实际为 %T", value)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("解析基础 URL 失败: %w", err)
	}

	seen := make(map[int]bool)
	episodes := make([]*model.Episode, 0, len(paths))

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		ep, ok := parseEpisodePath(path, base)
		if !ok {
			continue
		}
		if seen[ep.Number] {
			continue
		}
		seen[ep.Number] = true
		episodes = append(episodes, ep)
	}

	sort.Slice(episodes, func(i, j int) bool {
		return episodes[i].Number < episodes[j].Number
	})

	return episodes, nil
}

// parseEpisodePath 解析单个剧集路径并返回 Episode。
func parseEpisodePath(path string, base *url.URL) (*model.Episode, bool) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, false
	}
	abs := base.ResolveReference(u).String()

	matches := episodePattern.FindStringSubmatch(abs)
	if len(matches) < 4 {
		return nil, false
	}

	animeID, _ := strconv.ParseInt(matches[1], 10, 64)
	number, _ := strconv.Atoi(matches[3])
	if number <= 0 {
		return nil, false
	}

	return &model.Episode{
		AnimeID:   animeID,
		Number:    number,
		Title:     fmt.Sprintf("第%02d集", number),
		URL:       abs,
		IsCrawled: false,
		CreatedAt: time.Now(),
	}, true
}

package parser

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"jciyuan-spider-v2/internal/model"
)

// HTMLParser HTML 解析器实现
type HTMLParser struct {
	baseURL string
}

// NewHTMLParser 创建 HTML 解析器
func NewHTMLParser(baseURL string) *HTMLParser {
	return &HTMLParser{baseURL: baseURL}
}

// ParseAnimeDetail 解析动漫详情页
func (p *HTMLParser) ParseAnimeDetail(html string) (*model.AnimeInfo, error) {
	anime := &model.AnimeInfo{
		Tags:      []string{},
		Episodes:  []model.Episode{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	p.extractBasicInfo(html, anime)
	anime.Tags = p.extractTags(html)
	anime.Episodes = p.extractEpisodes(html, anime.ID)
	anime.EpisodeNum = len(anime.Episodes)

	return anime, nil
}

// extractBasicInfo 提取基本信息
func (p *HTMLParser) extractBasicInfo(html string, anime *model.AnimeInfo) {
	anime.Title = p.extractTitle(html)
	anime.Year = extractString(`(19|20)\d{2}`, html)
	anime.Region = p.extractRegion(html)
	anime.Description = p.extractDescription(html)
	anime.UpdateDate = extractString(`(\d{4}-\d{2}-\d{2})`, html)
	anime.CoverImage = p.extractCoverImage(html)
	anime.DoubanURL = extractString(`href=["']([^"']*douban[^"']*)["']`, html)
	anime.UpdateNum = extractInt(`更新至(\d+)集`, html)
}

// extractTitle 提取标题
func (p *HTMLParser) extractTitle(html string) string {
	title := extractString(`<title[^>]*>([^<]+)</title>`, html)
	if title != "" {
		title = cleanText(title)
		if idx := strings.Index(title, "_"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
		if idx := strings.Index(title, "-"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
		return title
	}

	title = extractString(`<h1[^>]*>([^<]+)</h1>`, html)
	if title != "" {
		return cleanText(title)
	}
	return "未知标题"
}

// extractRegion 提取地区
func (p *HTMLParser) extractRegion(html string) string {
	regions := []string{"大陆", "日本", "美国", "韩国", "港台", "欧美"}
	for _, region := range regions {
		if strings.Contains(html, region) {
			return region
		}
	}
	return "未知"
}

// extractDescription 提取简介
func (p *HTMLParser) extractDescription(html string) string {
	desc := extractString(`meta\s+name=["']description["']\s+content=["']([^"']+)["']`, html)
	if desc != "" {
		return cleanText(desc)
	}
	desc = extractString(`剧情[:：]([^<]{50,500})`, html)
	if desc != "" {
		return cleanText(desc)
	}
	return ""
}

// extractCoverImage 提取封面图
func (p *HTMLParser) extractCoverImage(html string) string {
	cover := extractString(`src=["']([^"']*doubaocdn[^"']*)["']`, html)
	if cover != "" {
		return cover
	}
	cover = extractString(`data-original=["']([^"']+)["']`, html)
	if cover != "" {
		return cover
	}
	return extractString(`og:image["']\s+content=["']([^"']+)["']`, html)
}

// extractTags 提取标签
func (p *HTMLParser) extractTags(html string) []string {
	tagList := []string{
		"热血", "冒险", "搞笑", "战斗", "奇幻", "科幻",
		"校园", "恋爱", "治愈", "运动", "音乐", "美食",
		"悬疑", "推理", "惊悚", "恐怖", "后宫", "机战",
		"战争", "历史", "励志", "社畜", "爆笑", "国产动漫",
		"日本动漫", "美国动漫", "韩国动漫",
	}

	content := strings.ToLower(html)
	tags := make([]string, 0)
	for _, tag := range tagList {
		if strings.Contains(content, strings.ToLower(tag)) {
			tags = append(tags, tag)
		}
	}
	return uniqueStrings(tags)
}

// extractEpisodes 提取剧集列表
func (p *HTMLParser) extractEpisodes(html string, animeID int64) []model.Episode {
	pattern := regexp.MustCompile(`/acgplay/(\d+)-(\d+)-(\d+)\.html`)
	matches := pattern.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool, len(matches))
	episodes := make([]model.Episode, 0, len(matches))

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		key := match[0]
		if seen[key] {
			continue
		}
		seen[key] = true

		episodeNum := parseInt(match[3])
		title := fmt.Sprintf("第%02d集", episodeNum)
		episodeURL := p.baseURL + match[0]
		isVIP := strings.Contains(html, "vip") || strings.Contains(html, "VIP")

		episodes = append(episodes, model.Episode{
			AnimeID:   animeID,
			Number:    episodeNum,
			Title:     title,
			URL:       episodeURL,
			IsVIP:     isVIP,
			IsCrawled: false,
			CreatedAt: time.Now(),
		})
	}

	// 使用 sort.Slice 替代冒泡排序
	sort.Slice(episodes, func(i, j int) bool {
		return episodes[i].Number < episodes[j].Number
	})

	// 按集数去重
	deduped := make([]model.Episode, 0, len(episodes))
	numSeen := make(map[int]bool, len(episodes))
	for _, ep := range episodes {
		if !numSeen[ep.Number] {
			numSeen[ep.Number] = true
			deduped = append(deduped, ep)
		}
	}

	return deduped
}

// parseInt 安全解析整数
func parseInt(s string) int {
	var result int
	fmt.Sscanf(strings.TrimSpace(s), "%d", &result)
	return result
}

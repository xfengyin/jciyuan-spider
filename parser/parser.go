// Package parser - HTML解析器
package parser

import (
	"regexp"
	"strings"
	"time"

	"jciyuan-spider-v2/model"
	"jciyuan-spider-v2/utils"
)

// ============================================================================
// HTML解析器
// ============================================================================

// Parser HTML解析器
type Parser struct {
	baseURL string
}

// NewParser 创建解析器
func NewParser(baseURL string) *Parser {
	return &Parser{
		baseURL: baseURL,
	}
}

// ParseAnimeDetail 解析动漫详情页
func (p *Parser) ParseAnimeDetail(html string) (*model.AnimeInfo, error) {
	anime := &model.AnimeInfo{
		Tags:      []string{},
		Episodes:  []model.Episode{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// 提取基本信息
	p.extractBasicInfo(html, anime)
	
	// 提取标签
	anime.Tags = p.extractTags(html)
	
	// 提取剧集列表
	anime.Episodes = p.extractEpisodes(html, anime.ID)
	anime.EpisodeNum = len(anime.Episodes)
	
	return anime, nil
}

// extractBasicInfo 提取基本信息
func (p *Parser) extractBasicInfo(html string, anime *model.AnimeInfo) {
	// 提取标题
	anime.Title = p.extractTitle(html)
	
	// 提取年份
	anime.Year = utils.ExtractString(`(19|20)\d{2}`, html)
	
	// 提取地区
	anime.Region = p.extractRegion(html)
	
	// 提取简介
	anime.Description = p.extractDescription(html)
	
	// 提取更新日期
	anime.UpdateDate = p.extractUpdateDate(html)
	
	// 提取封面图
	anime.CoverImage = p.extractCoverImage(html)
	
	// 提取豆瓣链接
	anime.DoubanURL = p.extractDoubanURL(html)
	
	// 提取更新集数
	anime.UpdateNum = utils.ExtractInt(`更新至(\d+)集`, html)
}

// extractTitle 提取标题
func (p *Parser) extractTitle(html string) string {
	// 方案1：从title标签提取
	title := utils.ExtractString(`<title[^>]*>([^<]+)</title>`, html)
	if title != "" {
		// 清理并移除后缀
		title = utils.CleanText(title)
		if idx := strings.Index(title, "_"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
		if idx := strings.Index(title, "-"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
		return title
	}
	
	// 方案2：从h1标签提取
	title = utils.ExtractString(`<h1[^>]*>([^<]+)</h1>`, html)
	if title != "" {
		return utils.CleanText(title)
	}
	
	return "未知标题"
}

// extractRegion 提取地区
func (p *Parser) extractRegion(html string) string {
	regions := []string{"大陆", "日本", "美国", "韩国", "港台", "欧美"}
	for _, region := range regions {
		if strings.Contains(html, region) {
			return region
		}
	}
	return "未知"
}

// extractDescription 提取简介
func (p *Parser) extractDescription(html string) string {
	// 从meta description提取
	desc := utils.ExtractString(`meta\s+name=["']description["']\s+content=["']([^"']+)["']`, html)
	if desc != "" {
		return utils.CleanText(desc)
	}
	
	// 从页面内容提取
	desc = utils.ExtractString(`剧情[:：]([^<]{50,500})`, html)
	if desc != "" {
		return utils.CleanText(desc)
	}
	
	return ""
}

// extractUpdateDate 提取更新日期
func (p *Parser) extractUpdateDate(html string) string {
	// 匹配格式：2026-04-19
	date := utils.ExtractString(`(\d{4}-\d{2}-\d{2})`, html)
	return date
}

// extractCoverImage 提取封面图
func (p *Parser) extractCoverImage(html string) string {
	// 优先提取doubaocdn
	cover := utils.ExtractString(`src=["']([^"']*doubaocdn[^"']*)["']`, html)
	if cover != "" {
		return cover
	}
	
	// 备用：其他图片
	cover = utils.ExtractString(`data-original=["']([^"']+)["']`, html)
	if cover != "" {
		return cover
	}
	
	cover = utils.ExtractString(`og:image["']\s+content=["']([^"']+)["']`, html)
	return cover
}

// extractDoubanURL 提取豆瓣链接
func (p *Parser) extractDoubanURL(html string) string {
	url := utils.ExtractString(`href=["']([^"']*douban[^"']*)["']`, html)
	return url
}

// extractTags 提取标签
func (p *Parser) extractTags(html string) []string {
	tags := []string{}
	
	// 常见动漫标签
	tagList := []string{
		"热血", "冒险", "搞笑", "战斗", "奇幻", "科幻",
		"校园", "恋爱", "治愈", "运动", "音乐", "美食",
		"悬疑", "推理", "惊悚", "恐怖", "后宫", "机战",
		"战争", "历史", "励志", "社畜", "爆笑", "国产动漫",
		"日本动漫", "美国动漫", "韩国动漫", "搞笑", "运动",
	}
	
	content := strings.ToLower(html)
	for _, tag := range tagList {
		if strings.Contains(content, strings.ToLower(tag)) {
			tags = append(tags, tag)
		}
	}
	
	// 去重
	tags = utils.UniqueStrings(tags)
	
	return tags
}

// extractEpisodes 提取剧集列表
func (p *Parser) extractEpisodes(html string, animeID int64) []model.Episode {
	episodes := []model.Episode{}
	seen := make(map[string]bool)
	
	// 提取所有集数链接
	pattern := regexp.MustCompile(`/acgplay/(\d+)-(\d+)-(\d+)\.html`)
	matches := pattern.FindAllStringSubmatch(html, -1)
	
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		
		// 去重键
		key := match[0]
		if seen[key] {
			continue
		}
		seen[key] = true
		
		episodeNum := utils.ParseInt(match[3])
		
		// 构建标题
		title := "第" + formatEpisodeNum(episodeNum) + "集"
		
		// 构建URL
		episodeURL := p.baseURL + match[0]
		
		// 判断是否VIP
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
	
	// 排序去重
	episodes = sortAndDedupeEpisodes(episodes)
	
	return episodes
}

// sortAndDedupeEpisodes 排序去重
func sortAndDedupeEpisodes(episodes []model.Episode) []model.Episode {
	// 排序
	for i := 0; i < len(episodes); i++ {
		for j := i + 1; j < len(episodes); j++ {
			if episodes[i].Number > episodes[j].Number {
				episodes[i], episodes[j] = episodes[j], episodes[i]
			}
		}
	}
	
	// 去重
	seen := make(map[int]bool)
	result := []model.Episode{}
	for _, ep := range episodes {
		if !seen[ep.Number] {
			seen[ep.Number] = true
			result = append(result, ep)
		}
	}
	
	return result
}

// formatEpisodeNum 格式化集数
func formatEpisodeNum(num int) string {
	if num < 10 {
		return "0" + utils.ToString(num)
	}
	return utils.ToString(num)
}

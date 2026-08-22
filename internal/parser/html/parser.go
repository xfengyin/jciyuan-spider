// Package html 提供配置驱动的 HTML Pipeline 解析器。
package html

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/logger"
	"jciyuan-spider/internal/model"
	"jciyuan-spider/internal/parser"
	"jciyuan-spider/internal/parser/extractor"
	"jciyuan-spider/internal/parser/processor"
)

// HTMLParser 配置驱动的 HTML 解析器。
type HTMLParser struct {
	cfg model.HTMLParserConfig
}

// New 创建 HTMLParser 实例，符合 parser.Builder 签名。
func New(cfg model.ParserConfig) (parser.Parser, error) {
	return &HTMLParser{cfg: cfg.HTML}, nil
}

func init() {
	parser.Register("html", New)
}

// Parse 实现 parser.Parser 接口，按配置提取字段并构造 AnimeInfo 与剧集列表。
func (p *HTMLParser) Parse(ctx context.Context, resp *fetcher.Response) (*parser.ParseResult, error) {
	if resp == nil {
		return nil, fmt.Errorf("响应对象不能为空")
	}

	traceID := ""
	if resp.Meta != nil {
		if v, ok := resp.Meta["traceId"].(string); ok {
			traceID = v
		}
	}
	log := logger.GetLogger("parser").WithTraceID(traceID)

	htmlStr, encoding, err := convertEncoding(resp.Body, resp.Headers, p.cfg.Encoding)
	if err != nil {
		log.Warn("编码转换失败，按原内容解析", logger.Err(err))
		htmlStr = string(resp.Body)
		encoding = "unknown"
	}

	doc := &parser.Document{
		URL:      resp.URL,
		HTML:     htmlStr,
		Encoding: encoding,
		Meta:     resp.Meta,
	}

	anime := &model.AnimeInfo{
		Tags:      []string{},
		Episodes:  []model.Episode{},
		DetailURL: resp.URL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	anime.ID = extractAnimeIDFromURL(resp.URL)

	var episodes []*model.Episode

	for _, ec := range p.cfg.Extractors {
		value, warn := p.extractField(ctx, doc, ec, log)
		if warn != "" {
			log.Warn("字段未命中", logger.String("field", ec.Field), logger.String("reason", warn))
			continue
		}
		if value == nil {
			continue
		}

		if ec.Field == "episodes" {
			eps, err := buildEpisodes(value, resp.URL)
			if err != nil {
				log.Warn("剧集解析失败", logger.Err(err))
				continue
			}
			episodes = eps
			continue
		}

		if err := assignField(anime, ec.Field, value); err != nil {
			log.Warn("字段赋值失败", logger.String("field", ec.Field), logger.Err(err))
		}
	}

	if len(episodes) > 0 {
		for _, ep := range episodes {
			anime.Episodes = append(anime.Episodes, *ep)
		}
		anime.EpisodeNum = len(episodes)
	}

	return &parser.ParseResult{
		Anime:    anime,
		Episodes: episodes,
		RawHTML:  resp.Body,
	}, nil
}

// extractField 按 ExtractorConfig 抽取单个字段。
// 返回值 (value, warnReason)；warnReason 非空表示未命中或异常。
func (p *HTMLParser) extractField(ctx context.Context, doc *parser.Document, ec model.ExtractorConfig, log logger.Logger) (interface{}, string) {
	ext, err := extractor.Build(ec.Selector)
	if err != nil {
		return nil, fmt.Sprintf("构造提取器失败: %v", err)
	}

	raw, err := ext.Extract(ctx, doc)
	if err != nil {
		return nil, fmt.Sprintf("提取失败: %v", err)
	}

	values, ok := raw.([]string)
	if !ok {
		return nil, fmt.Sprintf("提取器返回类型异常: %T", raw)
	}

	if len(values) == 0 {
		return nil, "未匹配到任何结果"
	}

	// 根据 multiple 配置决定处理单个值还是切片
	current := interface{}(values)
	if !ec.Multiple {
		current = values[0]
	}

	for _, pc := range ec.Processors {
		proc, err := processor.Build(pc)
		if err != nil {
			log.Warn("构造后处理器失败", logger.String("type", pc.Type), logger.Err(err))
			continue
		}
		current, err = proc.Process(ctx, current)
		if err != nil {
			log.Warn("后处理失败", logger.String("type", pc.Type), logger.Err(err))
			continue
		}
	}

	// 列表字段的去重与排序
	if ec.Multiple {
		if list, ok := current.([]string); ok {
			if ec.Deduplicate {
				list = uniqueStrings(list)
			}
			if ec.SortBy == "text" {
				sort.Strings(list)
			}
			current = list
		}
	}

	return current, ""
}

// assignField 将提取值写入 AnimeInfo 的对应字段。
func assignField(anime *model.AnimeInfo, field string, value interface{}) error {
	switch field {
	case "title":
		anime.Title = toString(value)
	case "year":
		anime.Year = toString(value)
	case "region":
		anime.Region = toString(value)
	case "tags":
		anime.Tags = toStringSlice(value)
	case "cover_image":
		anime.CoverImage = toString(value)
	case "description":
		anime.Description = toString(value)
	case "update_date":
		anime.UpdateDate = toString(value)
	case "episode_num":
		if n, err := strconv.Atoi(toString(value)); err == nil {
			anime.EpisodeNum = n
		}
	case "update_num":
		if n, err := strconv.Atoi(toString(value)); err == nil {
			anime.UpdateNum = n
		}
	case "douban_url":
		anime.DoubanURL = toString(value)
	default:
		return fmt.Errorf("未知字段: %s", field)
	}
	return nil
}

// extractAnimeIDFromURL 从详情页 URL 提取动漫 ID。
func extractAnimeIDFromURL(url string) int64 {
	re := regexp.MustCompile(`/acgdetail/(\d+)\.html`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return 0
	}
	var id int64
	fmt.Sscanf(matches[1], "%d", &id)
	return id
}

// toString 将提取值转换为字符串。
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if ss, ok := v.([]string); ok {
		if len(ss) > 0 {
			return ss[0]
		}
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// toStringSlice 将提取值转换为字符串切片。
func toStringSlice(v interface{}) []string {
	if ss, ok := v.([]string); ok {
		return ss
	}
	if s, ok := v.(string); ok && s != "" {
		return []string{s}
	}
	return nil
}

// uniqueStrings 对字符串切片去重并保持顺序。
func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

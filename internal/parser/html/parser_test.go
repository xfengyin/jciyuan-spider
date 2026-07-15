// Package html 的单元测试。
package html

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/model"
)

// newTestParser 返回配置好的 HTMLParser，用于测试。
func newTestParser() *HTMLParser {
	return &HTMLParser{cfg: model.HTMLParserConfig{
		Encoding: "utf-8",
		Extractors: []model.ExtractorConfig{
			{
				Field: "title",
				Selector: model.SelectorConfig{Type: "css", Value: "h1.title"},
				Processors: []model.ProcessorConfig{
					{Type: "trim"},
				},
			},
			{
				Field: "description",
				Selector: model.SelectorConfig{Type: "css", Value: "p.desc"},
				Processors: []model.ProcessorConfig{
					{Type: "clean_text"},
				},
			},
			{
				Field:    "tags",
				Selector: model.SelectorConfig{Type: "css", Value: "span.tag"},
				Multiple: true,
				Deduplicate: true,
			},
			{
				Field:    "episodes",
				Selector: model.SelectorConfig{Type: "regex", Value: `/acgplay/(\d+)-(\d+)-(\d+)\.html`},
				Multiple: true,
				Deduplicate: true,
			},
		},
	}}
}

// TestHTMLParserParseDetail 验证解析详情页能正确提取字段与剧集。
func TestHTMLParserParseDetail(t *testing.T) {
	p := newTestParser()
	html := `<html>
<head><title>测试动漫_2024_日本</title></head>
<body>
<h1 class="title">  测试动漫  </h1>
<p class="desc"><strong>简介</strong>：  这是一个<b>测试</b>动漫。</p>
<span class="tag">热血</span><span class="tag">冒险</span><span class="tag">热血</span>
<a href="/acgplay/37439-1-2.html">第2集</a>
<a href="/acgplay/37439-1-1.html">第1集</a>
</body>
</html>`

	resp := &fetcher.Response{
		URL:        "https://example.com/acgdetail/37439.html",
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"text/html; charset=utf-8"}},
		Body:       []byte(html),
	}

	result, err := p.Parse(context.Background(), resp)
	require.NoError(t, err)
	require.NotNil(t, result.Anime)

	assert.Equal(t, int64(37439), result.Anime.ID)
	assert.Equal(t, "测试动漫", result.Anime.Title)
	assert.Equal(t, "简介： 这是一个测试动漫。", result.Anime.Description)
	assert.Equal(t, []string{"热血", "冒险"}, result.Anime.Tags)
	assert.Len(t, result.Anime.Episodes, 2)
	assert.Equal(t, 1, result.Anime.Episodes[0].Number)
	assert.Equal(t, 2, result.Anime.Episodes[1].Number)
	assert.Equal(t, 2, result.Anime.EpisodeNum)
}

// TestHTMLParserParseNilResponse 验证空响应返回错误。
func TestHTMLParserParseNilResponse(t *testing.T) {
	p := newTestParser()
	_, err := p.Parse(context.Background(), nil)
	require.Error(t, err)
}

// TestHTMLParserEpisodeFieldParsing 验证 episodes 字段被正确解析为 Episode 列表。
func TestHTMLParserEpisodeFieldParsing(t *testing.T) {
	p := &HTMLParser{cfg: model.HTMLParserConfig{
		Encoding: "utf-8",
		Extractors: []model.ExtractorConfig{
			{
				Field:    "episodes",
				Selector: model.SelectorConfig{Type: "regex", Value: `/acgplay/(\d+)-(\d+)-(\d+)\.html`},
				Multiple: true,
			},
		},
	}}

	html := `<a href="/acgplay/123-0-3.html">3</a><a href="/acgplay/123-0-1.html">1</a>`
	resp := &fetcher.Response{URL: "https://example.com/acgdetail/123.html", Body: []byte(html)}

	result, err := p.Parse(context.Background(), resp)
	require.NoError(t, err)
	require.Len(t, result.Episodes, 2)
	assert.Equal(t, 1, result.Episodes[0].Number)
	assert.Equal(t, 3, result.Episodes[1].Number)
}

package html

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildEpisodesSortsAndDeduplicates 验证剧集列表会按集数排序并去重。
func TestBuildEpisodesSortsAndDeduplicates(t *testing.T) {
	paths := []string{
		"/acgplay/123-1-3.html",
		"/acgplay/123-1-1.html",
		"/acgplay/123-1-2.html",
		"/acgplay/123-1-3.html",
	}

	eps, err := buildEpisodes(paths, "https://example.com/acgdetail/123.html")
	require.NoError(t, err)
	require.Len(t, eps, 3)

	assert.Equal(t, 1, eps[0].Number)
	assert.Equal(t, 2, eps[1].Number)
	assert.Equal(t, 3, eps[2].Number)
}

// TestBuildEpisodesResolvesRelativeURL 验证相对路径会被解析为绝对 URL。
func TestBuildEpisodesResolvesRelativeURL(t *testing.T) {
	paths := []string{"/acgplay/123-1-1.html"}
	eps, err := buildEpisodes(paths, "https://example.com/acgdetail/123.html")
	require.NoError(t, err)
	require.Len(t, eps, 1)
	assert.Equal(t, "https://example.com/acgplay/123-1-1.html", eps[0].URL)
}

// TestParseEpisodePathInvalid 验证非法路径会被忽略。
func TestParseEpisodePathInvalid(t *testing.T) {
	base, _ := url.Parse("https://example.com")
	ep, ok := parseEpisodePath("/not-an-episode.html", base)
	assert.False(t, ok)
	assert.Nil(t, ep)
}

// TestBuildEpisodesIgnoresEmpty 验证空路径会被忽略。
func TestBuildEpisodesIgnoresEmpty(t *testing.T) {
	eps, err := buildEpisodes([]string{"", " ", "/acgplay/123-1-1.html"}, "https://example.com/acgdetail/123.html")
	require.NoError(t, err)
	assert.Len(t, eps, 1)
}

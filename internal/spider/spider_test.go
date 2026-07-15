// Package spider 的单元测试。
package spider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jciyuan-spider-v2/internal/fetcher"
	"jciyuan-spider-v2/internal/mocks"
	"jciyuan-spider-v2/internal/model"
	"jciyuan-spider-v2/internal/parser"
	"jciyuan-spider-v2/internal/resume"
	"jciyuan-spider-v2/internal/worker"
)

// testConfig 返回一个最小可用的 Spider 配置。
func testConfig(t *testing.T) *model.Config {
	return &model.Config{
		Spider: model.SpiderConfig{
			BaseURL:          "https://example.com",
			DetailURLPattern: "{{base_url}}/acgdetail/{{id}}.html",
			Concurrency:      2,
			QueueSize:        10,
		},
		Crawl: model.CrawlConfig{
			AnimeID: 37439,
		},
		Storage: model.StorageConfig{
			Output: model.OutputConfig{
				SaveJSON: true,
			},
			JSON: model.JSONStorageConfig{OutputDir: t.TempDir()},
		},
	}
}

// newTestSpider 使用 Mock 依赖构造 Spider 实例。
func newTestSpider(t *testing.T, cfg *model.Config) (*Spider, *mocks.MockFetcher, *mocks.MockParser, *mocks.MockStorage, *mocks.MockMetrics) {
	mf := mocks.NewMockFetcher()
	mp := mocks.NewMockParser()
	ms := mocks.NewMockStorage()
	mm := mocks.NewMockMetrics()
	log := mocks.NewMockLogger()

	resumeManager := resume.NewManager(ms)
	pool := worker.NewPool(cfg.Spider.Concurrency, cfg.Spider.QueueSize)

	s := New(Options{
		Config:  cfg,
		Fetcher: mf,
		Parser:  mp,
		Storage: ms,
		Resume:  resumeManager,
		Pool:    pool,
		Metrics: mm,
		Logger:  log,
	})
	t.Cleanup(s.Close)
	return s, mf, mp, ms, mm
}

// TestSpiderCrawlDetailSuccess 验证正常详情页抓取、解析、保存流程。
func TestSpiderCrawlDetailSuccess(t *testing.T) {
	cfg := testConfig(t)
	cfg.Crawl.Resume = false

	s, mf, mp, ms, mm := newTestSpider(t, cfg)
	detailURL := "https://example.com/acgdetail/37439.html"

	mf.SetResponse(detailURL, &fetcher.Response{
		URL:        detailURL,
		StatusCode: 200,
		Body:       []byte("<html></html>"),
	}, nil)

	mp.SetResult(detailURL, &parser.ParseResult{
		Anime: &model.AnimeInfo{
			ID:          37439,
			Title:       "测试动漫",
			Episodes:    []model.Episode{{Number: 1, URL: "https://example.com/acgplay/37439-1-1.html"}},
			EpisodeNum:  1,
		},
		RawHTML: []byte("<html></html>"),
	}, nil)

	require.NoError(t, s.Run(context.Background()))
	assert.Equal(t, StateCompleted, s.State())

	exists, err := ms.Exists(context.Background(), 37439)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, int64(1), mm.StorageSaveCount())
}

// TestSpiderResumeSkipCompleted 验证开启断点续爬且非增量时，已完成的动漫会被跳过。
func TestSpiderResumeSkipCompleted(t *testing.T) {
	cfg := testConfig(t)
	cfg.Crawl.Resume = true
	cfg.Crawl.Incremental = false

	s, mf, _, _, mm := newTestSpider(t, cfg)

	// 预先标记为已完成。
	require.NoError(t, s.resume.MarkCompleted(context.Background(), 37439))

	require.NoError(t, s.Run(context.Background()))
	assert.Equal(t, StateCompleted, s.State())
	assert.Equal(t, 0, mf.CallCount("https://example.com/acgdetail/37439.html"))
	assert.Equal(t, int64(0), mm.StorageSaveCount())
}

// TestSpiderIncrementalMergeEpisodes 验证增量模式下会合并旧剧集中的 M3U8 URL。
func TestSpiderIncrementalMergeEpisodes(t *testing.T) {
	cfg := testConfig(t)
	cfg.Crawl.Resume = true
	cfg.Crawl.Incremental = true

	s, mf, mp, ms, mm := newTestSpider(t, cfg)
	detailURL := "https://example.com/acgdetail/37439.html"

	// 旧数据：第1集已抓取到 M3U8。
	oldAnime := &model.AnimeInfo{
		ID:         37439,
		Title:      "旧标题",
		Episodes:   []model.Episode{{Number: 1, URL: "https://example.com/acgplay/37439-1-1.html", M3U8URL: "https://example.com/1.m3u8", IsCrawled: true}},
		EpisodeNum: 1,
	}
	require.NoError(t, ms.Save(context.Background(), oldAnime))

	mf.SetResponse(detailURL, &fetcher.Response{
		URL:        detailURL,
		StatusCode: 200,
		Body:       []byte("<html></html>"),
	}, nil)

	// 新数据：第1集与第2集，但都没有 M3U8。
	mp.SetResult(detailURL, &parser.ParseResult{
		Anime: &model.AnimeInfo{
			ID:         37439,
			Title:      "新标题",
			Episodes: []model.Episode{
				{Number: 1, URL: "https://example.com/acgplay/37439-1-1.html"},
				{Number: 2, URL: "https://example.com/acgplay/37439-1-2.html"},
			},
			EpisodeNum: 2,
		},
		RawHTML: []byte("<html></html>"),
	}, nil)

	require.NoError(t, s.Run(context.Background()))
	assert.Equal(t, StateCompleted, s.State())
	assert.Equal(t, int64(1), mm.StorageSaveCount())

	loaded, err := ms.Load(context.Background(), 37439)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "新标题", loaded.Title)
	require.Len(t, loaded.Episodes, 2)
	assert.Equal(t, "https://example.com/1.m3u8", loaded.Episodes[0].M3U8URL)
	assert.True(t, loaded.Episodes[0].IsCrawled)
}

// TestSpiderContextCancel 验证上下文取消后 Run 返回错误且不会保存数据。
func TestSpiderContextCancel(t *testing.T) {
	cfg := testConfig(t)
	cfg.Crawl.Resume = false

	s, mf, mp, ms, _ := newTestSpider(t, cfg)
	detailURL := "https://example.com/acgdetail/37439.html"

	ctx, cancel := context.WithCancel(context.Background())

	mf.SetResponse(detailURL, &fetcher.Response{
		URL:        detailURL,
		StatusCode: 200,
		Body:       []byte("<html></html>"),
	}, nil)

	// 解析器内部取消上下文，模拟运行中被取消。
	mp.SetResult(detailURL, nil, context.Canceled)

	cancel()
	err := s.Run(ctx)
	require.Error(t, err)

	exists, err := ms.Exists(context.Background(), 37439)
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestSpiderSaveFailure 验证存储失败时返回错误并记录失败指标。
func TestSpiderSaveFailure(t *testing.T) {
	cfg := testConfig(t)
	cfg.Crawl.Resume = false

	s, mf, mp, ms, mm := newTestSpider(t, cfg)
	detailURL := "https://example.com/acgdetail/37439.html"

	mf.SetResponse(detailURL, &fetcher.Response{
		URL:        detailURL,
		StatusCode: 200,
		Body:       []byte("<html></html>"),
	}, nil)

	mp.SetResult(detailURL, &parser.ParseResult{
		Anime: &model.AnimeInfo{
			ID:         37439,
			Title:      "测试动漫",
			EpisodeNum: 0,
		},
		RawHTML: []byte("<html></html>"),
	}, nil)

	// 让 Save 返回错误。
	ms.SetSaveError(assert.AnError)

	err := s.Run(context.Background())
	require.Error(t, err)
	assert.Equal(t, StateFailed, s.State())
	assert.Equal(t, int64(1), mm.GetStats().StorageSaveFail)
}

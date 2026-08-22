package spider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/logger"
	"jciyuan-spider/internal/model"
	"jciyuan-spider/internal/worker"
)

// crawlDetails 并发调度详情页任务并收集结果。
func (s *Spider) crawlDetails(ctx context.Context, ids []int64) map[int64]*detailResult {
	results := make(map[int64]*detailResult, len(ids))
	var mu sync.Mutex
	var wg sync.WaitGroup
	log := s.traceLog(ctx)

	for _, id := range ids {
		wg.Add(1)
		task := s.newDetailTask(id, &wg, results, &mu)
		if err := s.pool.Submit(ctx, task); err != nil {
			wg.Done()
			mu.Lock()
			results[id] = &detailResult{err: fmt.Errorf("提交任务失败: %w", err)}
			mu.Unlock()
			log.Error("提交详情任务失败", logger.Int64("anime_id", id), logger.Err(err))
		}
	}

	if err := waitWithContext(ctx, &wg); err != nil {
		log.Warn("等待详情任务时被取消", logger.Err(err))
	}
	return results
}

// newDetailTask 构造详情页任务。
// panic 恢复仅在此处处理：任务闭包需要把 panic 转为该动漫的错误结果，
// WorkerPool 的全局 panic 处理器只负责记日志，无法回写结果。
func (s *Spider) newDetailTask(
	animeID int64,
	wg *sync.WaitGroup,
	results map[int64]*detailResult,
	mu *sync.Mutex,
) worker.TaskFunc {
	return func(ctx context.Context) error {
		defer wg.Done()
		log := s.traceLog(ctx)
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("任务 panic: %v", r)
				log.Error("详情任务 panic", logger.Int64("anime_id", animeID), logger.Any("panic", r))
				mu.Lock()
				results[animeID] = &detailResult{err: err}
				mu.Unlock()
			}
		}()

		anime, raw, err := s.crawlDetailPage(ctx, animeID)
		mu.Lock()
		results[animeID] = &detailResult{anime: anime, rawHTML: raw, err: err}
		mu.Unlock()
		return err
	}
}

// crawlDetailPage 抓取并解析单个动漫详情页。
func (s *Spider) crawlDetailPage(ctx context.Context, animeID int64) (*model.AnimeInfo, []byte, error) {
	url := s.buildURL(animeID)
	log := s.traceLog(ctx)
	log.Info("开始抓取详情页", logger.Int64("anime_id", animeID), logger.String("url", url))

	resp, err := s.fetcher.Fetch(ctx, &fetcher.Request{
		URL:    url,
		Method: "GET",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("抓取详情页失败: %w", err)
	}
	if resp != nil && resp.Body != nil {
		s.metrics.AddBytes(ctx, int64(len(resp.Body)))
	}

	result, err := s.parser.Parse(ctx, resp)
	if err != nil {
		s.metrics.IncrParseFail(ctx)
		return nil, resp.Body, fmt.Errorf("解析详情页失败: %w", err)
	}

	anime := result.Anime
	if anime == nil {
		return nil, resp.Body, fmt.Errorf("解析结果为空")
	}

	anime.ID = animeID
	anime.DetailURL = url
	if anime.EpisodeNum == 0 {
		anime.EpisodeNum = len(anime.Episodes)
	}

	s.metrics.IncrParse(ctx)
	log.Info("详情页解析成功",
		logger.Int64("anime_id", animeID),
		logger.String("title", anime.Title),
		logger.Int("episodes", anime.EpisodeNum),
	)
	return anime, result.RawHTML, nil
}

// crawlEpisodes 并发抓取单集播放页并解析 M3U8 URL。
// 任务内的 panic 由 WorkerPool 的全局 panic 处理器统一记录。
func (s *Spider) crawlEpisodes(ctx context.Context, anime *model.AnimeInfo) error {
	eps := anime.Episodes
	if max := s.config.Crawl.MaxEpisodes; max > 0 && len(eps) > max {
		eps = eps[:max]
	}

	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex
	log := s.traceLog(ctx)

	for i := range eps {
		wg.Add(1)
		ep := &eps[i]
		task := worker.TaskFunc(func(taskCtx context.Context) error {
			defer wg.Done()
			taskLog := s.traceLog(taskCtx)
			if err := s.fetchM3U8(taskCtx, ep); err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				taskLog.Warn("M3U8 抓取失败",
					logger.Int64("anime_id", anime.ID),
					logger.Int("episode", ep.Number),
					logger.Err(err),
				)
			}
			return nil
		})

		if err := s.pool.Submit(ctx, task); err != nil {
			wg.Done()
			errMu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			errMu.Unlock()
			log.Error("提交 M3U8 任务失败", logger.Int("episode", ep.Number), logger.Err(err))
		}
	}

	_ = waitWithContext(ctx, &wg)
	return firstErr
}

// saveRawHTML 将原始 HTML 保存到输出目录，用于失败审计。
func (s *Spider) saveRawHTML(log logger.Logger, animeID int64, raw []byte) error {
	dir := s.config.Storage.JSON.OutputDir
	if dir == "" {
		dir = "./output"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	filename := filepath.Join(dir, fmt.Sprintf("%d.raw.html", animeID))
	if err := os.WriteFile(filename, raw, 0644); err != nil {
		return fmt.Errorf("写入原始 HTML 失败: %w", err)
	}
	log.Info("已保存原始 HTML", logger.Int64("anime_id", animeID), logger.String("file", filename))
	return nil
}

// waitWithContext 等待 WaitGroup 完成，或上下文取消时提前返回。
func waitWithContext(ctx context.Context, wg *sync.WaitGroup) error {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

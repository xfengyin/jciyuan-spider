// Package spider 是爬虫核心编排层，负责任务调度、WorkerPool 协调、状态机管理与结果保存。
package spider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"jciyuan-spider/internal/fetcher"
	"jciyuan-spider/internal/logger"
	"jciyuan-spider/internal/metrics"
	"jciyuan-spider/internal/model"
	"jciyuan-spider/internal/parser"
	"jciyuan-spider/internal/resume"
	"jciyuan-spider/internal/storage"
	"jciyuan-spider/internal/worker"
)

// TaskType 定义 WorkerPool 中的任务类型。
type TaskType string

const (
	// TaskTypeDetail 详情页爬取任务。
	TaskTypeDetail TaskType = "detail"
	// TaskTypeEpisode 单集播放页 M3U8 抓取任务。
	TaskTypeEpisode TaskType = "episode"
)

// State 定义爬虫状态机的可能状态。
type State string

const (
	// StateIdle 空闲态。
	StateIdle State = "idle"
	// StateRunning 运行中。
	StateRunning State = "running"
	// StateCompleted 已完成。
	StateCompleted State = "completed"
	// StateFailed 失败。
	StateFailed State = "failed"
	// StatePaused 暂停（通常由信号中断触发）。
	StatePaused State = "paused"
)

// m3u8Pattern 用于从播放页 HTML 中简单匹配 M3U8 URL。
var m3u8Pattern = regexp.MustCompile(`(?i)(https?://[^\s"']+\.m3u8[^\s"']*)`)

// Options 是创建 Spider 所需的依赖选项，所有依赖均通过接口注入，避免硬编码具体实现。
type Options struct {
	Config  *model.Config
	Fetcher fetcher.Fetcher
	Parser  parser.Parser
	Storage storage.Storage
	Resume  *resume.Manager
	Pool    *worker.Pool
	Metrics metrics.Metrics
	Logger  logger.Logger
}

// Spider 爬虫实例，编排各子模块协同工作。
type Spider struct {
	config  *model.Config
	fetcher fetcher.Fetcher
	parser  parser.Parser
	storage storage.Storage
	metrics metrics.Metrics
	resume  *resume.Manager
	pool    *worker.Pool
	log     logger.Logger
	state   State
	stateMu sync.RWMutex
}

// detailResult 保存单个动漫详情任务的执行结果。
type detailResult struct {
	anime   *model.AnimeInfo
	rawHTML []byte
	err     error
}

// New 使用依赖选项创建爬虫实例。
func New(opts Options) *Spider {
	if opts.Logger == nil {
		opts.Logger = logger.GetLogger("spider")
	}
	if opts.Metrics == nil {
		opts.Metrics = metrics.NewMemoryMetrics()
	}
	if opts.Pool != nil {
		opts.Pool.SetPanicHandler(func(r interface{}) {
			opts.Logger.Error("worker 任务 panic", logger.Any("panic", r))
		})
	}
	return &Spider{
		config:  opts.Config,
		fetcher: opts.Fetcher,
		parser:  opts.Parser,
		storage: opts.Storage,
		metrics: opts.Metrics,
		resume:  opts.Resume,
		pool:    opts.Pool,
		log:     opts.Logger,
		state:   StateIdle,
	}
}

// Run 执行爬取任务，按配置调度详情页任务与可选的 M3U8 任务。
func (s *Spider) Run(ctx context.Context) error {
	s.setState(StateRunning)
	defer s.finalizeState(ctx)

	// 当前版本仅支持单动漫 ID，未来可扩展为 ID 列表。
	ids := []int64{s.config.Crawl.AnimeID}
	idsToCrawl := make([]int64, 0, len(ids))

	log := s.traceLog(ctx)
	for _, id := range ids {
		if s.shouldSkip(ctx, id) {
			log.Info("动漫已标记完成，跳过", logger.Int64("anime_id", id))
			continue
		}
		if s.config.Crawl.Resume {
			if err := s.resume.MarkRunning(ctx, id); err != nil {
				log.Warn("标记运行状态失败", logger.Int64("anime_id", id), logger.Err(err))
			}
		}
		idsToCrawl = append(idsToCrawl, id)
	}

	if len(idsToCrawl) == 0 {
		log.Info("没有需要爬取的动漫")
		s.setState(StateCompleted)
		return nil
	}

	if s.pool == nil {
		return fmt.Errorf("worker pool 未初始化")
	}
	s.pool.Start(ctx)

	if s.metrics != nil {
		go s.reportQueueSize(ctx)
	}

	results := s.crawlDetails(ctx, idsToCrawl)
	return s.processResults(ctx, idsToCrawl, results)
}

// Close 释放爬虫持有的资源。
func (s *Spider) Close() {
	if s.pool != nil {
		_ = s.pool.Stop()
	}
	if s.fetcher != nil {
		_ = s.fetcher.Close()
	}
	if s.storage != nil {
		_ = s.storage.Close()
	}
}

// GetStats 获取统计快照。
func (s *Spider) GetStats() model.Stats {
	return s.metrics.GetStats()
}

// Metrics 返回爬虫持有的 Metrics 实例，便于上层启动健康/指标服务。
func (s *Spider) Metrics() metrics.Metrics {
	return s.metrics
}

// traceLog 返回携带 ctx 中 traceId 的 Logger，便于关键路径打印全链路日志。
func (s *Spider) traceLog(ctx context.Context) logger.Logger {
	return s.log.WithTrace(ctx)
}

// reportQueueSize 周期上报 Worker 队列长度，直到 ctx 取消。
func (s *Spider) reportQueueSize(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.metrics.SetQueueSize(ctx, s.pool.QueueSize())
		}
	}
}

// State 返回当前状态机状态。
func (s *Spider) State() State {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

// shouldSkip 判断动漫是否需要跳过：已完成且非增量模式则跳过。
func (s *Spider) shouldSkip(ctx context.Context, animeID int64) bool {
	if !s.config.Crawl.Resume {
		return false
	}
	if s.config.Crawl.Incremental {
		return false
	}
	return s.resume.IsCompleted(ctx, animeID)
}

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

// processResults 处理详情页结果：增量合并、M3U8 抓取、保存、状态标记与摘要输出。
func (s *Spider) processResults(
	ctx context.Context,
	ids []int64,
	results map[int64]*detailResult,
) error {
	var errs []error
	log := s.traceLog(ctx)

	for _, id := range ids {
		res, ok := results[id]
		if !ok || res == nil {
			err := fmt.Errorf("动漫 %d 无执行结果", id)
			errs = append(errs, err)
			continue
		}
		if res.err != nil {
			err := fmt.Errorf("动漫 %d 爬取失败: %w", id, res.err)
			log.Error(err.Error())
			if s.config.Crawl.Resume {
				_ = s.resume.MarkFailed(ctx, id, res.err.Error())
			}
			errs = append(errs, err)
			continue
		}

		anime := res.anime
		if s.config.Crawl.Incremental {
			merged, err := s.mergeWithOld(ctx, anime)
			if err != nil {
				log.Warn("增量合并失败，使用新数据", logger.Int64("anime_id", id), logger.Err(err))
			} else {
				anime = merged
			}
		}

		if s.config.Storage.Output.SaveM3U8 && len(anime.Episodes) > 0 {
			if err := s.crawlEpisodes(ctx, anime); err != nil {
				log.Warn("M3U8 抓取部分失败", logger.Int64("anime_id", id), logger.Err(err))
			}
		}

		if err := s.storage.Save(ctx, anime); err != nil {
			log.Error("保存动漫信息失败", logger.Int64("anime_id", id), logger.Err(err))
			s.metrics.IncrStorageSaveFail(ctx)
			if s.config.Storage.Output.SaveRawHTML && len(res.rawHTML) > 0 {
				_ = s.saveRawHTML(log, id, res.rawHTML)
			}
			if s.config.Crawl.Resume {
				_ = s.resume.MarkFailed(ctx, id, err.Error())
			}
			errs = append(errs, fmt.Errorf("保存动漫 %d 失败: %w", id, err))
			continue
		}

		s.metrics.IncrStorageSave(ctx)
		if s.config.Crawl.Resume {
			_ = s.resume.MarkCompleted(ctx, id)
		}
		s.showSummary(log, anime)
	}

	if len(errs) > 0 {
		s.setState(StateFailed)
		return fmt.Errorf("本次运行存在失败任务: %v", errs)
	}
	s.setState(StateCompleted)
	return nil
}

// mergeWithOld 加载旧数据并与新解析数据合并，实现增量更新。
func (s *Spider) mergeWithOld(ctx context.Context, anime *model.AnimeInfo) (*model.AnimeInfo, error) {
	old, err := s.storage.Load(ctx, anime.ID)
	if err != nil {
		return nil, err
	}
	if old == nil {
		return anime, nil
	}
	return s.mergeAnime(old, anime), nil
}

// mergeAnime 合并旧动漫数据与新动漫数据，保留已有的 M3U8 等附加信息。
func (s *Spider) mergeAnime(old, newest *model.AnimeInfo) *model.AnimeInfo {
	merged := *newest
	merged.Episodes = mergeEpisodes(old.Episodes, newest.Episodes)
	merged.EpisodeNum = len(merged.Episodes)
	return &merged
}

// mergeEpisodes 按集数合并剧集列表，旧剧集中已抓取的 M3U8 会被保留。
func mergeEpisodes(old, new []model.Episode) []model.Episode {
	existing := make(map[int]model.Episode, len(old))
	for _, ep := range old {
		existing[ep.Number] = ep
	}

	for _, ep := range new {
		if oldEp, ok := existing[ep.Number]; ok && ep.M3U8URL == "" {
			ep.M3U8URL = oldEp.M3U8URL
			ep.IsCrawled = oldEp.IsCrawled
		}
		existing[ep.Number] = ep
	}

	numbers := make([]int, 0, len(existing))
	for n := range existing {
		numbers = append(numbers, n)
	}
	sort.Ints(numbers)

	merged := make([]model.Episode, 0, len(numbers))
	for _, n := range numbers {
		merged = append(merged, existing[n])
	}
	return merged
}

// crawlEpisodes 并发抓取单集播放页并解析 M3U8 URL。
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
			defer func() {
				if r := recover(); r != nil {
					taskLog.Error("M3U8 任务 panic",
						logger.Int64("anime_id", anime.ID),
						logger.Int("episode", ep.Number),
						logger.Any("panic", r),
					)
				}
			}()
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

// showSummary 显示爬取摘要。
func (s *Spider) showSummary(log logger.Logger, anime *model.AnimeInfo) {
	log.Info("爬取摘要",
		logger.String("title", anime.Title),
		logger.String("year", anime.Year),
		logger.String("region", anime.Region),
		logger.Strings("tags", anime.Tags),
		logger.Int("episode_num", anime.EpisodeNum),
	)

	if len(anime.Episodes) > 0 {
		for i := 0; i < 5 && i < len(anime.Episodes); i++ {
			ep := anime.Episodes[i]
			vipTag := ""
			if ep.IsVIP {
				vipTag = " [VIP]"
			}
			log.Info("剧集",
				logger.Int("number", ep.Number),
				logger.String("title", ep.Title),
				logger.String("m3u8_url", ep.M3U8URL),
				logger.String("vip", vipTag),
			)
		}
		if anime.EpisodeNum > 5 {
			log.Info("更多剧集", logger.Int("total", anime.EpisodeNum))
		}
	}
}

// setState 原子设置状态机状态。
func (s *Spider) setState(state State) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.state = state
}

// finalizeState 在 Run 退出时根据上下文与当前状态确定最终状态。
func (s *Spider) finalizeState(ctx context.Context) {
	if r := recover(); r != nil {
		s.log.Error("Run 发生 panic", logger.Any("panic", r))
		s.setState(StateFailed)
	}

	state := s.State()
	if state != StateRunning {
		return
	}
	if ctx.Err() != nil {
		s.setState(StatePaused)
		return
	}
	s.setState(StateFailed)
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

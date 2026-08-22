// Package spider 是爬虫核心编排层，负责任务调度、WorkerPool 协调、状态机管理与结果保存。
package spider

import (
	"context"
	"fmt"
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

// State 返回当前状态机状态。
func (s *Spider) State() State {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
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

// Package sqlitestorage 提供基于 SQLite 的 Storage/StatusStorage 实现，支持 Upsert 与事务批量保存。
package sqlitestorage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"jciyuan-spider-v2/internal/model"
	"jciyuan-spider-v2/internal/storage"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage SQLite 存储实现。
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage 通过 DSN 创建 SQLite 存储实例。
func NewSQLiteStorage(dsn string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &SQLiteStorage{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("初始化表结构失败: %w", err)
	}
	return s, nil
}

// NewSQLiteStorageFromConfig 根据 StorageConfig 构造 Storage（SPI 构造器签名）。
func NewSQLiteStorageFromConfig(cfg model.StorageConfig) (storage.Storage, error) {
	return NewSQLiteStorage(cfg.SQLite.DSN)
}

func init() {
	// SPI 注册 SQLite Storage。
	storage.Register("sqlite", NewSQLiteStorageFromConfig)
}

// migrate 初始化数据库表结构。
func (s *SQLiteStorage) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS anime (
    id INTEGER PRIMARY KEY,
    title TEXT,
    year TEXT,
    region TEXT,
    tags TEXT,
    cover_image TEXT,
    description TEXT,
    update_date TEXT,
    episode_num INTEGER,
    update_num INTEGER,
    douban_url TEXT,
    detail_url TEXT,
    status INTEGER,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS episode (
    anime_id INTEGER,
    number INTEGER,
    title TEXT,
    url TEXT,
    m3u8_url TEXT,
    is_vip BOOLEAN,
    is_crawled BOOLEAN,
    created_at DATETIME,
    PRIMARY KEY (anime_id, number)
);

CREATE TABLE IF NOT EXISTS crawl_status (
    anime_id INTEGER PRIMARY KEY,
    status TEXT,
    current_index INTEGER,
    total_count INTEGER,
    success_count INTEGER,
    fail_count INTEGER,
    retry_count INTEGER,
    error_msg TEXT,
    last_crawl_at DATETIME
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("执行建表语句失败: %w", err)
	}
	return nil
}

// Save 保存动漫信息，使用 INSERT OR REPLACE 实现 Upsert。
func (s *SQLiteStorage) Save(ctx context.Context, anime *model.AnimeInfo) error {
	now := time.Now()
	if anime.CreatedAt.IsZero() {
		anime.CreatedAt = now
	}
	anime.UpdatedAt = now

	tagsJSON, err := json.Marshal(anime.Tags)
	if err != nil {
		return fmt.Errorf("序列化 tags 失败: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO anime
		(id, title, year, region, tags, cover_image, description, update_date,
		 episode_num, update_num, douban_url, detail_url, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		anime.ID, anime.Title, anime.Year, anime.Region, string(tagsJSON),
		anime.CoverImage, anime.Description, anime.UpdateDate, anime.EpisodeNum,
		anime.UpdateNum, anime.DoubanURL, anime.DetailURL, anime.Status,
		anime.CreatedAt, anime.UpdatedAt); err != nil {
		return fmt.Errorf("保存 anime 失败: %w", err)
	}

	if err := s.saveEpisodesTx(ctx, tx, anime.ID, anime.Episodes); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

// Load 加载动漫信息及其分集。
func (s *SQLiteStorage) Load(ctx context.Context, animeID int64) (*model.AnimeInfo, error) {
	var anime model.AnimeInfo
	var tagsJSON string

	row := s.db.QueryRowContext(ctx, `
		SELECT id, title, year, region, tags, cover_image, description, update_date,
		       episode_num, update_num, douban_url, detail_url, status, created_at, updated_at
		FROM anime WHERE id = ?`, animeID)
	err := row.Scan(&anime.ID, &anime.Title, &anime.Year, &anime.Region, &tagsJSON,
		&anime.CoverImage, &anime.Description, &anime.UpdateDate, &anime.EpisodeNum,
		&anime.UpdateNum, &anime.DoubanURL, &anime.DetailURL, &anime.Status,
		&anime.CreatedAt, &anime.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询 anime 失败: %w", err)
	}

	if err := json.Unmarshal([]byte(tagsJSON), &anime.Tags); err != nil {
		return nil, fmt.Errorf("解析 tags 失败: %w", err)
	}

	episodes, err := s.loadEpisodes(ctx, animeID)
	if err != nil {
		return nil, err
	}
	anime.Episodes = episodes
	return &anime, nil
}

// Exists 检查动漫信息是否存在。
func (s *SQLiteStorage) Exists(ctx context.Context, animeID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM anime WHERE id = ?)`, animeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("检查 anime 存在性失败: %w", err)
	}
	return exists, nil
}

// SaveBatch 批量保存动漫信息，使用事务保证原子性。
func (s *SQLiteStorage) SaveBatch(ctx context.Context, animes []*model.AnimeInfo) error {
	if len(animes) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, anime := range animes {
		if err := s.saveAnimeTx(ctx, tx, anime); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

// SaveStatus 保存爬取状态，使用 INSERT OR REPLACE 实现 Upsert。
func (s *SQLiteStorage) SaveStatus(ctx context.Context, status *model.CrawlStatus) error {
	if status.LastCrawlAt.IsZero() {
		status.LastCrawlAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO crawl_status
		(anime_id, status, current_index, total_count, success_count, fail_count, retry_count, error_msg, last_crawl_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		status.AnimeID, status.Status, status.CurrentIndex, status.TotalCount,
		status.SuccessCount, status.FailCount, status.RetryCount, status.ErrorMsg,
		status.LastCrawlAt)
	if err != nil {
		return fmt.Errorf("保存爬取状态失败: %w", err)
	}
	return nil
}

// LoadStatus 加载爬取状态。
func (s *SQLiteStorage) LoadStatus(ctx context.Context, animeID int64) (*model.CrawlStatus, error) {
	var status model.CrawlStatus
	row := s.db.QueryRowContext(ctx, `
		SELECT anime_id, status, current_index, total_count, success_count,
		       fail_count, retry_count, error_msg, last_crawl_at
		FROM crawl_status WHERE anime_id = ?`, animeID)
	err := row.Scan(&status.AnimeID, &status.Status, &status.CurrentIndex, &status.TotalCount,
		&status.SuccessCount, &status.FailCount, &status.RetryCount, &status.ErrorMsg,
		&status.LastCrawlAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("加载爬取状态失败: %w", err)
	}
	return &status, nil
}

// Close 关闭数据库连接。
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// saveAnimeTx 在事务内保存 anime 主记录。
func (s *SQLiteStorage) saveAnimeTx(ctx context.Context, tx *sql.Tx, anime *model.AnimeInfo) error {
	now := time.Now()
	if anime.CreatedAt.IsZero() {
		anime.CreatedAt = now
	}
	anime.UpdatedAt = now

	tagsJSON, err := json.Marshal(anime.Tags)
	if err != nil {
		return fmt.Errorf("序列化 tags 失败: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO anime
		(id, title, year, region, tags, cover_image, description, update_date,
		 episode_num, update_num, douban_url, detail_url, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		anime.ID, anime.Title, anime.Year, anime.Region, string(tagsJSON),
		anime.CoverImage, anime.Description, anime.UpdateDate, anime.EpisodeNum,
		anime.UpdateNum, anime.DoubanURL, anime.DetailURL, anime.Status,
		anime.CreatedAt, anime.UpdatedAt); err != nil {
		return fmt.Errorf("保存 anime %d 失败: %w", anime.ID, err)
	}

	if err := s.saveEpisodesTx(ctx, tx, anime.ID, anime.Episodes); err != nil {
		return err
	}
	return nil
}

// saveEpisodesTx 在事务内保存分集信息，先删除后插入保证幂等。
func (s *SQLiteStorage) saveEpisodesTx(ctx context.Context, tx *sql.Tx, animeID int64, episodes []model.Episode) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM episode WHERE anime_id = ?`, animeID); err != nil {
		return fmt.Errorf("清理旧分集失败: %w", err)
	}
	for _, ep := range episodes {
		ep.AnimeID = animeID
		if ep.CreatedAt.IsZero() {
			ep.CreatedAt = time.Now()
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO episode (anime_id, number, title, url, m3u8_url, is_vip, is_crawled, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			ep.AnimeID, ep.Number, ep.Title, ep.URL, ep.M3U8URL, ep.IsVIP, ep.IsCrawled, ep.CreatedAt); err != nil {
			return fmt.Errorf("保存分集 %d 失败: %w", ep.Number, err)
		}
	}
	return nil
}

// loadEpisodes 加载指定动漫的分集列表。
func (s *SQLiteStorage) loadEpisodes(ctx context.Context, animeID int64) ([]model.Episode, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT anime_id, number, title, url, m3u8_url, is_vip, is_crawled, created_at
		FROM episode WHERE anime_id = ? ORDER BY number`, animeID)
	if err != nil {
		return nil, fmt.Errorf("查询分集失败: %w", err)
	}
	defer rows.Close()

	var episodes []model.Episode
	for rows.Next() {
		var ep model.Episode
		if err := rows.Scan(&ep.AnimeID, &ep.Number, &ep.Title, &ep.URL, &ep.M3U8URL,
			&ep.IsVIP, &ep.IsCrawled, &ep.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描分集失败: %w", err)
		}
		episodes = append(episodes, ep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分集失败: %w", err)
	}
	return episodes, nil
}

// Package mysqlstorage 提供基于 MySQL 的 Storage/StatusStorage 实现，支持 Upsert 与事务批量保存。
package mysqlstorage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"jciyuan-spider/internal/model"
	"jciyuan-spider/internal/storage"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLStorage MySQL 存储实现。
type MySQLStorage struct {
	db *sql.DB
}

// NewMySQLStorage 通过 DSN 创建 MySQL 存储实例。
func NewMySQLStorage(dsn string) (*MySQLStorage, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 MySQL 失败: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)

	s := &MySQLStorage{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("初始化表结构失败: %w", err)
	}
	return s, nil
}

// NewMySQLStorageFromConfig 根据 StorageConfig 构造 Storage（SPI 构造器签名）。
func NewMySQLStorageFromConfig(cfg model.StorageConfig) (storage.Storage, error) {
	return NewMySQLStorage(cfg.MySQL.DSN)
}

func init() {
	// SPI 注册 MySQL Storage。
	storage.Register("mysql", NewMySQLStorageFromConfig)
}

// migrate 初始化数据库表结构。
func (s *MySQLStorage) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS anime (
    id BIGINT PRIMARY KEY,
    title VARCHAR(512),
    year VARCHAR(16),
    region VARCHAR(128),
    tags JSON,
    cover_image VARCHAR(1024),
    description TEXT,
    update_date VARCHAR(64),
    episode_num INT,
    update_num INT,
    douban_url VARCHAR(1024),
    detail_url VARCHAR(1024),
    status INT,
    created_at DATETIME,
    updated_at DATETIME
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS episode (
    anime_id BIGINT,
    number INT,
    title VARCHAR(512),
    url VARCHAR(1024),
    m3u8_url VARCHAR(1024),
    is_vip BOOLEAN,
    is_crawled BOOLEAN,
    created_at DATETIME,
    PRIMARY KEY (anime_id, number)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS crawl_status (
    anime_id BIGINT PRIMARY KEY,
    status VARCHAR(32),
    current_index INT,
    total_count INT,
    success_count INT,
    fail_count INT,
    retry_count INT,
    error_msg TEXT,
    last_crawl_at DATETIME
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("执行建表语句失败: %w", err)
	}
	return nil
}

// Save 保存动漫信息，使用 INSERT ... ON DUPLICATE KEY UPDATE 实现 Upsert。
func (s *MySQLStorage) Save(ctx context.Context, anime *model.AnimeInfo) error {
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
		INSERT INTO anime
		(id, title, year, region, tags, cover_image, description, update_date,
		 episode_num, update_num, douban_url, detail_url, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		title = VALUES(title), year = VALUES(year), region = VALUES(region),
		tags = VALUES(tags), cover_image = VALUES(cover_image), description = VALUES(description),
		update_date = VALUES(update_date), episode_num = VALUES(episode_num),
		update_num = VALUES(update_num), douban_url = VALUES(douban_url),
		detail_url = VALUES(detail_url), status = VALUES(status),
		created_at = VALUES(created_at), updated_at = VALUES(updated_at)`,
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
func (s *MySQLStorage) Load(ctx context.Context, animeID int64) (*model.AnimeInfo, error) {
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
func (s *MySQLStorage) Exists(ctx context.Context, animeID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM anime WHERE id = ?)`, animeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("检查 anime 存在性失败: %w", err)
	}
	return exists, nil
}

// SaveBatch 批量保存动漫信息，使用事务保证原子性。
func (s *MySQLStorage) SaveBatch(ctx context.Context, animes []*model.AnimeInfo) error {
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

// SaveStatus 保存爬取状态，使用 INSERT ... ON DUPLICATE KEY UPDATE 实现 Upsert。
func (s *MySQLStorage) SaveStatus(ctx context.Context, status *model.CrawlStatus) error {
	if status.LastCrawlAt.IsZero() {
		status.LastCrawlAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO crawl_status
		(anime_id, status, current_index, total_count, success_count, fail_count, retry_count, error_msg, last_crawl_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		status = VALUES(status), current_index = VALUES(current_index),
		total_count = VALUES(total_count), success_count = VALUES(success_count),
		fail_count = VALUES(fail_count), retry_count = VALUES(retry_count),
		error_msg = VALUES(error_msg), last_crawl_at = VALUES(last_crawl_at)`,
		status.AnimeID, status.Status, status.CurrentIndex, status.TotalCount,
		status.SuccessCount, status.FailCount, status.RetryCount, status.ErrorMsg,
		status.LastCrawlAt)
	if err != nil {
		return fmt.Errorf("保存爬取状态失败: %w", err)
	}
	return nil
}

// LoadStatus 加载爬取状态。
func (s *MySQLStorage) LoadStatus(ctx context.Context, animeID int64) (*model.CrawlStatus, error) {
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
func (s *MySQLStorage) Close() error {
	return s.db.Close()
}

// saveAnimeTx 在事务内保存 anime 主记录。
func (s *MySQLStorage) saveAnimeTx(ctx context.Context, tx *sql.Tx, anime *model.AnimeInfo) error {
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
		INSERT INTO anime
		(id, title, year, region, tags, cover_image, description, update_date,
		 episode_num, update_num, douban_url, detail_url, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		title = VALUES(title), year = VALUES(year), region = VALUES(region),
		tags = VALUES(tags), cover_image = VALUES(cover_image), description = VALUES(description),
		update_date = VALUES(update_date), episode_num = VALUES(episode_num),
		update_num = VALUES(update_num), douban_url = VALUES(douban_url),
		detail_url = VALUES(detail_url), status = VALUES(status),
		created_at = VALUES(created_at), updated_at = VALUES(updated_at)`,
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
func (s *MySQLStorage) saveEpisodesTx(ctx context.Context, tx *sql.Tx, animeID int64, episodes []model.Episode) error {
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
func (s *MySQLStorage) loadEpisodes(ctx context.Context, animeID int64) ([]model.Episode, error) {
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

// Package s3storage 提供基于 S3 对象存储的 Storage 实现，以单动漫 JSON 对象形式归档。
package s3storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"jciyuan-spider/internal/model"
	"jciyuan-spider/internal/storage"
)

// S3Storage S3 对象存储实现，仅实现 Storage 接口。
type S3Storage struct {
	client     *s3.Client
	bucket     string
	keyPrefix  string
}

// NewS3Storage 创建 S3 存储实例。
func NewS3Storage(cfg model.S3StorageConfig) (*S3Storage, error) {
	// 允许配置为空时从环境变量读取凭证与区域。
	endpoint := firstNonEmpty(cfg.Endpoint, os.Getenv("AWS_ENDPOINT_URL_S3"), os.Getenv("AWS_ENDPOINT_URL"))
	region := firstNonEmpty(cfg.Region, os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"), "us-east-1")
	bucket := firstNonEmpty(cfg.Bucket, os.Getenv("S3_BUCKET"))
	accessKey := firstNonEmpty(cfg.AccessKeyID, os.Getenv("AWS_ACCESS_KEY_ID"))
	secretKey := firstNonEmpty(cfg.SecretAccessKey, os.Getenv("AWS_SECRET_ACCESS_KEY"))

	if bucket == "" {
		return nil, fmt.Errorf("S3 bucket 不能为空")
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	if accessKey != "" && secretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	awscfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("加载 AWS 配置失败: %w", err)
	}

	s3Opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = true
		},
	}
	if endpoint != "" {
		awscfg.BaseEndpoint = aws.String(endpoint)
	}

	client := s3.NewFromConfig(awscfg, s3Opts...)
	return &S3Storage{
		client:    client,
		bucket:    bucket,
		keyPrefix: strings.TrimSuffix(cfg.KeyPrefix, "/"),
	}, nil
}

// NewS3StorageFromConfig 根据 StorageConfig 构造 Storage（SPI 构造器签名）。
func NewS3StorageFromConfig(cfg model.StorageConfig) (storage.Storage, error) {
	return NewS3Storage(cfg.S3)
}

func init() {
	// SPI 注册 S3 Storage。
	storage.Register("s3", NewS3StorageFromConfig)
}

// Save 将动漫信息以 JSON 对象形式上传至 S3。
func (s *S3Storage) Save(ctx context.Context, anime *model.AnimeInfo) error {
	anime.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(anime, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化动漫信息失败: %w", err)
	}

	key := s.objectKey(anime.ID)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("上传 S3 对象 %s 失败: %w", key, err)
	}
	return nil
}

// Load 当前最小可用版未实现从 S3 加载，返回 nil。
func (s *S3Storage) Load(ctx context.Context, animeID int64) (*model.AnimeInfo, error) {
	return nil, nil
}

// Exists 检查 S3 对象是否存在。
func (s *S3Storage) Exists(ctx context.Context, animeID int64) (bool, error) {
	key := s.objectKey(animeID)
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// 简单判断：只要出错就认为不存在，避免引入额外依赖。
		return false, nil
	}
	return true, nil
}

// SaveBatch 批量保存动漫信息，每个动漫独立上传。
func (s *S3Storage) SaveBatch(ctx context.Context, animes []*model.AnimeInfo) error {
	for _, anime := range animes {
		if err := s.Save(ctx, anime); err != nil {
			return err
		}
	}
	return nil
}

// Close 关闭存储。
func (s *S3Storage) Close() error { return nil }

// objectKey 生成 S3 对象 Key。
func (s *S3Storage) objectKey(animeID int64) string {
	if s.keyPrefix != "" {
		return fmt.Sprintf("%s/%d.json", s.keyPrefix, animeID)
	}
	return fmt.Sprintf("%d.json", animeID)
}

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

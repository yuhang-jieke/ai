// Package storage 提供统一的对象存储接口
// 支持 MinIO、Aliyun OSS、RustFS 等多种存储后端
package storage

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage MinIO 存储实现
type MinIOStorage struct {
	client *minio.Client
	bucket string
}

// NewMinIOStorage 创建 MinIO 存储实例
func NewMinIOStorage(cfg *MinIOConfig) (*MinIOStorage, error) {
	// 验证 endpoint 端口
	slog.Info("MinIO 配置", "endpoint", cfg.Endpoint, "secure", cfg.UseSSL, "region", cfg.Region)

	// 解析 endpoint
	if cfg.Endpoint == "" {
		cfg.Endpoint = "localhost:9000"
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		slog.Error("MinIO 客户端创建失败", "error", err, "endpoint", cfg.Endpoint)
		return nil, err
	}

	bucket := cfg.Bucket
	if bucket == "" {
		bucket = "default"
	}

	// 检查并创建 bucket
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		slog.Error("检查 bucket 失败", "error", err, "bucket", bucket)
		return nil, err
	}

	if !exists {
		slog.Info("Bucket 不存在，正在创建", "bucket", bucket)
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{
			Region: cfg.Region,
		})
		if err != nil {
			slog.Error("创建 bucket 失败", "error", err, "bucket", bucket)
			return nil, err
		}
		slog.Info("Bucket 创建成功", "bucket", bucket)
	}

	slog.Info("MinIO 存储初始化成功", "endpoint", cfg.Endpoint, "bucket", bucket, "api_port", "9000")

	return &MinIOStorage{
		client: client,
		bucket: bucket,
	}, nil
}

// PutObject 上传对象
func (s *MinIOStorage) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: metadata,
	})
	return err
}

// GetObject 获取对象
func (s *MinIOStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, ObjectInfo{}, err
	}

	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, ObjectInfo{}, err
	}

	return obj, ObjectInfo{
		Key:          key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		LastModified: info.LastModified.Unix(),
		Metadata:     info.UserMetadata,
	}, nil
}

// DeleteObject 删除对象
func (s *MinIOStorage) DeleteObject(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// DeleteObjects 批量删除对象
func (s *MinIOStorage) DeleteObjects(ctx context.Context, keys []string) error {
	objectsCh := make(chan minio.ObjectInfo)
	errCh := s.client.RemoveObjects(ctx, s.bucket, objectsCh, minio.RemoveObjectsOptions{})

	go func() {
		defer close(objectsCh)
		for _, key := range keys {
			objectsCh <- minio.ObjectInfo{Key: key}
		}
	}()

	for err := range errCh {
		if err.Err != nil {
			return err.Err
		}
	}
	return nil
}

// StatObject 获取对象信息
func (s *MinIOStorage) StatObject(ctx context.Context, key string) (ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, err
	}

	return ObjectInfo{
		Key:          key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		LastModified: info.LastModified.Unix(),
		Metadata:     info.UserMetadata,
	}, nil
}

// ListObjects 列出对象
func (s *MinIOStorage) ListObjects(ctx context.Context, prefix, marker, delimiter string, maxKeys int) (ListObjectsResult, error) {
	var objects []ObjectInfo
	var commonPrefixes []string

	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: delimiter == "",
		MaxKeys:   maxKeys,
	}) {
		if object.Err != nil {
			return ListObjectsResult{}, object.Err
		}

		objects = append(objects, ObjectInfo{
			Key:          object.Key,
			Size:         object.Size,
			ContentType:  object.ContentType,
			ETag:         object.ETag,
			LastModified: object.LastModified.Unix(),
		})
	}

	return ListObjectsResult{
		Objects:      objects,
		Prefix:       prefix,
		Delimiter:    delimiter,
		CommonPrefix: commonPrefixes,
	}, nil
}

// CopyObject 复制对象
func (s *MinIOStorage) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	_, err := s.client.CopyObject(ctx, minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: dstKey,
	}, minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: srcKey,
	})
	return err
}

// PresignedURL 生成预签名URL
func (s *MinIOStorage) PresignedURL(ctx context.Context, key string, expires int64) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucket, key, time.Duration(expires)*time.Second, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// BucketExists 检查存储桶是否存在
func (s *MinIOStorage) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return s.client.BucketExists(ctx, bucket)
}

// CreateBucket 创建存储桶
func (s *MinIOStorage) CreateBucket(ctx context.Context, bucket string) error {
	return s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

// DeleteBucket 删除存储桶
func (s *MinIOStorage) DeleteBucket(ctx context.Context, bucket string) error {
	return s.client.RemoveBucket(ctx, bucket)
}

// ListBuckets 列出所有存储桶
func (s *MinIOStorage) ListBuckets(ctx context.Context) ([]string, error) {
	buckets, err := s.client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, b := range buckets {
		names = append(names, b.Name)
	}
	return names, nil
}

// Type 返回存储类型
func (s *MinIOStorage) Type() StorageType {
	return StorageTypeMinIO
}

// Close 关闭连接
func (s *MinIOStorage) Close() error {
	return nil
}

// Package storage 提供统一的对象存储接口
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSStorage 阿里云 OSS 存储实现
type OSSStorage struct {
	client     *oss.Client
	bucket     *oss.Bucket
	bucketName string
}

// NewOSSStorage 创建阿里云 OSS 存储实例
func NewOSSStorage(cfg *OSSConfig) (*OSSStorage, error) {
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	bucketName := cfg.Bucket
	if bucketName == "" {
		bucketName = "default"
	}

	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}

	slog.Info("阿里云 OSS 存储初始化成功", "endpoint", cfg.Endpoint, "bucket", bucketName)

	return &OSSStorage{
		client:     client,
		bucket:     bucket,
		bucketName: bucketName,
	}, nil
}

// PutObject 上传对象
func (s *OSSStorage) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	options := []oss.Option{
		oss.ContentType(contentType),
	}

	for k, v := range metadata {
		options = append(options, oss.Meta(k, v))
	}

	return s.bucket.PutObject(key, reader, options...)
}

// GetObject 获取对象
func (s *OSSStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	result, err := s.bucket.GetObject(key)
	if err != nil {
		return nil, ObjectInfo{}, err
	}

	// 获取对象元信息
	meta, err := s.bucket.GetObjectDetailedMeta(key)
	if err != nil {
		result.Close()
		return nil, ObjectInfo{}, err
	}

	info := ObjectInfo{
		Key:         key,
		ContentType: meta.Get("Content-Type"),
		ETag:        meta.Get("ETag"),
		Metadata:    make(map[string]string),
	}

	if contentLength := meta.Get("Content-Length"); contentLength != "" {
		fmt.Sscanf(contentLength, "%d", &info.Size)
	}

	if lastModified := meta.Get("Last-Modified"); lastModified != "" {
		if t, err := time.Parse(time.RFC1123, lastModified); err == nil {
			info.LastModified = t.Unix()
		}
	}

	return result, info, nil
}

// DeleteObject 删除对象
func (s *OSSStorage) DeleteObject(ctx context.Context, key string) error {
	return s.bucket.DeleteObject(key)
}

// DeleteObjects 批量删除对象
func (s *OSSStorage) DeleteObjects(ctx context.Context, keys []string) error {
	_, err := s.bucket.DeleteObjects(keys)
	return err
}

// StatObject 获取对象信息
func (s *OSSStorage) StatObject(ctx context.Context, key string) (ObjectInfo, error) {
	meta, err := s.bucket.GetObjectDetailedMeta(key)
	if err != nil {
		return ObjectInfo{}, err
	}

	info := ObjectInfo{
		Key:         key,
		ContentType: meta.Get("Content-Type"),
		ETag:        meta.Get("ETag"),
		Metadata:    make(map[string]string),
	}

	if contentLength := meta.Get("Content-Length"); contentLength != "" {
		fmt.Sscanf(contentLength, "%d", &info.Size)
	}

	if lastModified := meta.Get("Last-Modified"); lastModified != "" {
		if t, err := time.Parse(time.RFC1123, lastModified); err == nil {
			info.LastModified = t.Unix()
		}
	}

	return info, nil
}

// ListObjects 列出对象
func (s *OSSStorage) ListObjects(ctx context.Context, prefix, marker, delimiter string, maxKeys int) (ListObjectsResult, error) {
	options := []oss.Option{
		oss.MaxKeys(maxKeys),
	}

	if prefix != "" {
		options = append(options, oss.Prefix(prefix))
	}
	if marker != "" {
		options = append(options, oss.Marker(marker))
	}
	if delimiter != "" {
		options = append(options, oss.Delimiter(delimiter))
	}

	result, err := s.bucket.ListObjects(options...)
	if err != nil {
		return ListObjectsResult{}, err
	}

	var objects []ObjectInfo
	for _, obj := range result.Objects {
		objects = append(objects, ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			ETag:         obj.ETag,
			LastModified: obj.LastModified.Unix(),
		})
	}

	return ListObjectsResult{
		Objects:      objects,
		Prefix:       result.Prefix,
		Delimiter:    result.Delimiter,
		IsTruncated:  result.IsTruncated,
		NextMarker:   result.NextMarker,
		CommonPrefix: result.CommonPrefixes,
	}, nil
}

// CopyObject 复制对象
func (s *OSSStorage) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	_, err := s.bucket.CopyObject(srcKey, dstKey)
	return err
}

// PresignedURL 生成预签名URL
func (s *OSSStorage) PresignedURL(ctx context.Context, key string, expires int64) (string, error) {
	url, err := s.bucket.SignURL(key, oss.HTTPGet, expires)
	if err != nil {
		return "", err
	}
	return url, nil
}

// BucketExists 检查存储桶是否存在
func (s *OSSStorage) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return s.client.IsBucketExist(bucket)
}

// CreateBucket 创建存储桶
func (s *OSSStorage) CreateBucket(ctx context.Context, bucket string) error {
	return s.client.CreateBucket(bucket)
}

// DeleteBucket 删除存储桶
func (s *OSSStorage) DeleteBucket(ctx context.Context, bucket string) error {
	return s.client.DeleteBucket(bucket)
}

// ListBuckets 列出所有存储桶
func (s *OSSStorage) ListBuckets(ctx context.Context) ([]string, error) {
	result, err := s.client.ListBuckets()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, b := range result.Buckets {
		names = append(names, b.Name)
	}
	return names, nil
}

// Type 返回存储类型
func (s *OSSStorage) Type() StorageType {
	return StorageTypeOSS
}

// Close 关闭连接
func (s *OSSStorage) Close() error {
	return nil
}

// ============ RustFS 实现 ============

// RustFSStorage RustFS 存储实现
type RustFSStorage struct {
	endpoint string
	token    string
	bucket   string
	client   *http.Client
}

// NewRustFSStorage 创建 RustFS 存储实例
func NewRustFSStorage(cfg *RustFSConfig) (*RustFSStorage, error) {
	bucket := cfg.Bucket
	if bucket == "" {
		bucket = "default"
	}

	slog.Info("RustFS 存储初始化成功", "endpoint", cfg.Endpoint, "bucket", bucket)

	return &RustFSStorage{
		endpoint: cfg.Endpoint,
		token:    cfg.Token,
		bucket:   bucket,
		client:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// PutObject 上传对象
func (s *RustFSStorage) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) error {
	url := fmt.Sprintf("%s/buckets/%s/objects/%s", s.endpoint, s.bucket, key)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("put object failed: %d", resp.StatusCode)
	}

	return nil
}

// GetObject 获取对象
func (s *RustFSStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	url := fmt.Sprintf("%s/buckets/%s/objects/%s", s.endpoint, s.bucket, key)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, ObjectInfo{}, err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, ObjectInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, ObjectInfo{}, fmt.Errorf("get object failed: %d", resp.StatusCode)
	}

	info := ObjectInfo{
		Key:         key,
		ContentType: resp.Header.Get("Content-Type"),
		Metadata:    make(map[string]string),
	}

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		fmt.Sscanf(contentLength, "%d", &info.Size)
	}

	return resp.Body, info, nil
}

// DeleteObject 删除对象
func (s *RustFSStorage) DeleteObject(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/buckets/%s/objects/%s", s.endpoint, s.bucket, key)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete object failed: %d", resp.StatusCode)
	}

	return nil
}

// DeleteObjects 批量删除对象
func (s *RustFSStorage) DeleteObjects(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := s.DeleteObject(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// StatObject 获取对象信息
func (s *RustFSStorage) StatObject(ctx context.Context, key string) (ObjectInfo, error) {
	url := fmt.Sprintf("%s/buckets/%s/objects/%s", s.endpoint, s.bucket, key)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return ObjectInfo{}, err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return ObjectInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ObjectInfo{}, fmt.Errorf("stat object failed: %d", resp.StatusCode)
	}

	info := ObjectInfo{
		Key:         key,
		ContentType: resp.Header.Get("Content-Type"),
		ETag:        resp.Header.Get("ETag"),
		Metadata:    make(map[string]string),
	}

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		fmt.Sscanf(contentLength, "%d", &info.Size)
	}

	return info, nil
}

// ListObjects 列出对象
func (s *RustFSStorage) ListObjects(ctx context.Context, prefix, marker, delimiter string, maxKeys int) (ListObjectsResult, error) {
	url := fmt.Sprintf("%s/buckets/%s/objects?prefix=%s&max_keys=%d", s.endpoint, s.bucket, prefix, maxKeys)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ListObjectsResult{}, err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return ListObjectsResult{}, err
	}
	defer resp.Body.Close()

	// 简化处理，实际需要解析JSON响应
	return ListObjectsResult{
		Objects: []ObjectInfo{},
		Prefix:  prefix,
	}, nil
}

// CopyObject 复制对象
func (s *RustFSStorage) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	// 先获取源对象
	reader, _, err := s.GetObject(ctx, srcKey)
	if err != nil {
		return err
	}
	defer reader.Close()

	// 上传到目标位置
	var buf bytes.Buffer
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return err
	}

	return s.PutObject(ctx, dstKey, &buf, int64(buf.Len()), "", nil)
}

// PresignedURL 生成预签名URL
func (s *RustFSStorage) PresignedURL(ctx context.Context, key string, expires int64) (string, error) {
	// RustFS 需要实现预签名URL逻辑
	return fmt.Sprintf("%s/buckets/%s/objects/%s?token=%s&expires=%d", s.endpoint, s.bucket, key, s.token, expires), nil
}

// BucketExists 检查存储桶是否存在
func (s *RustFSStorage) BucketExists(ctx context.Context, bucket string) (bool, error) {
	url := fmt.Sprintf("%s/buckets/%s", s.endpoint, bucket)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// CreateBucket 创建存储桶
func (s *RustFSStorage) CreateBucket(ctx context.Context, bucket string) error {
	url := fmt.Sprintf("%s/buckets/%s", s.endpoint, bucket)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create bucket failed: %d", resp.StatusCode)
	}

	return nil
}

// DeleteBucket 删除存储桶
func (s *RustFSStorage) DeleteBucket(ctx context.Context, bucket string) error {
	url := fmt.Sprintf("%s/buckets/%s", s.endpoint, bucket)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete bucket failed: %d", resp.StatusCode)
	}

	return nil
}

// ListBuckets 列出所有存储桶
func (s *RustFSStorage) ListBuckets(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/buckets", s.endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 简化处理，实际需要解析JSON响应
	return []string{s.bucket}, nil
}

// Type 返回存储类型
func (s *RustFSStorage) Type() StorageType {
	return StorageTypeRustFS
}

// Close 关闭连接
func (s *RustFSStorage) Close() error {
	return nil
}

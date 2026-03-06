// Package storage 提供统一的对象存储接口
// 支持 MinIO、Aliyun OSS、RustFS 等多种存储后端
// 通过配置切换，使用同一个接口
package storage

import (
	"context"
	"io"
)

// StorageType 存储类型
type StorageType string

const (
	StorageTypeMinIO  StorageType = "minio"
	StorageTypeOSS    StorageType = "oss"
	StorageTypeRustFS StorageType = "rustfs"
)

// Config 存储配置
type Config struct {
	// Type 存储类型: minio, oss, rustfs
	Type StorageType `yaml:"type" json:"type"`

	// MinIO 配置
	MinIO MinIOConfig `yaml:"minio" json:"minio"`

	// OSS 配置
	OSS OSSConfig `yaml:"oss" json:"oss"`

	// RustFS 配置
	RustFS RustFSConfig `yaml:"rustfs" json:"rustfs"`

	// 通用配置
	Bucket string `yaml:"bucket" json:"bucket"` // 默认存储桶
	Region string `yaml:"region" json:"region"` // 地域
}

// MinIOConfig MinIO 配置
type MinIOConfig struct {
	Endpoint        string `yaml:"endpoint" json:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id" json:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key" json:"secret_access_key"`
	UseSSL          bool   `yaml:"use_ssl" json:"use_ssl"`
	Bucket          string `yaml:"bucket" json:"bucket"`
	Region          string `yaml:"region" json:"region"`
}

// OSSConfig 阿里云 OSS 配置
type OSSConfig struct {
	Endpoint        string `yaml:"endpoint" json:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id" json:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret" json:"access_key_secret"`
	Bucket          string `yaml:"bucket" json:"bucket"`
	Region          string `yaml:"region" json:"region"`
}

// RustFSConfig RustFS 配置
type RustFSConfig struct {
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	Token    string `yaml:"token" json:"token"`
	Bucket   string `yaml:"bucket" json:"bucket"`
}

// ObjectInfo 对象信息
type ObjectInfo struct {
	Key          string            // 对象键
	Size         int64             // 大小
	ContentType  string            // 内容类型
	ETag         string            // ETag
	LastModified int64             // 最后修改时间（Unix时间戳）
	Metadata     map[string]string // 元数据
}

// ListObjectsResult 列表结果
type ListObjectsResult struct {
	Objects      []ObjectInfo // 对象列表
	Prefix       string       // 前缀
	Delimiter    string       // 分隔符
	IsTruncated  bool         // 是否截断
	NextMarker   string       // 下一页标记
	CommonPrefix []string     // 公共前缀
}

// Storage 统一对象存储接口
// 所有存储后端必须实现此接口
type Storage interface {
	// PutObject 上传对象
	PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) error

	// GetObject 获取对象
	GetObject(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)

	// DeleteObject 删除对象
	DeleteObject(ctx context.Context, key string) error

	// DeleteObjects 批量删除对象
	DeleteObjects(ctx context.Context, keys []string) error

	// StatObject 获取对象信息
	StatObject(ctx context.Context, key string) (ObjectInfo, error)

	// ListObjects 列出对象
	ListObjects(ctx context.Context, prefix, marker, delimiter string, maxKeys int) (ListObjectsResult, error)

	// CopyObject 复制对象
	CopyObject(ctx context.Context, srcKey, dstKey string) error

	// PresignedURL 生成预签名URL（用于临时访问）
	PresignedURL(ctx context.Context, key string, expires int64) (string, error)

	// BucketExists 检查存储桶是否存在
	BucketExists(ctx context.Context, bucket string) (bool, error)

	// CreateBucket 创建存储桶
	CreateBucket(ctx context.Context, bucket string) error

	// DeleteBucket 删除存储桶
	DeleteBucket(ctx context.Context, bucket string) error

	// ListBuckets 列出所有存储桶
	ListBuckets(ctx context.Context) ([]string, error)

	// Type 返回存储类型
	Type() StorageType

	// Close 关闭连接
	Close() error
}

// UploadOption 上传选项
type UploadOption struct {
	ContentType string
	Metadata    map[string]string
}

// DownloadOption 下载选项
type DownloadOption struct {
	VersionID string
	Range     string // 字节范围，如 "bytes=0-1023"
}

// Package storage 提供统一的对象存储接口
// 支持 MinIO、Aliyun OSS、RustFS 等多种存储后端
package storage

import (
	"fmt"
	"log/slog"
)

// New 根据配置创建对应的存储实例
// 通过修改配置中的 Type 字段来切换不同的存储后端
// 支持: minio, oss, rustfs
func New(cfg *Config) (Storage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("storage config is nil")
	}

	slog.Info("初始化对象存储", "type", cfg.Type)

	switch cfg.Type {
	case StorageTypeMinIO:
		return NewMinIOStorage(&cfg.MinIO)
	case StorageTypeOSS:
		return NewOSSStorage(&cfg.OSS)
	case StorageTypeRustFS:
		return NewRustFSStorage(&cfg.RustFS)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s, supported: minio, oss, rustfs", cfg.Type)
	}
}

// MustNew 创建存储实例，如果失败则 panic
func MustNew(cfg *Config) Storage {
	s, err := New(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create storage: %v", err))
	}
	return s
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Type: StorageTypeMinIO,
		MinIO: MinIOConfig{
			Endpoint:        "115.190.57.118:9000",
			AccessKeyID:     "minioadmin",
			SecretAccessKey: "minioadmin",
			UseSSL:          false,
			Bucket:          "yuhang",
			Region:          "",
		},
	}
}

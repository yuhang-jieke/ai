// Package redis provides Redis client initialization and management
package redis

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"github.com/yuhang-jieke/ai/internal/config"
)

// Client Redis客户端单例
var Client *redis.Client

// Init 初始化Redis客户端
func Init(cfg *config.RedisConfig) error {
	if cfg == nil {
		return fmt.Errorf("redis config is nil")
	}

	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx := context.Background()
	_, err := Client.Ping(ctx).Result()
	if err != nil {
		slog.Error("Redis连接失败", "error", err, "host", cfg.Host, "port", cfg.Port)
		return fmt.Errorf("redis连接失败: %w", err)
	}

	slog.Info("Redis连接成功", "host", cfg.Host, "port", cfg.Port, "db", cfg.DB)
	return nil
}

// GetClient 获取Redis客户端实例
func GetClient() *redis.Client {
	return Client
}

// Close 关闭Redis连接
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

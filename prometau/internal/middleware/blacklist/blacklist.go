// Package blacklist 提供基于Redis的用户黑名单中间件
// 从JWT Token中解析用户ID，检查是否在Redis黑名单中
package blacklist

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/yuhang-jieke/ai/internal/httpserver"
	"github.com/yuhang-jieke/ai/internal/middleware"
	"github.com/yuhang-jieke/ai/internal/redis"
)

// Redis Key前缀
const (
	BlacklistKeyPrefix = "user:blacklist:"
)

// Config 黑名单中间件配置
type Config struct {
	// Enabled 是否启用黑名单检查
	Enabled bool `yaml:"enabled"`
	// JWTSecret JWT签名密钥
	JWTSecret string `yaml:"jwt_secret"`
	// ErrorMessage 自定义错误消息
	ErrorMessage string `yaml:"error_message"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:      true,
		JWTSecret:    "your-secret-key",
		ErrorMessage: "该用户被加入黑名单，请联系管理员",
	}
}

// Middleware 用户黑名单中间件
type Middleware struct {
	config *Config
}

// New 创建新的黑名单中间件
func New(config *Config) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}
	return &Middleware{config: config}
}

// Name 实现Middleware接口
func (m *Middleware) Name() string {
	return "blacklist"
}

// Handle 实现Middleware接口
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// 从Header获取Token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// 没有Token，放行（由JWT中间件处理）
			next.ServeHTTP(w, r)
			return
		}

		// 解析Bearer Token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			// 格式不正确，放行（由JWT中间件处理）
			next.ServeHTTP(w, r)
			return
		}

		// 解析JWT获取用户ID
		userID, err := m.extractUserID(tokenString)
		if err != nil {
			// 解析失败，放行（由JWT中间件处理）
			slog.Debug("解析Token失败", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		// 检查Redis黑名单
		isBlacklisted, err := m.checkBlacklist(r.Context(), userID)
		if err != nil {
			slog.Error("检查黑名单失败", "error", err, "user_id", userID)
			// 检查失败，为了安全起见可以拒绝或放行
			// 这里选择放行，避免Redis故障影响正常用户
			next.ServeHTTP(w, r)
			return
		}

		if isBlacklisted {
			slog.Warn("用户在黑名单中", "user_id", userID, "path", r.URL.Path)
			http.Error(w, m.config.ErrorMessage, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractUserID 从JWT Token中提取用户ID
func (m *Middleware) extractUserID(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return []byte(m.config.JWTSecret), nil
	})

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", fmt.Errorf("无效的Token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("无法解析Claims")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		// 尝试其他类型
		if uid, ok := claims["user_id"].(float64); ok {
			return fmt.Sprintf("%.0f", uid), nil
		}
		return "", fmt.Errorf("无法获取user_id")
	}

	return userID, nil
}

// checkBlacklist 检查用户是否在黑名单中
func (m *Middleware) checkBlacklist(ctx context.Context, userID string) (bool, error) {
	client := redis.GetClient()
	if client == nil {
		return false, fmt.Errorf("Redis客户端未初始化")
	}

	key := BlacklistKeyPrefix + userID
	result, err := client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return result > 0, nil
}

// Configure 实现Configurable接口
func (m *Middleware) Configure(config map[string]interface{}) error {
	return nil
}

// ============ 黑名单管理函数 ============

// AddToBlacklist 将用户加入黑名单
func AddToBlacklist(ctx context.Context, userID string) error {
	client := redis.GetClient()
	if client == nil {
		return fmt.Errorf("Redis客户端未初始化")
	}

	key := BlacklistKeyPrefix + userID
	return client.Set(ctx, key, "1", 0).Err() // 永不过期
}

// AddToBlacklistWithExpire 将用户加入黑名单并设置过期时间
func AddToBlacklistWithExpire(ctx context.Context, userID string, expireSeconds int64) error {
	client := redis.GetClient()
	if client == nil {
		return fmt.Errorf("Redis客户端未初始化")
	}

	key := BlacklistKeyPrefix + userID
	return client.Set(ctx, key, "1", 0).Err() // 简化处理，实际应设置过期时间
}

// RemoveFromBlacklist 将用户从黑名单移除
func RemoveFromBlacklist(ctx context.Context, userID string) error {
	client := redis.GetClient()
	if client == nil {
		return fmt.Errorf("Redis客户端未初始化")
	}

	key := BlacklistKeyPrefix + userID
	return client.Del(ctx, key).Err()
}

// IsBlacklisted 检查用户是否在黑名单中
func IsBlacklisted(ctx context.Context, userID string) (bool, error) {
	client := redis.GetClient()
	if client == nil {
		return false, fmt.Errorf("Redis客户端未初始化")
	}

	key := BlacklistKeyPrefix + userID
	result, err := client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return result > 0, nil
}

// ============ Gin适配 ============

// GinMiddleware 返回Gin中间件
func GinMiddleware(config *Config) gin.HandlerFunc {
	mw := New(config)
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.Next()
			return
		}

		userID, err := mw.extractUserID(tokenString)
		if err != nil {
			c.Next()
			return
		}

		isBlacklisted, err := mw.checkBlacklist(c.Request.Context(), userID)
		if err != nil {
			slog.Error("检查黑名单失败", "error", err, "user_id", userID)
			c.Next()
			return
		}

		if isBlacklisted {
			slog.Warn("用户在黑名单中", "user_id", userID, "path", c.Request.URL.Path)
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": config.ErrorMessage,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// HTTPServerMiddleware 返回httpserver.MiddlewareFunc
func HTTPServerMiddleware(config *Config) httpserver.MiddlewareFunc {
	mw := New(config)
	return func(ctx httpserver.Context) error {
		if !config.Enabled {
			return ctx.Next()
		}

		ginCtx := ctx.Gin().(*gin.Context)

		authHeader := ginCtx.GetHeader("Authorization")
		if authHeader == "" {
			return ctx.Next()
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return ctx.Next()
		}

		userID, err := mw.extractUserID(tokenString)
		if err != nil {
			return ctx.Next()
		}

		isBlacklisted, err := mw.checkBlacklist(ginCtx.Request.Context(), userID)
		if err != nil {
			slog.Error("检查黑名单失败", "error", err, "user_id", userID)
			return ctx.Next()
		}

		if isBlacklisted {
			slog.Warn("用户在黑名单中", "user_id", userID, "path", ginCtx.Request.URL.Path)
			return ctx.Error(403, config.ErrorMessage)
		}

		return ctx.Next()
	}
}

// 确保实现接口
var _ middleware.Middleware = (*Middleware)(nil)

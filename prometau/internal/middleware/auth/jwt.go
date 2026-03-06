// Package auth 提供基于 JWT 的身份认证中间件
// 支持多种 Token 提取方式和路径豁免
package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ContextKey 定义上下文键的类型
type ContextKey string

const (
	// UserIDKey 用户 ID 的上下文键
	UserIDKey ContextKey = "user_id"
	// UserNameKey 用户名的上下文键
	UserNameKey ContextKey = "user_name"
	// TokenKey JWT Token 的上下文键
	TokenKey ContextKey = "token"
)

// Config 存储 JWT 认证配置
type Config struct {
	// Secret JWT 签名密钥，生产环境应使用环境变量
	Secret string `yaml:"secret"`
	// SigningMethod 签名算法（如 HS256）
	SigningMethod jwt.SigningMethod `yaml:"signing_method"`
	// Expiration Token 过期时间，默认 24 小时
	Expiration time.Duration `yaml:"expiration"`
	// Exempts 免认证路径列表，支持通配符（如 /api/v1/auth/*）
	Exempts []string `yaml:"exempts"`
	// TokenLookup Token 查找位置，格式为 "来源:键名"
	// 支持的来源：header, query, cookie
	TokenLookup []string `yaml:"token_lookup"`
}

// DefaultConfig 返回默认的 JWT 配置
// 用于快速初始化和测试
func DefaultConfig() *Config {
	return &Config{
		Secret:        "your-secret-key", // 生产环境应修改
		SigningMethod: jwt.SigningMethodHS256,
		Expiration:    24 * time.Hour,
		Exempts:       []string{"/health", "/api/v1/auth/*"},
		TokenLookup:   []string{"header:Authorization"},
	}
}

// Claims 表示 JWT Token 中的声明信息
// 包含用户自定义声明和标准声明
type Claims struct {
	// UserID 用户 ID
	UserID string `json:"user_id"`
	// UserName 用户名
	UserName string `json:"user_name"`
	// RegisteredClaims JWT 标准声明（过期时间、签发时间等）
	jwt.RegisteredClaims
}

// Middleware 实现 JWT 认证中间件
type Middleware struct {
	config *Config
}

// New 创建新的 JWT 认证中间件
// config: JWT 配置，如果为 nil 则使用默认配置
// 返回：JWT 中间件实例
func New(config *Config) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}
	if config.SigningMethod == nil {
		config.SigningMethod = jwt.SigningMethodHS256
	}
	return &Middleware{config: config}
}

// Name 实现 Middleware 接口
// 返回中间件名称 "jwt"
func (m *Middleware) Name() string {
	return "jwt"
}

// Handle 实现 Middleware 接口
// 处理 HTTP 请求，进行 JWT 认证
// next: 下一个处理器
// 返回：包装后的处理器
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查路径是否需要认证
		if m.isExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// 从请求中提取 Token
		tokenString, err := m.extractToken(r)
		if err != nil {
			slog.Warn("failed to extract token", "path", r.URL.Path, "error", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 解析和验证 Token
		claims, err := m.validateToken(tokenString)
		if err != nil {
			slog.Warn("invalid token", "path", r.URL.Path, "error", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 将用户信息添加到请求上下文
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserNameKey, claims.UserName)
		ctx = context.WithValue(ctx, TokenKey, tokenString)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractToken 从 HTTP 请求中提取 JWT Token
// 支持从 Header、Query 参数、Cookie 中提取
// r: HTTP 请求
// 返回：Token 字符串和错误信息
func (m *Middleware) extractToken(r *http.Request) (string, error) {
	for _, lookup := range m.config.TokenLookup {
		parts := strings.Split(lookup, ":")
		if len(parts) != 2 {
			continue
		}

		source := parts[0]
		key := parts[1]

		switch source {
		case "header":
			// 从请求头提取（默认从 Authorization Header）
			value := r.Header.Get(key)
			if value != "" {
				// 移除 "Bearer " 前缀（如果存在）
				return strings.TrimPrefix(value, "Bearer "), nil
			}

		case "query":
			// 从 URL 查询参数提取
			value := r.URL.Query().Get(key)
			if value != "" {
				return value, nil
			}

		case "cookie":
			// 从 Cookie 提取
			cookie, err := r.Cookie(key)
			if err == nil && cookie.Value != "" {
				return cookie.Value, nil
			}
		}
	}

	return "", errors.New("token not found")
}

// validateToken 验证和解析 JWT Token
// tokenString: JWT Token 字符串
// 返回：解析后的 Claims 和错误信息
func (m *Middleware) validateToken(tokenString string) (*Claims, error) {
	// 解析 Token 并验证签名
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	// 检查 Token 是否有效
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// 类型断言获取 Claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
}

// isExempt 检查路径是否豁免于认证
// path: 请求路径
// 返回：是否需要认证
func (m *Middleware) isExempt(path string) bool {
	for _, exempt := range m.config.Exempts {
		if exempt == path {
			return true
		}
		// 支持通配符匹配（如 /api/v1/auth/*）
		if strings.HasSuffix(exempt, "/*") {
			prefix := strings.TrimSuffix(exempt, "/*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}

// GenerateToken 为用户生成新的 JWT Token
// config: JWT 配置
// userID: 用户 ID
// userName: 用户名
// 返回：JWT Token 字符串和错误信息
func GenerateToken(config *Config, userID, userName string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		UserName: userName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.Expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(config.SigningMethod, claims)
	return token.SignedString([]byte(config.Secret))
}

// Configure 实现 Configurable 接口
// 用于动态配置中间件（暂未实现）
func (m *Middleware) Configure(config map[string]interface{}) error {
	// TODO: 实现动态配置
	return nil
}

// GetUser 从请求上下文中获取用户信息
// ctx: 请求上下文
// 返回：用户 ID、用户名、是否成功
func GetUser(ctx context.Context) (userID string, userName string, ok bool) {
	userID, ok = ctx.Value(UserIDKey).(string)
	if !ok {
		return "", "", false
	}
	userName, ok = ctx.Value(UserNameKey).(string)
	return userID, userName, ok
}

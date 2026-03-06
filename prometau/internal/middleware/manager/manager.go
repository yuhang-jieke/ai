// Package manager 提供统一的中间件管理器
// 整合黑白名单、限流、认证、CORS等中间件
package manager

import (
	"net/http"

	"github.com/yuhang-jieke/ai/internal/middleware"
	"github.com/yuhang-jieke/ai/internal/middleware/auth"
	"github.com/yuhang-jieke/ai/internal/middleware/cors"
	"github.com/yuhang-jieke/ai/internal/middleware/ipfilter"
	"github.com/yuhang-jieke/ai/internal/middleware/ratelimit"
	"github.com/yuhang-jieke/ai/internal/middleware/recovery"
)

// MiddlewareType 中间件类型
type MiddlewareType string

const (
	MiddlewareRecovery  MiddlewareType = "recovery"  // 恢复中间件
	MiddlewareCORS      MiddlewareType = "cors"      // CORS中间件
	MiddlewareIPFilter  MiddlewareType = "ipfilter"  // IP黑白名单
	MiddlewareRateLimit MiddlewareType = "ratelimit" // 限流中间件
	MiddlewareJWT       MiddlewareType = "jwt"       // JWT认证
)

// Manager 中间件管理器
type Manager struct {
	chain       *middleware.Chain
	middlewares map[MiddlewareType]middleware.Middleware
}

// NewManager 创建新的中间件管理器
func NewManager() *Manager {
	return &Manager{
		chain:       middleware.NewChain(),
		middlewares: make(map[MiddlewareType]middleware.Middleware),
	}
}

// Use 添加中间件到管理器
func (m *Manager) Use(mt MiddlewareType, mw middleware.Middleware) *Manager {
	m.middlewares[mt] = mw
	m.chain.Use(mw)
	return m
}

// Build 构建中间件链
func (m *Manager) Build(finalHandler http.Handler) http.Handler {
	return m.chain.Build(finalHandler)
}

// Get 获取指定类型的中间件
func (m *Manager) Get(mt MiddlewareType) middleware.Middleware {
	return m.middlewares[mt]
}

// Chain 获取中间件链
func (m *Manager) Chain() *middleware.Chain {
	return m.chain
}

// Remove 移除指定类型的中间件
func (m *Manager) Remove(mt MiddlewareType) {
	delete(m.middlewares, mt)
	// 需要重建链
	m.chain = middleware.NewChain()
	for _, mw := range m.middlewares {
		m.chain.Use(mw)
	}
}

// ============ 便捷创建方法 ============

// WithRecovery 添加恢复中间件
func (m *Manager) WithRecovery(config *recovery.Config) *Manager {
	return m.Use(MiddlewareRecovery, recovery.New(config))
}

// WithCORS 添加CORS中间件
func (m *Manager) WithCORS(config *cors.Config) *Manager {
	return m.Use(MiddlewareCORS, cors.New(config))
}

// WithIPFilter 添加IP黑白名单中间件
func (m *Manager) WithIPFilter(config *ipfilter.Config) *Manager {
	return m.Use(MiddlewareIPFilter, ipfilter.New(config))
}

// WithJWT 添加JWT认证中间件
func (m *Manager) WithJWT(config *auth.Config) *Manager {
	return m.Use(MiddlewareJWT, auth.New(config))
}

// WithRateLimit 添加限流中间件(基于sentinel)
func (m *Manager) WithRateLimit(config *ratelimit.Config) *Manager {
	return m.Use(MiddlewareRateLimit, ratelimit.New(config))
}

// ============ 预设中间件链 ============

// DefaultChain 创建默认中间件链(Recovery -> CORS -> RateLimit)
func DefaultChain() *Manager {
	m := NewManager()
	m.WithRecovery(nil).
		WithCORS(nil).
		WithRateLimit(ratelimit.DefaultConfig())
	return m
}

// SecureChain 创建安全中间件链(Recovery -> CORS -> IPFilter -> JWT)
func SecureChain() *Manager {
	m := NewManager()
	m.WithRecovery(nil).
		WithCORS(nil).
		WithIPFilter(nil).
		WithJWT(nil)
	return m
}

// FullChain 创建完整中间件链(Recovery -> CORS -> RateLimit -> IPFilter -> JWT)
func FullChain() *Manager {
	m := NewManager()
	m.WithRecovery(nil).
		WithCORS(nil).
		WithRateLimit(ratelimit.DefaultConfig()).
		WithIPFilter(nil).
		WithJWT(nil)
	return m
}

// APIChain 创建API专用中间件链(RateLimit -> JWT)
// 适用于需要认证的API端点
func APIChain() *Manager {
	m := NewManager()
	m.WithRateLimit(ratelimit.DefaultConfig()).
		WithJWT(nil)
	return m
}

// ============ 全局单例管理 ============

var defaultManager *Manager

// InitDefault 初始化默认管理器
func InitDefault() {
	if defaultManager == nil {
		defaultManager = DefaultChain()
	}
}

// Default 获取默认管理器
func Default() *Manager {
	if defaultManager == nil {
		InitDefault()
	}
	return defaultManager
}

// SetDefault 设置默认管理器
func SetDefault(m *Manager) {
	defaultManager = m
}

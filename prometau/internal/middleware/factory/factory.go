// Package factory 提供中间件工厂，用于快速创建常用中间件链
package factory

import (
	"github.com/yuhang-jieke/ai/internal/middleware"
	"github.com/yuhang-jieke/ai/internal/middleware/auth"
	"github.com/yuhang-jieke/ai/internal/middleware/cors"
	"github.com/yuhang-jieke/ai/internal/middleware/ipfilter"
	"github.com/yuhang-jieke/ai/internal/middleware/recovery"
)

// MiddlewareFactory 中间件工厂
// 用于创建常用的中间件实例和中间件链
type MiddlewareFactory struct{}

// New 创建新的中间件工厂
// 返回：中间件工厂实例
func New() *MiddlewareFactory {
	return &MiddlewareFactory{}
}

// CreateDefaultChain 创建默认中间件链
// 包含常用的中间件：Recovery -> CORS -> IPFilter -> JWT Auth
// 返回：配置好的中间件链
func (f *MiddlewareFactory) CreateDefaultChain() *middleware.Chain {
	chain := middleware.NewChain()
	chain.Use(recovery.New(nil))
	chain.Use(cors.New(nil))
	chain.Use(ipfilter.New(nil))
	chain.Use(auth.New(nil))
	return chain
}

// CreateMinimalChain 创建最小中间件链
// 仅包含基础中间件：Recovery -> CORS
// 返回：配置好的中间件链
func (f *MiddlewareFactory) CreateMinimalChain() *middleware.Chain {
	chain := middleware.NewChain()
	chain.Use(recovery.New(nil))
	chain.Use(cors.New(nil))
	return chain
}

// CreateSecureChain 创建安全增强中间件链
// 包含所有安全相关中间件：Recovery -> CORS -> IPFilter -> JWT Auth
// 返回：配置好的中间件链
func (f *MiddlewareFactory) CreateSecureChain() *middleware.Chain {
	chain := middleware.NewChain()
	chain.Use(recovery.New(nil))
	chain.Use(cors.New(nil))
	chain.Use(ipfilter.New(nil))
	chain.Use(auth.New(nil))
	return chain
}

// ============ 便捷函数 ============

// NewRecovery 创建恢复中间件（便捷函数）
// config: 配置，传 nil 使用默认配置
// 返回：恢复中间件
func NewRecovery(config *recovery.Config) middleware.Middleware {
	return recovery.New(config)
}

// NewCORS 创建 CORS 中间件（便捷函数）
// config: 配置，传 nil 使用默认配置
// 返回：CORS 中间件
func NewCORS(config *cors.Config) middleware.Middleware {
	return cors.New(config)
}

// NewIPFilter 创建 IP 过滤中间件（便捷函数）
// config: 配置，传 nil 使用默认配置
// 返回：IP 过滤中间件
func NewIPFilter(config *ipfilter.Config) middleware.Middleware {
	return ipfilter.New(config)
}

// NewJWT 创建 JWT 认证中间件（便捷函数）
// config: 配置，传 nil 使用默认配置
// 返回：JWT 认证中间件
func NewJWT(config *auth.Config) middleware.Middleware {
	return auth.New(config)
}

// ============ 便捷函数结束 ============

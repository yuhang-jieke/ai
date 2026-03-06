// Package middleware 提供 HTTP 中间件链和接口定义
// 用于构建可组合的 HTTP 请求处理管道
package middleware

import (
	"net/http"
)

// Middleware 定义 HTTP 中间件接口
// 所有中间件都必须实现此接口
type Middleware interface {
	// Name 返回中间件名称，用于日志和调试
	Name() string

	// Handle 将当前中间件包装到下一个处理器上
	// next: 下一个处理器
	// 返回：包装后的处理器
	Handle(next http.Handler) http.Handler
}

// Configurable 接口由支持动态配置的中间件实现
// 允许在运行时修改中间件配置
type Configurable interface {
	// Configure 配置中间件参数
	// config: 配置映射表
	// 返回：错误信息（如果有）
	Configure(config map[string]interface{}) error
}

// Chain 管理中间件链
// 按照添加顺序依次执行中间件
type Chain struct {
	// middlewares 中间件列表
	middlewares []Middleware
}

// NewChain 创建新的中间件链
// 返回：新创建的中间件链实例
func NewChain() *Chain {
	return &Chain{
		middlewares: make([]Middleware, 0),
	}
}

// Use 向链中添加中间件
// mw: 要添加的中间件
// 返回：当前链（支持链式调用）
func (c *Chain) Use(mw Middleware) *Chain {
	c.middlewares = append(c.middlewares, mw)
	return c
}

// Build 构建最终的处理器，应用所有中间件
// 中间件按照与添加相反的顺序包装（LIFO）
// finalHandler: 最终的业务处理器
// 返回：包装后的完整处理器
func (c *Chain) Build(finalHandler http.Handler) http.Handler {
	handler := finalHandler
	// 从后向前包装，确保执行顺序与添加顺序一致
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i].Handle(handler)
	}
	return handler
}

// Middlewares 返回链中的所有中间件
// 返回：中间件切片
func (c *Chain) Middlewares() []Middleware {
	return c.middlewares
}

// MiddlewareFunc 函数类型的中间件适配器
// 允许将普通函数转换为中间件
type MiddlewareFunc func(http.Handler) http.Handler

// Handle 实现 Middleware 接口
// 将函数调用委托给底层函数
func (mf MiddlewareFunc) Handle(next http.Handler) http.Handler {
	return mf(next)
}

// Name 实现 Middleware 接口
// 返回函数类型中间件的名称
func (mf MiddlewareFunc) Name() string {
	return "MiddlewareFunc"
}

// Handler 便捷的处理器函数类型
// 简化中间件编写
type Handler func(http.ResponseWriter, *http.Request)

// AdaptMiddleware 将普通 Handler 转换为 Middleware
// handler: 普通的 HTTP 处理器函数
// 返回：包装后的中间件
func AdaptMiddleware(handler Handler) Middleware {
	return MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 先执行当前处理器
			handler(w, r)
			// 再执行下一个处理器
			next.ServeHTTP(w, r)
		})
	})
}

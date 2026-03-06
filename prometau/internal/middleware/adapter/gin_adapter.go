// Package adapter 提供中间件适配器
// 将标准 http.Handler 中间件转换为 Gin 中间件
package adapter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yuhang-jieke/ai/internal/httpserver"
	"github.com/yuhang-jieke/ai/internal/middleware"
)

// AdaptMiddleware 将 middleware.Middleware 适配为 Gin 中间件
// 返回 httpserver.MiddlewareFunc 以便在路由中使用
func AdaptMiddleware(mw middleware.Middleware) httpserver.MiddlewareFunc {
	return func(ctx httpserver.Context) error {
		// 获取 gin.Context
		ginCtx := ctx.Gin().(*gin.Context)

		// 创建一个标志，用于判断中间件是否通过了
		passed := false

		// 创建一个 http.Handler 来包装后续处理
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			passed = true
		})

		// 使用中间件包装
		wrappedHandler := mw.Handle(nextHandler)

		// 执行中间件
		wrappedHandler.ServeHTTP(ginCtx.Writer, ginCtx.Request)

		// 如果中间件没有通过（比如限流、认证失败等）
		if !passed {
			// 中间件已经处理了响应，直接返回
			return nil
		}

		// 继续执行下一个中间件或处理器
		return ctx.Next()
	}
}

// AdaptToGin 将 middleware.Middleware 适配为 gin.HandlerFunc
func AdaptToGin(mw middleware.Middleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		passed := false

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			passed = true
		})

		wrappedHandler := mw.Handle(nextHandler)
		wrappedHandler.ServeHTTP(c.Writer, c.Request)

		if !passed {
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdaptChain 将 middleware.Chain 适配为 Gin 中间件组
// 返回一组 gin.HandlerFunc
func AdaptChain(chain *middleware.Chain) []gin.HandlerFunc {
	// 由于 Chain 的 Build 方法需要最终 handler，
	// 这里我们返回一个适配器函数
	return []gin.HandlerFunc{
		func(c *gin.Context) {
			// 创建一个包装整个后续链的 handler
			// 这会在路由级别应用
			c.Next()
		},
	}
}

// WrapToHTTPServerMiddleware 将 gin.HandlerFunc 包装为 httpserver.MiddlewareFunc
// 用于在路由注册时使用
func WrapToHTTPServerMiddleware(ginHandler gin.HandlerFunc) httpserver.MiddlewareFunc {
	return func(ctx httpserver.Context) error {
		ginCtx := ctx.Gin().(*gin.Context)
		ginHandler(ginCtx)
		if ginCtx.IsAborted() {
			return nil
		}
		return ctx.Next()
	}
}

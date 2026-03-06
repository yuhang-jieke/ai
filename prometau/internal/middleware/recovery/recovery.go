// Package recovery 提供 Panic 恢复中间件
// 捕获 HTTP 处理过程中的 panic，防止服务崩溃
package recovery

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Config 存储恢复中间件配置
type Config struct {
	// Enabled 是否启用恢复中间件
	Enabled bool `yaml:"enabled"`
	// LogStack 是否记录堆栈信息
	LogStack bool `yaml:"log_stack"`
	// StackTraceLen 堆栈跟踪长度（字节）
	StackTraceLen int `yaml:"stack_trace_len"`
	// ResponseBody 错误响应内容
	ResponseBody string `yaml:"response_body"`
}

// DefaultConfig 返回默认的恢复中间件配置
// 用于快速初始化和测试
func DefaultConfig() *Config {
	return &Config{
		Enabled:       true,
		LogStack:      true,
		StackTraceLen: 1024,
		ResponseBody:  "Internal Server Error",
	}
}

// Middleware 实现 Panic 恢复中间件
type Middleware struct {
	config *Config
}

// New 创建新的恢复中间件
// config: 恢复中间件配置，如果为 nil 则使用默认配置
// 返回：恢复中间件实例
func New(config *Config) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}
	return &Middleware{config: config}
}

// Name 实现 Middleware 接口
// 返回中间件名称 "recovery"
func (m *Middleware) Name() string {
	return "recovery"
}

// Handle 实现 Middleware 接口
// 捕获处理过程中的 panic 并进行恢复
// next: 下一个处理器
// 返回：包装后的处理器
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 如果未启用，直接跳过
		if !m.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// 使用 defer 捕获 panic
		defer func() {
			if err := recover(); err != nil {
				// 记录错误日志
				slog.Error("捕获到 Panic",
					"error", err,
					"path", r.URL.Path,
					"method", r.Method,
				)

				// 记录堆栈跟踪
				if m.config.LogStack {
					stack := debug.Stack()
					slog.Error("Panic 堆栈跟踪",
						"stack", string(stack[:m.config.StackTraceLen]),
					)
				}

				// 返回错误响应
				http.Error(w, m.config.ResponseBody, http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Configure 实现 Configurable 接口
// 用于动态配置中间件（暂未实现）
func (m *Middleware) Configure(config map[string]interface{}) error {
	// TODO: 实现动态配置
	return nil
}

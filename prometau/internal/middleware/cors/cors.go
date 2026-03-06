// Package cors 提供跨域资源共享 (CORS) 中间件
// 支持预检请求处理、自定义响应头等功能
package cors

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Config 存储 CORS 配置
type Config struct {
	// AllowOrigins 允许的源列表（如 https://example.com）
	AllowOrigins []string `yaml:"allow_origins"`
	// AllowMethods 允许的 HTTP 方法
	AllowMethods []string `yaml:"allow_methods"`
	// AllowHeaders 允许的 HTTP 头部
	AllowHeaders []string `yaml:"allow_headers"`
	// AllowCredentials 是否允许凭证（Cookie、Authorization 等）
	AllowCredentials bool `yaml:"allow_credentials"`
	// ExposeHeaders 暴露给浏览器的头部列表
	ExposeHeaders []string `yaml:"expose_headers"`
	// MaxAge 预检请求的缓存时间（秒）
	MaxAge time.Duration `yaml:"max_age"`
	// AllowAllOrigins 是否允许所有源（生产环境慎用）
	AllowAllOrigins bool `yaml:"allow_all_origins"`
}

// DefaultConfig 返回默认的 CORS 配置
// 默认允许所有源和常用方法
func DefaultConfig() *Config {
	return &Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: false,
		ExposeHeaders:    []string{},
		MaxAge:           12 * time.Hour,
		AllowAllOrigins:  true,
	}
}

// Middleware 实现 CORS 中间件
type Middleware struct {
	config *Config
}

// New 创建新的 CORS 中间件
// config: CORS 配置，如果为 nil 则使用默认配置
// 返回：CORS 中间件实例
func New(config *Config) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}
	return &Middleware{config: config}
}

// Name 实现 Middleware 接口
// 返回中间件名称 "cors"
func (m *Middleware) Name() string {
	return "cors"
}

// Handle 实现 Middleware 接口
// 处理 CORS 请求，设置响应头
// next: 下一个处理器
// 返回：包装后的处理器
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 处理预检请求（OPTIONS 方法）
		if r.Method == "OPTIONS" {
			m.setCORSHeaders(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 为普通请求设置 CORS 头部
		m.setCORSHeaders(w, r)
		next.ServeHTTP(w, r)
	})
}

// setCORSHeaders 设置 CORS 响应头部
// w: HTTP 响应写入器
// r: HTTP 请求
func (m *Middleware) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	// 检查源是否被允许
	allowOrigin := m.getAllowOrigin(origin)
	if allowOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
	}

	// 允许凭证
	if m.config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// 暴露头部
	if len(m.config.ExposeHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(m.config.ExposeHeaders, ", "))
	}

	// 处理预检请求
	if r.Method == "OPTIONS" {
		// 允许的方法
		if len(m.config.AllowMethods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.config.AllowMethods, ", "))
		}

		// 允许的头部
		if len(m.config.AllowHeaders) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.config.AllowHeaders, ", "))
		}

		// 预检请求缓存时间
		if m.config.MaxAge > 0 {
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(int(m.config.MaxAge.Seconds())))
		}
	}
}

// getAllowOrigin 获取允许的源
// origin: 请求的源
// 返回：允许的源，如果不允许则返回空字符串
func (m *Middleware) getAllowOrigin(origin string) string {
	// 如果允许所有源，返回通配符
	if m.config.AllowAllOrigins {
		return "*"
	}

	// 如果没有源头部，返回空
	if origin == "" {
		return ""
	}

	// 检查是否在允许列表中
	for _, allowed := range m.config.AllowOrigins {
		if allowed == origin || allowed == "*" {
			return origin
		}
	}

	return ""
}

// Configure 实现 Configurable 接口
// 用于动态配置中间件（暂未实现）
func (m *Middleware) Configure(config map[string]interface{}) error {
	// TODO: 实现动态配置
	return nil
}

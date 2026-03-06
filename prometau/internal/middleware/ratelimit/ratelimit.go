// Package ratelimit 提供基于 Alibaba Sentinel 的限流中间件
package ratelimit

import (
	"log/slog"
	"net/http"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/flow"
)

// Config 限流配置
type Config struct {
	// Enabled 是否启用限流
	Enabled bool `yaml:"enabled"`
	// ResourceName 资源名称（用于sentinel统计）
	ResourceName string `yaml:"resource_name"`
	// Threshold QPS阈值
	Threshold float64 `yaml:"threshold"`
	// ControlBehavior 控制行为 (Reject/WarmUp/Throttling)
	ControlBehavior flow.ControlBehavior `yaml:"control_behavior"`
	// RelationStrategy 关联策略
	RelationStrategy flow.RelationStrategy `yaml:"relation_strategy"`
	// WarmUpPeriodSec 预热周期（秒），仅WarmUp模式有效
	WarmUpPeriodSec uint32 `yaml:"warm_up_period_sec"`
	// MaxQueueingTimeMs 最大排队时间（毫秒），仅Throttling模式有效
	MaxQueueingTimeMs uint32 `yaml:"max_queueing_time_ms"`
}

// DefaultConfig 返回默认限流配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:          true,
		ResourceName:     "default-api",
		Threshold:        1000,
		ControlBehavior:  flow.Reject,
		RelationStrategy: 0, // Direct
	}
}

// Middleware 限流中间件
type Middleware struct {
	config  *Config
	rule    *flow.Rule
	enabled bool
}

// New 创建新的限流中间件
// config: 限流配置，如果为 nil 则使用默认配置
func New(config *Config) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}

	m := &Middleware{
		config:  config,
		enabled: config.Enabled,
	}

	if m.enabled {
		m.initRule()
	}

	return m
}

// initRule 初始化Sentinel限流规则
func (m *Middleware) initRule() {
	m.rule = &flow.Rule{
		Resource:               m.config.ResourceName,
		TokenCalculateStrategy: flow.Direct,
		ControlBehavior:        m.config.ControlBehavior,
		Threshold:              m.config.Threshold,
		RelationStrategy:       m.config.RelationStrategy,
		WarmUpPeriodSec:        m.config.WarmUpPeriodSec,
		MaxQueueingTimeMs:      m.config.MaxQueueingTimeMs,
	}

	// 加载规则
	_, err := flow.LoadRules([]*flow.Rule{m.rule})
	if err != nil {
		slog.Error("加载Sentinel限流规则失败", "error", err, "resource", m.config.ResourceName)
	} else {
		slog.Info("Sentinel限流规则加载成功", "resource", m.config.ResourceName, "threshold", m.config.Threshold)
	}
}

// Name 实现Middleware接口
func (m *Middleware) Name() string {
	return "ratelimit"
}

// Handle 实现Middleware接口
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// 使用Sentinel进行限流
		entry, blockError := sentinel.Entry(m.config.ResourceName)
		if blockError != nil {
			// 被限流
			slog.Warn("请求被限流",
				"resource", m.config.ResourceName,
				"path", r.URL.Path,
				"method", r.Method,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// 确保退出entry
		defer entry.Exit()

		// 继续处理请求
		next.ServeHTTP(w, r)
	})
}

// Configure 实现Configurable接口
func (m *Middleware) Configure(config map[string]interface{}) error {
	// 支持动态更新配置
	if threshold, ok := config["threshold"].(float64); ok {
		m.config.Threshold = threshold
		if m.rule != nil {
			m.rule.Threshold = threshold
		}
	}
	return nil
}

// UpdateThreshold 动态更新QPS阈值
func (m *Middleware) UpdateThreshold(threshold float64) {
	if m.rule != nil {
		m.rule.Threshold = threshold
		m.config.Threshold = threshold
		// 重新加载规则
		flow.LoadRules([]*flow.Rule{m.rule})
	}
}

// SetEnabled 启用或禁用限流
func (m *Middleware) SetEnabled(enabled bool) {
	m.enabled = enabled
}

// GetStats 获取统计信息（可用于监控）
func (m *Middleware) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"resource":  m.config.ResourceName,
		"threshold": m.config.Threshold,
		"enabled":   m.enabled,
	}
}

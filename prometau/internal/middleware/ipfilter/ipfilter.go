// Package ipfilter 提供 IP 黑名单和白名单过滤中间件
// 支持精确 IP、CIDR 块和通配符匹配
package ipfilter

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
)

// Config 存储 IP 过滤配置
type Config struct {
	// Blacklist 黑名单配置
	Blacklist *IPListConfig `yaml:"blacklist"`
	// Whitelist 白名单配置
	Whitelist *IPListConfig `yaml:"whitelist"`
}

// IPListConfig 存储 IP 列表配置
type IPListConfig struct {
	// Enabled 是否启用该列表
	Enabled bool `yaml:"enabled"`
	// IPs IP 地址或 CIDR 块列表
	IPs []string `yaml:"ips"`
	// Paths 生效路径列表（空表示所有路径）
	Paths []string `yaml:"paths"`
}

// Middleware 实现 IP 过滤中间件
type Middleware struct {
	config       *Config
	blacklistIPs []*net.IPNet // 解析后的黑名单 IP 网络列表
	whitelistIPs []*net.IPNet // 解析后的白名单 IP 网络列表
	mu           sync.RWMutex // 读写锁，保护 IP 列表
}

// New 创建新的 IP 过滤中间件
// config: IP 过滤配置，如果为 nil 则使用默认空配置
// 返回：IP 过滤中间件实例
func New(config *Config) *Middleware {
	m := &Middleware{
		config:       config,
		blacklistIPs: make([]*net.IPNet, 0),
		whitelistIPs: make([]*net.IPNet, 0),
	}
	m.parseIPs()
	return m
}

// Name 实现 Middleware 接口
// 返回中间件名称 "ipfilter"
func (m *Middleware) Name() string {
	return "ipfilter"
}

// Handle 实现 Middleware 接口
// 处理 HTTP 请求，进行 IP 过滤
// next: 下一个处理器
// 返回：包装后的处理器
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取客户端 IP
		clientIP := m.getClientIP(r)

		// 优先检查黑名单（黑名单优先级更高）
		if m.config.Blacklist != nil && m.config.Blacklist.Enabled {
			if m.isIPInList(clientIP, m.blacklistIPs) {
				slog.Warn("IP 被黑名单拦截", "ip", clientIP, "path", r.URL.Path)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		// 检查白名单
		if m.config.Whitelist != nil && m.config.Whitelist.Enabled {
			if !m.isIPInList(clientIP, m.whitelistIPs) {
				slog.Warn("IP 不在白名单中", "ip", clientIP, "path", r.URL.Path)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// parseIPs 解析配置中的 IP 地址和 CIDR 块
func (m *Middleware) parseIPs() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config.Blacklist != nil {
		m.blacklistIPs = m.parseIPList(m.config.Blacklist.IPs)
	}

	if m.config.Whitelist != nil {
		m.whitelistIPs = m.parseIPList(m.config.Whitelist.IPs)
	}
}

// parseIPList 解析 IP 地址和 CIDR 块列表
// ips: IP 地址或 CIDR 块字符串数组
// 返回：解析后的 IP 网络指针切片
func (m *Middleware) parseIPList(ips []string) []*net.IPNet {
	result := make([]*net.IPNet, 0, len(ips))

	for _, ipStr := range ips {
		if strings.Contains(ipStr, "/") {
			// CIDR 块（如 192.168.1.0/24）
			_, ipNet, err := net.ParseCIDR(ipStr)
			if err != nil {
				slog.Error("解析 CIDR 失败", "cidr", ipStr, "error", err)
				continue
			}
			result = append(result, ipNet)
		} else {
			// 单个 IP 地址
			ip := net.ParseIP(ipStr)
			if ip == nil {
				slog.Error("解析 IP 失败", "ip", ipStr)
				continue
			}
			// 转换为 /32 CIDR 表示
			ipNet := &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(32, 32),
			}
			result = append(result, ipNet)
		}
	}

	return result
}

// isIPInList 检查 IP 是否在给定的 IP 列表中
// ip: 要检查的 IP 地址
// ipList: IP 网络列表
// 返回：是否在列表中
func (m *Middleware) isIPInList(ip net.IP, ipList []*net.IPNet) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ipNet := range ipList {
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// getClientIP 从 HTTP 请求中提取客户端 IP
// r: HTTP 请求
// 返回：客户端 IP 地址
func (m *Middleware) getClientIP(r *http.Request) net.IP {
	// 优先从 X-Forwarded-For 头获取（代理环境）
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := net.ParseIP(strings.TrimSpace(ips[0]))
			if ip != nil {
				return ip
			}
		}
	}

	// 从 X-Real-IP 头获取
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		ip := net.ParseIP(xri)
		if ip != nil {
			return ip
		}
	}

	// 回退到 RemoteAddr
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip != "" {
		return net.ParseIP(ip)
	}

	return nil
}

// Configure 实现 Configurable 接口
// 用于动态配置中间件（暂未实现）
func (m *Middleware) Configure(config map[string]interface{}) error {
	// TODO: 实现动态配置
	return nil
}

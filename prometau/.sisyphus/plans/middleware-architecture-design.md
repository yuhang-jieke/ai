# 中间件架构设计规划

## 设计原则

### 1. 执行顺序原则
```
安全相关 (CORS/黑名单) → 认证相关 (JWT) → 保护相关 (限流) → 业务逻辑
```

**理由**：
- 尽早拒绝非法请求，减少后续处理开销
- 认证在限流之前，避免未认证请求消耗限流配额
- CORS 最先处理，因为浏览器会先发预检请求

### 2. 配置驱动原则
所有中间件支持：
- YAML/JSON 配置文件
- 环境变量覆盖
- 动态热更新

### 3. 可插拔原则
- 每个中间件独立
- 可单独启用/禁用
- 支持自定义中间件注入

---

## 中间件详细设计

### 1. CORS 中间件

**位置**: 最外层

**配置**:
```yaml
middleware:
  cors:
    enabled: true
    allow_origins:
      - "http://localhost:3000"
      - "https://app.example.com"
    allow_methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
    allow_headers:
      - "Content-Type"
      - "Authorization"
    allow_credentials: true
    max_age: 86400
```

**实现要点**:
- 处理 OPTIONS 预检请求
- 设置 CORS 响应头
- 支持通配符和正则匹配

---

### 2. IP 黑白名单中间件

**位置**: CORS 之后，认证之前

**配置**:
```yaml
middleware:
  ip_filter:
    enabled: true
    # 黑名单优先，命中即拒绝
    blacklist:
      enabled: true
      ips:
        - "192.168.1.100"
        - "10.0.0.0/8"
      paths:
        - "/admin/*"
    
    # 白名单模式下，非白名单 IP 全部拒绝
    whitelist:
      enabled: false
      mode: "allow"  # allow: 仅允许白名单，deny: 仅拒绝白名单
      ips:
        - "127.0.0.1"
        - "10.10.10.0/24"
      paths:
        - "/internal/*"
```

**实现要点**:
- 支持 IP 段 (CIDR)
- 支持路径匹配
- 黑名单优先级高于白名单

---

### 3. JWT 认证中间件

**位置**: 黑白名单之后，限流之前

**配置**:
```yaml
middleware:
  jwt:
    enabled: true
    secret: "${JWT_SECRET}"
    signing_method: "HS256"
    expiration: 3600  # 1 小时
    
    #  exempts 路径（不需要认证）
    exempts:
      - "/health"
      - "/api/v1/public/*"
      - "/api/v1/auth/login"
    
    # 令牌传递方式
    token_lookup:
      - "header:Authorization"
      - "query:token"
      - "cookie:jwt_token"
    
    # 刷新令牌
    refresh:
      enabled: true
      expiration: 604800  # 7 天
```

**实现要点**:
- 支持多位置提取 Token
- 支持 Token 刷新
- 支持豁免路径
- 自动注入用户信息到 Context

---

### 4. 限流中间件 (Sentinel)

**位置**: 认证之后，业务之前

**配置**:
```yaml
middleware:
  ratelimit:
    enabled: true
    type: "sentinel"  # sentinel, redis, local
    
    sentinel:
      # 应用名称
      app_name: "prometau"
      # 控制台地址
      dashboard_addr: "localhost:8080"
      
    # 限流规则
    rules:
      # QPS 限流
      - resource: "/api/*"
        type: "qps"
        count: 100
        burst: 50
        
      # 并发数限流
      - resource: "/api/upload"
        type: "concurrent"
        count: 10
        
      # 用户维度限流
      - resource: "/api/search"
        type: "qps"
        count: 10
        scope: "user"  # user, ip, global
        
      # IP 维度限流
      - resource: "/api/*"
        type: "qps"
        count: 1000
        scope: "ip"
    
    # 降级策略
    fallback:
      enabled: true
      response:
        code: 429
        message: "Too many requests, please try again later"
```

**实现要点**:
- 多维度限流（全局/IP/用户）
- 多种限流算法（令牌桶/漏桶/滑动窗口）
- 支持热点参数限流
- 降级处理

---

## 代码架构

### 目录结构
```
internal/middleware/
├── middleware.go       # 中间件接口和链
├── cors/
│   └── cors.go         # CORS 实现
├── ipfilter/
│   └── ipfilter.go     # IP 黑白名单
├── auth/
│   └── jwt.go          # JWT 认证
├── ratelimit/
│   ├── ratelimit.go    # 限流接口
│   └── sentinel.go     # Sentinel 实现
└── recovery/
    └── recovery.go     #  panic 恢复
```

### 中间件接口
```go
type Middleware interface {
    Name() string
    Handle(next http.Handler) http.Handler
}

type Configurable interface {
    Configure(config map[string]interface{}) error
}

type Chain struct {
    middlewares []Middleware
}

func (c *Chain) Use(mw Middleware) *Chain
func (c *Chain) Build(handler http.Handler) http.Handler
```

### 使用示例
```go
// 创建中间件链
chain := middleware.NewChain()

// 按顺序添加中间件
chain.Use(
    cors.New(corsConfig),
    ipfilter.New(ipFilterConfig),
    auth.NewJWT(jwtConfig),
    ratelimit.New(sentinelConfig),
    recovery.New(),
)

// 应用中间件
handler := chain.Build(router)
```

---

## 性能优化

### 1. 缓存策略
| 中间件 | 缓存内容 | 过期时间 |
|--------|----------|----------|
| IP 过滤 | IP 匹配结果 | 5 分钟 |
| JWT | Token 解析结果 | Token 有效期 |
| 限流 | 计数状态 | 实时 |

### 2. 跳过高架
以下情况跳过中间件：
- 健康检查端点 (`/health`, `/ready`)
- 静态文件请求
- OPTIONS 预检请求

### 3. 异步日志
中间件日志异步写入，不阻塞请求处理。

---

## 监控指标

```prometheus
# 中间件指标
middleware_duration_seconds{middleware="cors"}
middleware_duration_seconds{middleware="jwt"}
middleware_duration_seconds{middleware="ratelimit"}

# 认证指标
jwt_auth_total{status="success"}
jwt_auth_total{status="invalid_token"}
jwt_auth_total{status="expired"}

# 限流指标
ratelimit_total{resource="/api/*", status="allowed"}
ratelimit_total{resource="/api/*", status="blocked"}

# IP 过滤指标
ipfilter_total{type="blacklist", action="denied"}
ipfilter_total{type="whitelist", action="denied"}
```

---

## 配置示例（完整版）

```yaml
middleware:
  # 全局开关
  enabled: true
  
  # 执行顺序
  order:
    - cors
    - ipfilter
    - jwt
    - ratelimit
    - recovery
  
  cors:
    enabled: true
    allow_origins: ["*"]
    allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_headers: ["Content-Type", "Authorization", "X-Request-ID"]
  
  ipfilter:
    enabled: true
    blacklist:
      ips: ["192.168.1.100", "10.0.0.0/8"]
    whitelist:
      enabled: false
      ips: []
  
  jwt:
    enabled: true
    secret: "${JWT_SECRET:your-secret-key}"
    exempts:
      - "/health"
      - "/api/v1/auth/*"
  
  ratelimit:
    enabled: true
    rules:
      - resource: "/api/*"
        qps: 100
        scope: "global"
      - resource: "/api/search"
        qps: 10
        scope: "user"
```

---

## 最佳实践

### 1. 开发环境
```yaml
middleware:
  jwt:
    exempts: ["/*"]  # 开发时跳过认证
  ratelimit:
    enabled: false   # 开发时关闭限流
```

### 2. 生产环境
```yaml
middleware:
  jwt:
    secret: "${VAULT_SECRET}"  # 从密钥管理系统获取
  ratelimit:
    sentinel:
      dashboard_addr: "sentinel.example.com"
```

### 3. 灰度发布
```yaml
middleware:
  ratelimit:
    rules:
      - resource: "/api/v2/*"
        qps: 10
        scope: "user"
        condition: "header[X-Canary] == true"
```

---

## 扩展方向

1. **动态配置** - 从 Nacos/Apollo 动态加载中间件配置
2. **插件系统** - 支持自定义中间件插件
3. **链路追踪** - 集成 OpenTelemetry
4. **统一错误处理** - 中间件错误统一格式
5. **A/B 测试** - 基于中间件条件的流量分流

# 计划: 中间件统一封装与调用

## 背景

项目已有多个中间件实现，但缺少统一调用接口。用户要求：
1. 将黑白名单、限流(sentinel)、身份认证(jwt)、CORS 封装成可调用方式
2. 限流改用 Alibaba Sentinel 库实现
3. 提供统一的调用接口

## 现有资产分析

| 中间件 | 位置 | 接口类型 | 状态 |
|--------|------|----------|------|
| IP黑白名单 | `internal/middleware/ipfilter/` | `middleware.Middleware` | ✅ 已实现 |
| CORS | `internal/middleware/cors/` | `middleware.Middleware` | ✅ 已实现 |
| JWT认证 | `internal/middleware/auth/` | `middleware.Middleware` | ✅ 已实现 |
| Recovery | `internal/middleware/recovery/` | `middleware.Middleware` | ✅ 已实现 |
| 限流 | `gospacex/core/protocol/http/middleware/` | 简单令牌桶 | ⚠️ 需改用sentinel |
| 工厂 | `internal/middleware/factory/` | 已有基础 | ✅ 可扩展 |

## 需要创建的组件

### 1. 中间件管理器 (`internal/middleware/manager/manager.go`)

**职责**：
- 统一管理所有中间件
- 提供链式调用API
- 支持动态添加/移除中间件
- 提供预设中间件链

**核心接口**：
```go
type Manager struct {
    chain      *middleware.Chain
    middleware map[MiddlewareType]middleware.Middleware
}

// 链式调用
manager := NewManager().
    WithRecovery(nil).
    WithCORS(nil).
    WithIPFilter(ipConfig).
    WithRateLimit(rateConfig).
    WithJWT(jwtConfig)

// 构建最终handler
handler := manager.Build(finalHandler)
```

### 2. Sentinel限流中间件 (`internal/middleware/ratelimit/ratelimit.go`)

**职责**：
- 基于 Alibaba Sentinel 实现限流
- 支持 QPS 限流
- 支持并发数限流
- 支持熔断降级

**配置结构**：
```go
type RateLimitConfig struct {
    ResourceName     string                    // 资源名称
    Threshold        int64                     // QPS阈值
    ControlBehavior  flow.ControlBehavior      // 控制行为
    RelationStrategy flow.RelationStrategy     // 关联策略
}
```

### 3. Gin适配器 (`internal/middleware/adapter/gin_adapter.go`)

**职责**：
- 将标准 `http.Handler` 中间件适配为 Gin 中间件
- 保持与 `httpserver.Context` 的兼容性

## 实现步骤

### Step 1: 创建限流中间件
- 文件: `internal/middleware/ratelimit/ratelimit.go`
- 使用 `github.com/alibaba/sentinel-golang`
- 实现 `middleware.Middleware` 接口

### Step 2: 创建中间件管理器
- 文件: `internal/middleware/manager/manager.go`
- 提供统一调用入口
- 实现预设链：DefaultChain, SecureChain, FullChain

### Step 3: 创建Gin适配器
- 文件: `internal/middleware/adapter/gin_adapter.go`
- 转换 `http.Handler` 中间件为 Gin 中间件

### Step 4: 更新工厂模式
- 扩展 `internal/middleware/factory/factory.go`
- 添加限流中间件创建方法
- 添加管理器创建方法

### Step 5: 更新路由配置
- 修改 `internal/router/setup.go`
- 使用新的中间件管理器

## Sentinel 集成注意事项

1. **初始化**: 需要在应用启动时调用 `sentinel.InitDefault()`
2. **规则加载**: 使用 `sentinel.LoadRules()` 加载限流规则
3. **资源定义**: 每个API端点可以作为独立资源
4. **性能**: Sentinel 有一定性能开销，需合理配置

## 兼容性设计

现有系统使用 `httpserver.Context` 抽象层，需要适配：

```
标准中间件链                    Gin中间件链
     │                              │
     ▼                              ▼
http.Handler ──────适配器──────► gin.HandlerFunc
     │                              │
     ▼                              ▼
middleware.Middleware          httpserver.MiddlewareFunc
```

## 使用示例

### 方式1: 使用预设链
```go
// 在 main.go 或路由设置中
manager := manager.FullChain()  // Recovery -> CORS -> RateLimit -> IPFilter -> JWT
handler := manager.Build(mux)
```

### 方式2: 自定义链
```go
manager := manager.NewManager().
    WithRecovery(recovery.DefaultConfig()).
    WithCORS(cors.DefaultConfig()).
    WithRateLimit(&ratelimit.RateLimitConfig{
        ResourceName: "api",
        Threshold:    1000,
    }).
    WithJWT(auth.DefaultConfig())
```

### 方式3: 单独使用某个中间件
```go
// 只使用IP黑白名单
ipFilter := ipfilter.New(ipConfig)
chain := middleware.NewChain().Use(ipFilter)
handler := chain.Build(finalHandler)
```

### 方式4: 在路由上应用
```go
// 类似现有的 jwtAuth 用法
apiGroup.POST("/products", productHandler.CreateProduct, adaptMiddleware(rateLimitMW))
```

## 测试要点

- [ ] Sentinel 限流是否生效
- [ ] 中间件链顺序是否正确
- [ ] 与 Gin 框架的兼容性
- [ ] 配置热更新是否正常
- [ ] 错误处理是否完善
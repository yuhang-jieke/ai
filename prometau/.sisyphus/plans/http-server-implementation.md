# HTTP 服务器多框架支持实现计划

## 概述
实现可切换的 HTTP 服务器架构，支持 Gin、Echo、Fiber、Chi、Iris、Hertz 框架。

## 架构设计

### 设计模式
1. **工厂模式** - 根据配置创建不同 HTTP 服务器实例
2. **适配器模式** - 统一 Handler 接口，框架可替换
3. **依赖注入** - Router/Handler 解耦

### 目录结构
```
internal/httpserver/
├── server.go           # 核心接口定义 (已创建)
├── gin_server.go       # Gin 实现
├── echo_server.go      # Echo 实现
├── fiber_server.go     # Fiber 实现
├── chi_server.go       # Chi 实现
├── iris_server.go      # Iris 实现
├── hertz_server.go     # Hertz 实现
└── response.go         # 统一响应处理
```

## 实现任务

### Phase 1: 基础设施
- [ ] **T1** 创建 `internal/httpserver/server.go` - 核心接口 (✅ 已完成)
- [ ] **T2** 创建 `internal/httpserver/response.go` - 统一响应处理
- [ ] **T3** 更新 `internal/config/config.go` - 添加 HTTP Server 配置

### Phase 2: Gin 实现 (默认)
- [ ] **T4** 创建 `internal/httpserver/gin_server.go` - Gin 适配器
- [ ] **T5** 创建 `internal/httpserver/gin_context.go` - Gin Context 适配器

### Phase 3: 其他框架实现
- [ ] **T6** 创建 `internal/httpserver/echo_server.go` - Echo 适配器
- [ ] **T7** 创建 `internal/httpserver/fiber_server.go` - Fiber 适配器
- [ ] **T8** 创建 `internal/httpserver/chi_server.go` - Chi 适配器

### Phase 4: 集成测试
- [ ] **T9** 创建 `internal/httpserver/server_test.go` - 工厂模式测试
- [ ] **T10** 更新 `cmd/prometau/main.go` - 集成 HTTP 服务器

### Phase 5: 示例 API
- [ ] **T11** 创建 `internal/api/product_handler.go` - 商品 API Handler
- [ ] **T12** 创建 `internal/router/setup.go` - 路由注册

## 配置示例

```yaml
server:
  type: "gin"  # gin, echo, fiber, chi, iris, hertz
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30
  write_timeout: 30
  mode: "release"  # debug, release, test
```

## 验收标准
1. ✅ 默认使用 Gin 启动
2. ✅ 可通过配置切换框架
3. ✅ Handler 与框架解耦
4. ✅ 统一的响应格式
5. ✅ 所有测试通过

## 依赖
- github.com/gin-gonic/gin (默认)
- github.com/labstack/echo/v4 (可选)
- github.com/gofiber/fiber/v2 (可选)
- github.com/go-chi/chi/v5 (可选)
- github.com/kataras/iris/v12 (可选)
- github.com/cloudwego/hertz (可选)
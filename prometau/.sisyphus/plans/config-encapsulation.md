# Work Plan: Viper + Nacos 配置封装

## Overview
封装 Viper + Nacos 配置管理模块，实现 Nacos 远程配置优先级高于本地配置，支持热加载。

## Requirements Summary
| Item | Decision |
|------|----------|
| 优先级 | Nacos > 命令行 > 环境变量 > 配置文件 > 默认值 |
| 热加载 | ✅ 需要监听 Nacos 配置变更 |
| 降级策略 | ❌ 本期不实现 |
| 测试策略 | TDD 测试驱动开发 |
| 配置模块 | MySQL, Redis, HTTP Server, 日志, Nacos |

## Scope Boundaries
**INCLUDE:**
- 配置读取、优先级合并
- 热加载支持（监听 Nacos 配置变更）
- 结构体绑定
- 本地配置文件示例
- 单元测试

**EXCLUDE:**
- Nacos 不可用时的降级策略
- 配置加密/解密
- 多环境配置切换（dev/test/prod）

---

## Dependencies
```bash
go get github.com/spf13/viper@v1.18.2
go get github.com/nacos-group/nacos-sdk-go/v2@v2.2.3
go get github.com/spf13/pflag@v1.0.5
```

---

## Implementation Tasks

### Phase 1: 基础设施 (Foundation)
- [ ] **T1.1** 安装依赖 (viper, nacos-sdk-go, pflag)
- [ ] **T1.2** 创建 `internal/config/config.go` - 定义配置结构体
  - Config (顶层)
  - ServerConfig
  - DatabaseConfig (MySQL)
  - RedisConfig
  - LogConfig
  - NacosConfig (含 Username/Password 认证字段)
- [ ] **T1.3** 创建 `internal/config/options.go` - 函数式选项模式
  - Option 接口
  - WithConfigFile()
  - WithNacos()
  - WithEnvPrefix()
- [ ] **T1.4** 创建 `configs/config.yaml` - 本地配置文件示例

### Phase 2: Viper 封装 + 测试 (TDD)
- [ ] **T2.1** 创建 `internal/config/manager.go` - 配置管理器结构体
  - Manager 结构体 (含 sync.RWMutex 保护并发访问)
  - NewManager() 构造函数
- [ ] **T2.2** 编写 `internal/config/manager_test.go` - 测试优先级
  - TestLoadLocalConfig - 测试本地配置加载
  - TestEnvPriority - 测试环境变量优先级
  - TestDefaultValues - 测试默认值
- [ ] **T2.3** 实现 Load() 方法
  - 加载本地配置文件
  - 绑定环境变量 (AutomaticEnv + SetEnvPrefix)
  - 设置默认值
- [ ] **T2.4** 实现 Get()/Bind() 方法 (加读锁)

### Phase 3: Nacos 集成 + 测试 (TDD)
- [ ] **T3.1** 创建 `internal/config/nacos.go` - Nacos 客户端封装
  - NacosClient 结构体
  - Connect() 连接方法 (支持用户名密码认证)
  - GetConfig() 获取配置
  - Close() 关闭连接
- [ ] **T3.2** 编写 `internal/config/nacos_test.go`
  - TestNacosConfigStruct - 测试配置结构
  - TestNacosConnect (使用 mock 或 skip.Short)
- [ ] **T3.3** 集成 Nacos 到 Manager.Load() (加写锁)
  - 连接 Nacos
  - 获取远程配置
  - MergeConfig 到 Viper (覆盖同名 key)
- [ ] **T3.4** 实现错误处理
  - Nacos 连接失败时记录日志 (slog)
  - 配置获取失败时的处理

### Phase 4: 热加载 + 测试 (TDD)
- [ ] **T4.1** 创建 `internal/config/watcher.go` - 配置监听器
  - Watcher 结构体 (含 callbacks 列表和互斥锁)
  - OnChange() 回调注册
  - triggerCallbacks() 触发回调
- [ ] **T4.2** 编写 `internal/config/watcher_test.go`
  - TestWatcherCallback - 测试回调触发
  - TestMultipleCallbacks - 测试多回调
- [ ] **T4.3** 实现 Watch() 方法
  - 监听 Nacos 配置变更
  - 触发回调函数 (在新 goroutine 中执行)
- [ ] **T4.4** 实现 Close() 方法
  - 停止监听
  - 关闭 Nacos 连接

### Phase 5: 集成验证
- [ ] **T5.1** 更新 `cmd/prometau/main.go` - 使用配置管理器
- [ ] **T5.2** 运行 `go test ./internal/config/...` 验证测试通过
- [ ] **T5.3** 运行 `go build ./cmd/prometau` 验证编译通过
- [ ] **T5.4** 手动验证配置加载流程

---

## File Structure
```
internal/config/
├── config.go        # 配置结构体定义
├── manager.go       # 配置管理器（统一入口）
├── nacos.go         # Nacos 客户端封装
├── options.go       # 函数式选项模式
├── watcher.go       # 配置变更监听器
├── manager_test.go  # 管理器测试
├── nacos_test.go    # Nacos 客户端测试
└── watcher_test.go  # 监听器测试

configs/
└── config.yaml      # 本地配置文件示例
```

---

## Verification Commands
```bash
# 运行测试
go test ./internal/config/... -v

# 编译验证
go build ./cmd/prometau

# 运行程序
./prometau
```

---

## Acceptance Criteria
1. ✅ 配置管理器可通过函数式选项初始化
2. ✅ 配置优先级正确：Nacos > 环境变量 > 配置文件 > 默认值
3. ✅ 支持 MySQL、Redis、Server、Log、Nacos 配置模块
4. ✅ 热加载功能正常工作
5. ✅ 所有单元测试通过
6. ✅ 主程序可正常加载配置

---

## Risk & Mitigation
| Risk | Mitigation |
|------|------------|
| Nacos 连接失败导致启动失败 | 记录错误日志，允许使用本地配置（本期不实现降级） |
| 配置格式不兼容 | 使用 YAML 格式，Viper 原生支持 |
| 热加载导致配置不一致 | 使用回调机制，让调用方决定如何处理 |

---

## Auto-Resolved Decisions
1. **配置文件格式**: YAML (Viper 原生支持，可读性好)
2. **Nacos 默认参数**: 
   - ServerAddr: `127.0.0.1:8848`
   - Namespace: `public`
   - DataID: `prometau.yaml`
   - Group: `DEFAULT_GROUP`
3. **环境变量前缀**: `PROMETAU_` (如 `PROMETAU_DATABASE_HOST`)
4. **日志**: 使用标准库 `slog` (Go 1.21+)

## Decisions Needed from User
- [ ] Nacos 服务器实际地址、Namespace、DataID、Group（用户将提供）
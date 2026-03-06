# 测试修复计划

## 问题描述
在 `TestDefaultValues` 测试中，期望的默认服务器主机是 `0.0.0.0`，但实际得到的是 `127.0.0.1`。

## 根本原因
`internal/config/config.go` 中的 `DefaultConfig()` 函数设置的默认值与测试期望不一致：

- **当前值**: `Server.Host = "127.0.0.1"`
- **测试期望**: `Server.Host = "0.0.0.0"`

## 修复方案
需要修改 `internal/config/config.go` 文件中的 `DefaultConfig()` 函数：

```
// Server 配置部分应该改为：
Server: ServerConfig{
    Host:         "0.0.0.0",  // 改为 0.0.0.0
    Port:         8080,
    ReadTimeout:  30,
    WriteTimeout: 30,
},
```

同时更新其他默认值以保持一致性：
- Database.Host: "127.0.0.1" (本地开发)
- Redis.Host: "127.0.0.1" (本地开发)  
- Nacos.ServerAddr: "127.0.0.1:8848" (本地开发)

## 执行步骤
1. 修改 `internal/config/config.go` 中的 `DefaultConfig()` 函数
2. 重新运行测试：`go test ./internal/config/... -v`
3. 验证所有测试通过

## 注意事项
- `0.0.0.0` 是更合适的服务器默认绑定地址，允许外部访问
- `127.0.0.1` 仅限本地回环，生产环境通常需要 `0.0.0.0`
- 其他服务（数据库、Redis、Nacos）保持 `127.0.0.1` 作为本地开发默认值
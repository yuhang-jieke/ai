# MySQL 8.0+ 认证问题修复计划

## 问题描述
```
Error 1524 (HY000): Plugin 'mysql_native_password' is not loaded
```

## 根本原因
MySQL 8.0+ 默认使用 `caching_sha2_password` 认证插件，而不是旧版的 `mysql_native_password`。

## 解决方案

### 修改文件: `internal/database/mysql.go`

**当前代码 (第22-31行)**:
```go
dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=false&allowCleartextPasswords=1",
    cfg.Username,
    cfg.Password,
    cfg.Host,
    cfg.Port,
    cfg.Database,
)
```

**修复后代码**:
```go
// MySQL 8.0+ uses caching_sha2_password by default
// Use minimal DSN to let driver auto-negotiate authentication
dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
    cfg.Username,
    cfg.Password,
    cfg.Host,
    cfg.Port,
    cfg.Database,
)
```

### 关键点
1. **移除所有认证相关参数** - `tls=false`, `allowCleartextPasswords=1` 等
2. **简化 DSN** - 只保留必要的 `parseTime=true`
3. **让驱动自动协商** - `go-sql-driver/mysql` 会自动处理 `caching_sha2_password`

## 执行步骤

1. 运行 `/start-work` 启动执行器
2. 执行器会自动修复代码
3. 重新编译并测试

## 验证命令
```bash
cd /c/Users/ZhuanZ/Desktop/prometau
go build -o prometau.exe ./cmd/prometau
./prometau.exe
```

预期输出:
```
MySQL connection established successfully!
Database: root@115.190.57.118:3306/ai
```
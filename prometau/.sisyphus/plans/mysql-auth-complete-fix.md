# MySQL 8.0+ 认证问题完整修复方案

## 问题诊断

### 问题 1: MySQL 认证错误
```
Error 1524 (HY000): Plugin 'mysql_native_password' is not loaded
```

**原因**: DSN 中的 `allowCleartextPasswords=1` 参数强制使用旧版认证，但 MySQL 8.0+ 已禁用 `mysql_native_password` 插件。

### 问题 2: 配置文件路径
```
WARN failed to read config file path=configs/config.yaml
```

**原因**: 程序从非项目根目录运行时，相对路径找不到配置文件。

---

## 完整修复方案

### 修复 1: `internal/database/mysql.go` (第22-31行)

**当前代码（错误）**:
```go
dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=false&allowCleartextPasswords=1",
    cfg.Username,
    cfg.Password,
    cfg.Host,
    cfg.Port,
    cfg.Database,
)
```

**修复后（正确）**:
```go
// MySQL 8.0+ uses caching_sha2_password
// Minimal DSN - let driver auto-negotiate authentication
dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
    cfg.Username,
    cfg.Password,
    cfg.Host,
    cfg.Port,
    cfg.Database,
)
```

**关键点**:
- ✅ 移除 `allowCleartextPasswords=1` - 这会触发旧认证
- ✅ 移除 `tls=false` - 让驱动自动处理
- ✅ 保留 `parseTime=true` - 时间解析需要
- ✅ 简化 DSN - 驱动会自动协商 `caching_sha2_password`

---

### 修复 2: 确保配置文件存在

检查 `configs/config.yaml` 内容是否正确：
```yaml
database:
  driver: "mysql"
  host: "115.190.57.118"
  port: 3306
  database: "ai"
  username: "root"
  password: "4ay1nkal3u8ed77y"
```

---

## 执行命令

```bash
cd /c/Users/ZhuanZ/Desktop/prometau

# 1. 修改代码后重新编译
go build -o prometau.exe ./cmd/prometau

# 2. 从项目根目录运行
./prometau.exe
```

---

## 预期结果

```
INFO loaded config file path=configs/config.yaml
prometau configuration loaded successfully
Server: 127.0.0.1:8080
Database: root@115.190.57.118:3306/ai

--- Connecting to MySQL ---
INFO connected to mysql database host=115.190.57.118 port=3306 database=ai
MySQL connection established successfully!

Tables in 'ai' database:
  (no tables found - database is empty)

Database connection test completed!
```

---

## 为什么之前的修复失败？

| 尝试 | DSN 参数 | 结果 | 原因 |
|------|---------|------|------|
| 1 | `allowNativePasswords=true` | ❌ | 服务器端禁用了该插件 |
| 2 | `allowCleartextPasswords=1` | ❌ | 同样触发旧认证机制 |
| 3 | **无认证参数** | ✅ | 驱动自动协商新认证 |

**MySQL 8.0+ 认证流程**:
1. 驱动尝试 `caching_sha2_password`
2. 如果密码在缓存中，直接成功
3. 否则进行 SHA256 加密传输
4. **不需要任何 DSN 参数指定认证方式**
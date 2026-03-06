# 计划: 对象存储封装

## 目标
实现统一的对象存储接口，支持 MinIO、Aliyun OSS、RustFS 三种存储后端，通过配置切换，使用同一个接口。

## 架构设计

### 目录结构
```
internal/storage/
├── storage.go        # 统一接口定义
├── minio.go          # MinIO 实现
├── oss.go            # Aliyun OSS 实现
├── rustfs.go         # RustFS 实现
└── factory.go        # 工厂方法
```

### 统一接口 (Storage)
```go
type Storage interface {
    PutObject(ctx, key, reader, size, contentType, metadata) error
    GetObject(ctx, key) (reader, info, error)
    DeleteObject(ctx, key) error
    ListObjects(ctx, prefix, marker, delimiter, maxKeys) (result, error)
    PresignedURL(ctx, key, expires) (url, error)
    Type() StorageType
    Close() error
}
```

## 配置格式
```yaml
storage:
  type: "minio"  # minio | oss | rustfs
  bucket: "my-bucket"
  
  minio:
    endpoint: "localhost:9000"
    access_key_id: "minioadmin"
    secret_access_key: "minioadmin"
    use_ssl: false
    
  oss:
    endpoint: "oss-cn-hangzhou.aliyuncs.com"
    access_key_id: "your-id"
    access_key_secret: "your-secret"
    
  rustfs:
    endpoint: "http://localhost:8000"
    token: "your-token"
```

## 实现步骤

### Step 1: 创建统一接口
- 文件: `internal/storage/storage.go`
- 定义 Storage 接口
- 定义配置结构体
- 定义 ObjectInfo、ListObjectsResult 等类型

### Step 2: 实现 MinIO 存储类
- 文件: `internal/storage/minio.go`
- 使用 `github.com/minio/minio-go/v7`
- 实现所有 Storage 接口方法

### Step 3: 实现 Aliyun OSS 存储类
- 文件: `internal/storage/oss.go`
- 使用 `github.com/aliyun/aliyun-oss-go-sdk/oss`
- 实现所有 Storage 接口方法

### Step 4: 实现 RustFS 存储类
- 文件: `internal/storage/rustfs.go`
- 使用 HTTP API 调用
- 实现所有 Storage 接口方法

### Step 5: 创建工厂方法
- 文件: `internal/storage/factory.go`
- New(cfg) 根据配置类型返回对应实现
- 存储类型切换只需改配置

### Step 6: 更新配置结构
- 文件: `internal/config/config.go`
- 添加 Storage 配置字段

## 使用示例

```go
// 创建存储客户端（根据配置自动选择）
storage, err := storage.New(&cfg.Storage)

// 上传文件 - 不论是 MinIO、OSS 还是 RustFS，接口相同
err := storage.PutObject(ctx, "file.txt", reader, size, "text/plain", nil)

// 下载文件
reader, info, err := storage.GetObject(ctx, "file.txt")

// 删除文件
err := storage.DeleteObject(ctx, "file.txt")

// 生成预签名URL
url, err := storage.PresignedURL(ctx, "file.txt", 3600)
```

## 切换存储方式
只需修改配置文件中的 `storage.type`：
- `"minio"` → 使用 MinIO
- `"oss"` → 使用阿里云 OSS
- `"rustfs"` → 使用 RustFS

## 依赖包
- `github.com/minio/minio-go/v7` ✅ 已安装
- `github.com/aliyun/aliyun-oss-go-sdk/oss` ✅ 已安装
- RustFS 使用标准 HTTP 客户端
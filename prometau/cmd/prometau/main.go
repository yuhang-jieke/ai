package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"syscall"
	"time"

	"github.com/yuhang-jieke/ai/internal/config"
	"github.com/yuhang-jieke/ai/internal/database"
	"github.com/yuhang-jieke/ai/internal/httpserver"
	"github.com/yuhang-jieke/ai/internal/model"
	"github.com/yuhang-jieke/ai/internal/redis"
	"github.com/yuhang-jieke/ai/internal/repository"
	"github.com/yuhang-jieke/ai/internal/router"
	"github.com/yuhang-jieke/ai/internal/storage"
)

func main() {
	// ============ 1. 加载配置 ============
	configPath, err := findConfigFile()
	if err != nil {
		fmt.Println("========================================")
		fmt.Println("ERROR: Configuration file not found!")
		fmt.Println("========================================")
		fmt.Println(err)
		os.Exit(1)
	}

	slog.Info("found config file", "path", configPath)

	mgr := config.NewManager(config.WithConfigFile(configPath))
	if err := mgr.Load(); err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	cfg, err := mgr.Get()
	if err != nil {
		slog.Error("failed to get configuration", "error", err)
		os.Exit(1)
	}

	printConfig(configPath, cfg)

	// ============ 2. 初始化数据库 ============
	// 方式1: 使用配置结构体（原有方式）
	//gormDB, err := database.NewGormMySQL(cfg.Database)

	// 方式2: 使用DSN字符串（线上部署推荐）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/ai?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
	)
	gormDB, err := database.NewGormMySQLWithDSN(dsn, nil)

	// 方式3: 使用选项结构体
	// opt := &database.DatabaseOption{
	// 	Host:     "localhost",
	// 	Port:     3306,
	// 	Username: "root",
	// 	Password: "password",
	// 	Database: "mydb",
	// }
	// gormDB, err := database.NewGormMySQLWithOption(opt)

	if err != nil {
		slog.Error("failed to connect to mysql", "error", err)
		os.Exit(1)
	}
	defer gormDB.Close()

	slog.Info("MySQL connection established")

	// ============ 3. 数据库迁移 ============
	// 禁用 GORM AutoMigrate，使用手动迁移（migration.go）
	/*if err := gormDB.Migrate(); err != nil {
		slog.Error("数据库迁移失败", "error", err)
		os.Exit(1)
	}*/

	// 方式2: 只迁移特定表
	err = gormDB.Migrate(&model.User{}, &model.Product{})
	if err != nil {
		panic("数据表迁移失败")
	}

	// 获取底层 SQL DB
	sqlDB, err := gormDB.GetSQLDB()
	if err != nil {
		slog.Error("failed to get sql.DB", "error", err)
		os.Exit(1)
	}

	// 迁移 users 表
	/*if err := database.CheckAndCreateUsersTable(sqlDB); err != nil {
		slog.Error("failed to run user migrations", "error", err)
		os.Exit(1)
	}*/

	// 迁移 products 表（原有逻辑保留）
	/*if err := database.CheckAndCreateProductsTable(sqlDB); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}*/

	// ============ 4. 初始化 Redis ============
	if err := redis.Init(&cfg.Redis); err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close()

	slog.Info("Redis connection established")

	// ============ 5. 初始化对象存储 ============
	// 根据配置自动选择：minio / oss / rustfs
	// 使用默认值作为后备（防止 Nacos 配置覆盖后为空）
	storageDefaults := storage.DefaultConfig()
	var store storage.Storage
	storeCfg := &storage.Config{
		Type: storage.StorageType(cfg.Storage.Type),
		MinIO: storage.MinIOConfig{
			Endpoint:        getOrDefault(cfg.Storage.MinIO.Endpoint, storageDefaults.MinIO.Endpoint),
			AccessKeyID:     getOrDefault(cfg.Storage.MinIO.AccessKeyID, storageDefaults.MinIO.AccessKeyID),
			SecretAccessKey: getOrDefault(cfg.Storage.MinIO.SecretAccessKey, storageDefaults.MinIO.SecretAccessKey),
			UseSSL:          cfg.Storage.MinIO.UseSSL,
			Bucket:          getOrDefault(cfg.Storage.MinIO.Bucket, storageDefaults.MinIO.Bucket),
			Region:          getOrDefault(cfg.Storage.MinIO.Region, storageDefaults.MinIO.Region),
		},
		OSS: storage.OSSConfig{
			Endpoint:        getOrDefault(cfg.Storage.OSS.Endpoint, storageDefaults.OSS.Endpoint),
			AccessKeyID:     getOrDefault(cfg.Storage.OSS.AccessKeyID, storageDefaults.OSS.AccessKeyID),
			AccessKeySecret: getOrDefault(cfg.Storage.OSS.AccessKeySecret, storageDefaults.OSS.AccessKeySecret),
			Bucket:          getOrDefault(cfg.Storage.OSS.Bucket, storageDefaults.OSS.Bucket),
			Region:          getOrDefault(cfg.Storage.OSS.Region, storageDefaults.OSS.Region),
		},
		RustFS: storage.RustFSConfig{
			Endpoint: getOrDefault(cfg.Storage.RustFS.Endpoint, storageDefaults.RustFS.Endpoint),
			Token:    getOrDefault(cfg.Storage.RustFS.Token, storageDefaults.RustFS.Token),
			Bucket:   getOrDefault(cfg.Storage.RustFS.Bucket, storageDefaults.RustFS.Bucket),
		},
	}

	store, err = storage.New(storeCfg)
	if err != nil {
		slog.Warn("对象存储初始化失败，存储API将不可用", "error", err)
		// 不阻止应用启动，存储API将被禁用
	}
	if store != nil {
		defer store.Close()
		slog.Info("对象存储初始化成功", "type", store.Type())
	}

	// ============ 6. 创建 HTTP 服务器 ============
	serverConfig := httpserver.LoadHTTPServerConfig(cfg)
	factory := httpserver.NewServerFactory(serverConfig)

	server, err := factory.Create()
	if err != nil {
		slog.Error("failed to create HTTP server", "error", err)
		os.Exit(1)
	}

	// ============ 6. 配置路由 ============
	setupRoutes(server, sqlDB, gormDB, store)

	// ============ 7. 启动服务器 ============
	go func() {
		if err := server.Start(); err != nil {
			slog.Error("failed to start server", "error", err)
		}
	}()

	printStartupInfo(serverConfig)

	// ============ 8. 优雅关闭 ============
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(serverConfig.ShutdownTimeout)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("Server exited properly")
}

// findConfigFile searches for config file in multiple locations.
func findConfigFile() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	execDir := filepath.Dir(execPath)

	configPaths := []string{
		"configs/config.yaml",
		filepath.Join(execDir, "configs", "config.yaml"),
		filepath.Join(execDir, "..", "configs", "config.yaml"),
	}

	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("config file not found in any of: %v", configPaths)
}

// printConfig prints the application configuration.
func printConfig(path string, cfg *config.Config) {
	fmt.Println("========================================")
	fmt.Println("       PROMETAU CONFIGURATION")
	fmt.Println("========================================")
	fmt.Printf("Config File: %s\n", path)
	fmt.Printf("HTTP Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Database: %s@%s:%d/%s\n", cfg.Database.Username, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)
	fmt.Println("========================================\n")
}

// printStartupInfo 打印启动信息
func printStartupInfo(cfg *httpserver.ServerConfig) {
	fmt.Println("\n========================================")
	fmt.Println("       🚀 SERVER RUNNING")
	fmt.Println("========================================")
	fmt.Printf("API Base URL: http://%s:%d/api\n", cfg.Host, cfg.Port)
	fmt.Println("Endpoints:")
	fmt.Println("  POST   /api/auth/login   - 用户登录")
	fmt.Println("  GET    /api/products     - 获取产品列表")
	fmt.Println("  POST   /api/products     - 创建产品 (需认证)")
	fmt.Println("  PUT    /api/products/:id - 更新产品 (需认证)")
	fmt.Println("  DELETE /api/products/:id - 删除产品 (需认证)")
	fmt.Println("========================================\n")
}

// getOrDefault 返回 value，如果为空则返回 defaultValue
func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// setupRoutes configures routes based on server type.
func setupRoutes(server httpserver.Server, db *sql.DB, gormDB *database.GormMySQL, store storage.Storage) {
	productRepo := repository.NewProductRepository(db)
	userRepo := repository.NewUserRepository(gormDB.DB)

	// For Gin server - use type assertion
	if ginServer, ok := server.(*httpserver.GinServer); ok {
		engine := ginServer.GetEngine()
		r := &httpserver.GinRouteRegister{
			Engine:      engine,
			RouterGroup: &engine.RouterGroup,
		}
		router.SetupRouter(r, productRepo, userRepo, store)
	}
}

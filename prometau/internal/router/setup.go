package router

import (
	"fmt"
	"log/slog"
	"strings"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/yuhang-jieke/ai/internal/api"
	"github.com/yuhang-jieke/ai/internal/httpserver"
	"github.com/yuhang-jieke/ai/internal/middleware/adapter"
	"github.com/yuhang-jieke/ai/internal/middleware/auth"
	"github.com/yuhang-jieke/ai/internal/middleware/blacklist"
	"github.com/yuhang-jieke/ai/internal/middleware/ipfilter"
	"github.com/yuhang-jieke/ai/internal/middleware/ratelimit"
	"github.com/yuhang-jieke/ai/internal/repository"
	"github.com/yuhang-jieke/ai/internal/storage"
)

// 初始化标志
var sentinelInitialized bool

// initSentinel 初始化Sentinel（只需执行一次）
func initSentinel() {
	if sentinelInitialized {
		return
	}

	// 初始化Sentinel
	err := sentinel.InitDefault()
	if err != nil {
		slog.Error("Sentinel初始化失败", "error", err)
	} else {
		slog.Info("Sentinel初始化成功")
		sentinelInitialized = true
	}
}

// ============ JWT 认证中间件 ============

// jwtAuth JWT 认证中间件
// 从 Authorization Header 提取并验证 Token
func jwtAuth(ctx httpserver.Context) error {
	// 获取 gin.Context（使用新的 Gin() 方法）
	ginCtx := ctx.Gin().(*gin.Context)

	// 获取 Authorization Header
	authHeader := ginCtx.GetHeader("Authorization")
	if authHeader == "" {
		return ctx.Error(401, "缺少 Authorization 头")
	}

	// 解析 Bearer Token
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return ctx.Error(401, "无效的 Authorization 格式")
	}

	// 验证 Token（这里使用简化验证，生产环境请使用完整 JWT 验证）
	claims, err := validateJWT(tokenString)
	if err != nil {
		return ctx.Error(401, "无效的 Token: "+err.Error())
	}

	// 将用户信息存入上下文，供后续处理器使用
	ctx.Set("user_id", claims["user_id"])
	ctx.Set("user_name", claims["user_name"])

	return ctx.Next()
}

// validateJWT 验证 JWT Token（完整实现版本）
// 现在使用 github.com/golang-jwt/jwt 进行完整验证
func validateJWT(tokenString string) (map[string]interface{}, error) {
	// 使用 golang-jwt 库验证 Token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法是否符合预期
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		// 固定秘钥，实际生产环境中应当从配置中读取
		return []byte("your-secret-key"), nil // 注意：生产环境应从配置或环境变量获取
	})

	if err != nil {
		return nil, err
	}

	// 检查 Token 是否有效
	if !token.Valid {
		return nil, fmt.Errorf("无效的 Token")
	}

	// 提取 Claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("无法解析 Claims")
	}

	return claims, nil
}

// ============ 中间件工厂函数 ============

// NewRateLimitMiddleware 创建限流中间件（基于Sentinel）
func NewRateLimitMiddleware(resourceName string, threshold float64) httpserver.MiddlewareFunc {
	initSentinel()

	config := &ratelimit.Config{
		Enabled:         true,
		ResourceName:    resourceName,
		Threshold:       threshold,
		ControlBehavior: flow.Reject,
	}

	mw := ratelimit.New(config)
	return adapter.AdaptMiddleware(mw)
}

// NewIPFilterMiddleware 创建IP黑白名单中间件
func NewIPFilterMiddleware(blacklist, whitelist []string) httpserver.MiddlewareFunc {
	config := &ipfilter.Config{}

	if len(blacklist) > 0 {
		config.Blacklist = &ipfilter.IPListConfig{
			Enabled: true,
			IPs:     blacklist,
		}
	}

	if len(whitelist) > 0 {
		config.Whitelist = &ipfilter.IPListConfig{
			Enabled: true,
			IPs:     whitelist,
		}
	}

	mw := ipfilter.New(config)
	return adapter.AdaptMiddleware(mw)
}

// NewJWTMiddleware 创建JWT认证中间件
func NewJWTMiddleware(secret string, exempts []string) httpserver.MiddlewareFunc {
	config := &auth.Config{
		Secret:  secret,
		Exempts: exempts,
	}

	mw := auth.New(config)
	return adapter.AdaptMiddleware(mw)
}

// NewBlacklistMiddleware 创建用户黑名单中间件（基于Redis）
// jwtSecret: JWT签名密钥，用于解析Token获取用户ID
// errorMessage: 自定义错误消息
func NewBlacklistMiddleware(jwtSecret, errorMessage string) httpserver.MiddlewareFunc {
	config := &blacklist.Config{
		Enabled:      true,
		JWTSecret:    jwtSecret,
		ErrorMessage: errorMessage,
	}
	return blacklist.HTTPServerMiddleware(config)
}

// ============ CORS 中间件 ============

// corsMiddleware 独立的 CORS 中间件函数
func corsMiddleware(ctx httpserver.Context) error {
	ginCtx := ctx.Gin().(*gin.Context)

	ginCtx.Header("Access-Control-Allow-Origin", "*")
	ginCtx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	ginCtx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if ginCtx.Request.Method == "OPTIONS" {
		ginCtx.Status(204)
		return nil
	}

	return ctx.Next()
}

// ============ 路由设置 ============

// SetupRouter 配置所有应用路由
// r: 路由注册器
// productRepo: 产品仓库
// userRepo: 用户仓库
// store: 对象存储实例（可选，传nil则不启用存储API）
func SetupRouter(r httpserver.RouteRegister, productRepo *repository.ProductRepository, userRepo *repository.UserRepository, store storage.Storage) {
	// 初始化Sentinel
	initSentinel()

	// 创建处理器
	productHandler := api.NewProductHandler(productRepo)
	userHandler := api.NewUserHandler(userRepo)

	// ============ 创建中间件实例 ============
	// 限流中间件：每秒最多100个请求
	rateLimitMW := NewRateLimitMiddleware("api-products", 100)

	// 用户黑名单中间件：从Token解析用户ID并检查Redis黑名单
	blacklistMW := NewBlacklistMiddleware(
		"your-secret-key", // JWT密钥（需与登录时的密钥一致）
		"该用户被加入黑名单，请联系管理员", // 自定义错误消息
	)

	// IP黑白名单中间件（可选配置）
	// ipFilterMW := NewIPFilterMiddleware(
	// 	[]string{"192.168.1.100"}, // 黑名单
	// 	[]string{},                  // 白名单（为空表示不启用白名单）
	// )

	// Health check endpoint - 不需要认证
	r.GET("/health", func(ctx httpserver.Context) error {
		return ctx.Success(map[string]string{
			"status": "ok",
		})
	})

	// API v1 路由组
	apiGroup := r.Group("/api")

	// 路由组级别中间件（可选）
	// apiGroup.Use(corsMiddleware)

	{
		// ============ 认证接口 - 无需认证 ============
		auth := apiGroup.Group("/auth")
		{
			auth.POST("/login", userHandler.Login)
		}

		// ============ 公开接口 - 不需要认证 ============
		apiGroup.GET("/products", productHandler.GetAllProducts)
		apiGroup.GET("/products/search", productHandler.SearchProducts)
		apiGroup.GET("/products/:id", productHandler.GetProduct)

		// ============ 需要认证的接口 ============
		// 中间件执行顺序：限流 -> 黑名单检查 -> JWT认证 -> 处理器
		// 1. rateLimitMW: 限流保护
		// 2. blacklistMW: 检查用户是否在Redis黑名单中
		// 3. jwtAuth: JWT身份验证
		apiGroup.POST("/products", productHandler.CreateProduct, rateLimitMW, blacklistMW, jwtAuth)
		apiGroup.PUT("/products/:id", productHandler.UpdateProduct, rateLimitMW, blacklistMW, jwtAuth)
		apiGroup.DELETE("/products/:id", productHandler.DeleteProduct, rateLimitMW, blacklistMW, jwtAuth)

		// ============ 文件存储接口 - 使用统一接口 ============
		// 切换配置后（minio/oss/rustfs），代码无需任何修改
		if store != nil {
			storageHandler := api.NewStorageHandler(store)
			storageGroup := apiGroup.Group("/storage")

			// 公开接口
			storageGroup.GET("/info", storageHandler.GetStorageInfo)     // 获取存储类型信息
			storageGroup.GET("/list", storageHandler.ListFiles)          // 列出文件
			storageGroup.GET("/download", storageHandler.DownloadFile)   // 下载文件
			storageGroup.GET("/presign", storageHandler.GetPresignedURL) // 获取预签名URL
			storageGroup.GET("/buckets", storageHandler.ListBuckets)     // 列出存储桶

			// 需要认证的接口
			storageGroup.POST("/upload", storageHandler.UploadFile, rateLimitMW, jwtAuth)   // 上传文件
			storageGroup.DELETE("/delete", storageHandler.DeleteFile, rateLimitMW, jwtAuth) // 删除文件
			storageGroup.POST("/copy", storageHandler.CopyFile, rateLimitMW, jwtAuth)       // 复制文件
			storageGroup.GET("/file/info", storageHandler.GetFileInfo)                      // 获取文件信息
		}
	}
}

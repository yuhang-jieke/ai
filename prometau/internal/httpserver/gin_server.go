package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/yuhang-jieke/ai/internal/config"
)

// GinServer implements Server interface using Gin framework.
type GinServer struct {
	config *ServerConfig
	engine *gin.Engine
	server *http.Server
}

// GetEngine returns the underlying gin.Engine.
func (s *GinServer) GetEngine() *gin.Engine {
	return s.engine
}

// NewGinServer creates a new Gin HTTP server.
func NewGinServer(cfg *ServerConfig) (Server, error) {
	// Set Gin mode
	switch cfg.Mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// Add recovery middleware
	engine.Use(gin.Recovery())

	// Add logger middleware
	engine.Use(gin.Logger())

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	s := &GinServer{
		config: cfg,
		engine: engine,
		server: &http.Server{
			Addr:         addr,
			Handler:      engine,
			ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
		},
	}

	return s, nil
}

// Start starts the Gin HTTP server.
func (s *GinServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	fmt.Printf("🚀 Starting Gin server on http://%s\n", addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the Gin server.
func (s *GinServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// RegisterRoutes registers API routes with framework-agnostic handlers.
func (s *GinServer) RegisterRoutes(r RouteRegister) {
	// Convert RouteRegister to Gin routes
	s.registerRoutes(r)
}

// registerRoutes converts framework-agnostic routes to Gin routes.
func (s *GinServer) registerRoutes(router RouteRegister) {
	// This will be called from main.go with actual route definitions
	// Example:
	// router.GET("/health", healthHandler)
	// router.POST("/products", createProductHandler)
}

// GinContext wraps gin.Context to implement Context interface.
type GinContext struct {
	c *gin.Context
}

// Request returns the HTTP request.
func (c *GinContext) Request() interface{} {
	return c.c.Request
}

// Gin returns the gin context.
// 使用此方法在中间件中获取 gin.Context
func (c *GinContext) Gin() interface{} {
	return c.c
}

// Response returns the HTTP response writer.
func (c *GinContext) Response() interface{} {
	return c.c.Writer
}

// Param returns route parameter by name.
func (c *GinContext) Param(name string) string {
	return c.c.Param(name)
}

// Query returns query parameter by name.
func (c *GinContext) Query(name string) string {
	return c.c.Query(name)
}

// Bind binds request body to struct.
func (c *GinContext) Bind(obj interface{}) error {
	return c.c.ShouldBind(obj)
}

// ShouldBind binds request body with validation and flexible type conversion
// only for authentication endpoints that need it.
func (c *GinContext) ShouldBind(obj interface{}) error {
	contentType := c.c.ContentType()

	// For application/json content
	if contentType == "application/json" {
		// Check if this is an authentication endpoint that needs flexible type conversion
		path := c.c.Request.URL.Path
		if path == "/api/auth/login" || path == "/api/auth/register" {
			// Use flexible type conversion for auth endpoints (allows numbers as strings)
			bodyBytes, err := io.ReadAll(c.c.Request.Body)
			if err != nil {
				return err
			}

			// Restore body for subsequent operations
			c.c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			// Parse JSON manually
			var jsonData interface{}
			if err := json.Unmarshal(bodyBytes, &jsonData); err != nil {
				return err
			}

			// Convert types for string fields (password, account, etc.)
			jsonDataConverted, err := convertTypesForStringFields(jsonData)
			if err != nil {
				return err
			}

			// Marshal converted data back to JSON
			convertedBytes, err := json.Marshal(jsonDataConverted)
			if err != nil {
				return err
			}

			// Deserialize to target struct
			return json.Unmarshal(convertedBytes, obj)
		}
		// For other endpoints, use standard JSON binding (preserves numeric types)
		return c.c.ShouldBindJSON(obj)
	} else if contentType == "application/x-www-form-urlencoded" {
		return c.c.ShouldBindWith(obj, binding.Form)
	} else if contentType == "multipart/form-data" {
		return c.c.ShouldBindWith(obj, binding.FormMultipart)
	}
	// Default behavior
	return c.c.ShouldBind(obj)
}

// convertTypesForStringFields converts numeric values to strings
// only for authentication endpoints where password/account might be sent as numbers
func convertTypesForStringFields(data interface{}) (interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v2 := range v {
			converted, err := convertTypesForStringFields(v2)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, v2 := range v {
			converted, err := convertTypesForStringFields(v2)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	case float64, int, int64:
		// Convert numbers to strings for auth fields
		return fmt.Sprintf("%v", v), nil
	case string:
		return v, nil
	case bool:
		return strconv.FormatBool(v), nil
	case nil:
		return v, nil
	default:
		return v, nil
	}
}

// JSON sends JSON response.
func (c *GinContext) JSON(code int, data interface{}) error {
	c.c.JSON(code, data)
	return nil
}

// Success sends success response.
func (c *GinContext) Success(data interface{}) error {
	c.c.JSON(http.StatusOK, NewSuccessResponse(data))
	return nil
}

// Error sends error response.
// 返回 error 以便中间件可以检测到错误并终止请求
func (c *GinContext) Error(code int, message string) error {
	c.c.JSON(code, NewErrorResponse(code, message))
	c.c.Abort() // 终止后续处理
	return fmt.Errorf("error: %s", message)
}

// Set sets context value.
func (c *GinContext) Set(key string, value interface{}) {
	c.c.Set(key, value)
}

// Get gets context value.
func (c *GinContext) Get(key string) interface{} {
	v, exists := c.c.Get(key)
	if !exists {
		return nil
	}
	return v
}

// Next calls next middleware.
func (c *GinContext) Next() error {
	c.c.Next()
	return nil
}

// GinRouteRegister implements RouteRegister for Gin.
type GinRouteRegister struct {
	Engine      *gin.Engine
	RouterGroup *gin.RouterGroup
}

// GET registers a GET route.
func (r *GinRouteRegister) GET(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	handlers := append(convertMiddlewares(middleware), convertHandler(handler))
	r.RouterGroup.GET(path, handlers...)
}

// POST registers a POST route.
func (r *GinRouteRegister) POST(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	handlers := append(convertMiddlewares(middleware), convertHandler(handler))
	r.RouterGroup.POST(path, handlers...)
}

// PUT registers a PUT route.
func (r *GinRouteRegister) PUT(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	handlers := append(convertMiddlewares(middleware), convertHandler(handler))
	r.RouterGroup.PUT(path, handlers...)
}

// DELETE registers a DELETE route.
func (r *GinRouteRegister) DELETE(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	handlers := append(convertMiddlewares(middleware), convertHandler(handler))
	r.RouterGroup.DELETE(path, handlers...)
}

// PATCH registers a PATCH route.
func (r *GinRouteRegister) PATCH(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	handlers := append(convertMiddlewares(middleware), convertHandler(handler))
	r.RouterGroup.PATCH(path, handlers...)
}

// Group creates a new route group.
func (r *GinRouteRegister) Group(path string) RouteGroup {
	ginGroup := r.RouterGroup.Group(path)
	return &GinRouteRegister{Engine: r.Engine, RouterGroup: ginGroup}
}

// Use adds middleware to the group.
func (r *GinRouteRegister) Use(middleware ...MiddlewareFunc) {
	r.RouterGroup.Use(convertMiddlewares(middleware)...)
}

// convertHandler converts framework-agnostic handler to Gin handler.
func convertHandler(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ginCtx := &GinContext{c: c}
		if err := h(ginCtx); err != nil {
			// 如果请求已被中止（Error() 方法会调用 Abort()），则不再重复输出
			if c.IsAborted() {
				return
			}
			// 否则发送错误响应
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}
}

// convertMiddlewares converts framework-agnostic middlewares to Gin middlewares.
func convertMiddlewares(mw []MiddlewareFunc) []gin.HandlerFunc {
	result := make([]gin.HandlerFunc, 0, len(mw))
	for _, m := range mw {
		result = append(result, convertMiddleware(m))
	}
	return result
}

// convertMiddleware converts framework-agnostic middleware to Gin middleware.
func convertMiddleware(mw MiddlewareFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ginCtx := &GinContext{c: c}
		if err := mw(ginCtx); err != nil {
			// 如果请求已被中止（Error() 方法会调用 Abort()），则不再处理
			if c.IsAborted() {
				return
			}
			// 否则发送错误响应
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		// 如果未被中止，继续执行下一个处理器
		if !c.IsAborted() {
			c.Next()
		}
	}
}

// LoadHTTPServerConfig loads HTTP server config from application config.
func LoadHTTPServerConfig(cfg *config.Config) *ServerConfig {
	return &ServerConfig{
		Type:            ServerTypeGin, // Default to Gin
		Host:            cfg.Server.Host,
		Port:            cfg.Server.Port,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		ShutdownTimeout: 10,
		Mode:            "release",
	}
}

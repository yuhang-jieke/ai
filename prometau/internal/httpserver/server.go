package httpserver

import (
	"context"
	"fmt"
)

// ServerType represents the type of HTTP server.
type ServerType string

const (
	ServerTypeGin   ServerType = "gin"
	ServerTypeEcho  ServerType = "echo"
	ServerTypeFiber ServerType = "fiber"
	ServerTypeChi   ServerType = "chi"
	ServerTypeIris  ServerType = "iris"
	ServerTypeHertz ServerType = "hertz"
)

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Type            ServerType `mapstructure:"type"`
	Host            string     `mapstructure:"host"`
	Port            int        `mapstructure:"port"`
	ReadTimeout     int        `mapstructure:"read_timeout"`
	WriteTimeout    int        `mapstructure:"write_timeout"`
	ShutdownTimeout int        `mapstructure:"shutdown_timeout"`
	Mode            string     `mapstructure:"mode"` // debug, release, test
}

// DefaultServerConfig returns default server configuration.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Type:            ServerTypeGin,
		Host:            "0.0.0.0",
		Port:            8080,
		ReadTimeout:     30,
		WriteTimeout:    30,
		ShutdownTimeout: 10,
		Mode:            "release",
	}
}

// Server defines the interface for HTTP servers.
type Server interface {
	// Start starts the HTTP server
	Start() error

	// Shutdown gracefully shuts down the server
	Shutdown(ctx context.Context) error

	// RegisterRoutes registers API routes
	RegisterRoutes(r RouteRegister)
}

// RouteRegister defines the interface for route registration.
// This allows handlers to be framework-agnostic.
type RouteRegister interface {
	// GET registers a GET route
	GET(path string, handler HandlerFunc, middleware ...MiddlewareFunc)

	// POST registers a POST route
	POST(path string, handler HandlerFunc, middleware ...MiddlewareFunc)

	// PUT registers a PUT route
	PUT(path string, handler HandlerFunc, middleware ...MiddlewareFunc)

	// DELETE registers a DELETE route
	DELETE(path string, handler HandlerFunc, middleware ...MiddlewareFunc)

	// PATCH registers a PATCH route
	PATCH(path string, handler HandlerFunc, middleware ...MiddlewareFunc)

	// Group creates a new route group
	Group(path string) RouteGroup
}

// RouteGroup represents a group of routes.
type RouteGroup interface {
	GET(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	POST(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	PUT(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	DELETE(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	PATCH(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	Group(path string) RouteGroup
	Use(middleware ...MiddlewareFunc)
}

// HandlerFunc is the framework-agnostic handler function signature.
type HandlerFunc func(ctx Context) error

// MiddlewareFunc is the framework-agnostic middleware function signature.
type MiddlewareFunc func(ctx Context) error

// Context defines the interface for HTTP context.
// This abstracts away framework-specific context implementations.
type Context interface {
	// Request returns the HTTP request
	Request() interface{}

	// Response returns the HTTP response writer
	Response() interface{}

	// Gin returns the gin context (framework-specific)
	// 使用此方法获取 gin.Context 进行框架特定操作
	Gin() interface{}

	// Param returns route parameter by name
	Param(name string) string

	// Query returns query parameter by name
	Query(name string) string

	// Bind binds request body to struct
	Bind(obj interface{}) error

	// ShouldBind binds request body with validation
	ShouldBind(obj interface{}) error

	// JSON sends JSON response
	JSON(code int, data interface{}) error

	// Success sends success response
	Success(data interface{}) error

	// Error sends error response
	Error(code int, message string) error

	// Set sets context value
	Set(key string, value interface{})

	// Get gets context value
	Get(key string) interface{}

	// Next calls next middleware
	Next() error
}

// Response represents a standard API response.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewSuccessResponse creates a success response.
func NewSuccessResponse(data interface{}) *Response {
	return &Response{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// NewErrorResponse creates an error response.
func NewErrorResponse(code int, message string) *Response {
	return &Response{
		Code:    code,
		Message: message,
	}
}

// ServerFactory creates HTTP server instances.
type ServerFactory struct {
	config *ServerConfig
}

// NewServerFactory creates a new server factory.
func NewServerFactory(config *ServerConfig) *ServerFactory {
	if config == nil {
		config = DefaultServerConfig()
	}
	return &ServerFactory{config: config}
}

// Create creates an HTTP server based on configuration.
func (f *ServerFactory) Create() (Server, error) {
	switch f.config.Type {
	case ServerTypeGin:
		return NewGinServer(f.config)
	// TODO: Implement other server types
	// case ServerTypeEcho:
	// 	return NewEchoServer(f.config)
	// case ServerTypeFiber:
	// 	return NewFiberServer(f.config)
	// case ServerTypeChi:
	// 	return NewChiServer(f.config)
	// case ServerTypeIris:
	// 	return NewIrisServer(f.config)
	// case ServerTypeHertz:
	// 	return NewHertzServer(f.config)
	default:
		return nil, fmt.Errorf("unsupported server type: %s. Currently only 'gin' is supported", f.config.Type)
	}
}

// GetSupportedServers returns list of supported server types.
func GetSupportedServers() []ServerType {
	return []ServerType{
		ServerTypeGin,
		ServerTypeEcho,
		ServerTypeFiber,
		ServerTypeChi,
		ServerTypeIris,
		ServerTypeHertz,
	}
}

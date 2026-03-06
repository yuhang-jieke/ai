package api

import (
	"context"
	"fmt"

	"github.com/yuhang-jieke/ai/internal/httpserver"
	"github.com/yuhang-jieke/ai/internal/middleware/auth"
	"github.com/yuhang-jieke/ai/internal/repository"
)

// LoginRequest 登录请求结构
type LoginRequest struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	UserID uint   `json:"user_id"`
	Token  string `json:"token,omitempty"`
}

// UserHandler 用户处理器
type UserHandler struct {
	userRepo *repository.UserRepository
}

// NewUserHandler 创建新的用户处理器
func NewUserHandler(userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// Login 用户登录
// POST /api/auth/login
func (h *UserHandler) Login(ctx httpserver.Context) error {
	// 1. 绑定请求参数
	var req LoginRequest
	if err := ctx.ShouldBind(&req); err != nil {
		return ctx.Error(400, "参数错误: "+err.Error())
	}

	// 2. 获取 context.Context
	requestCtx := context.Background()

	// 3. 根据账号查询用户
	user, err := h.userRepo.FindByAccount(requestCtx, req.Account)
	if err != nil {
		return ctx.Error(500, "数据库查询失败")
	}

	// 4. 验证用户是否存在
	if user == nil {
		return ctx.Error(401, "账号或密码错误")
	}

	// 5. 验证密码（这里暂时使用明文比较，实际项目应该使用 bcrypt 加密）
	if user.Password != req.Password {
		return ctx.Error(401, "账号或密码错误")
	}

	// 6. 创建JWT配置并生成 Token
	config := auth.DefaultConfig()
	token, err := auth.GenerateToken(config, fmt.Sprintf("%d", user.ID), user.Account)
	if err != nil {
		return ctx.Error(500, "生成认证令牌失败")
	}

	// 7. 返回登录成功响应
	response := LoginResponse{
		UserID: user.ID,
		Token:  token,
	}

	return ctx.Success(response)
}

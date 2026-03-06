package repository

import (
	"context"
	"errors"

	"github.com/yuhang-jieke/ai/internal/model"
	"gorm.io/gorm"
)

// UserRepository 用户仓库，处理用户相关的数据库操作
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建新的用户仓库实例
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByAccount 根据账号查找用户
// account: 用户账号
// 返回：用户信息和错误
func (r *UserRepository) FindByAccount(ctx context.Context, account string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("account = ?", account).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 用户不存在返回 nil
		}
		return nil, err
	}
	return &user, nil
}

// Create 创建新用户
// user: 用户模型
// 返回：用户ID和错误
func (r *UserRepository) Create(ctx context.Context, user *model.User) (uint, error) {
	err := r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

// FindByID 根据ID查找用户
// id: 用户ID
// 返回：用户信息和错误
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

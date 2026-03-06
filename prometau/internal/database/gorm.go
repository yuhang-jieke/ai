// Package database provides database connection management.
package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/yuhang-jieke/ai/internal/config"
	"github.com/yuhang-jieke/ai/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormMySQL 封装 GORM MySQL 连接
type GormMySQL struct {
	DB *gorm.DB
}

// DatabaseOption 数据库连接选项
type DatabaseOption struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
	LogLevel        logger.LogLevel // 日志级别
}

// DefaultDatabaseOption 返回默认数据库选项
func DefaultDatabaseOption() *DatabaseOption {
	return &DatabaseOption{
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		ConnMaxLifetime: 3600,
		LogLevel:        logger.Info,
	}
}

// NewGormMySQL 创建 GORM MySQL 连接
// 支持两种调用方式：
// 1. 传入配置结构体: NewGormMySQL(cfg.Database)
// 2. 传入选项: NewGormMySQL(WithHost("localhost"), WithPort(3306), ...)
func NewGormMySQL(cfg config.DatabaseConfig) (*GormMySQL, error) {
	opt := &DatabaseOption{
		Host:            cfg.Host,
		Port:            cfg.Port,
		Username:        cfg.Username,
		Password:        cfg.Password,
		Database:        cfg.Database,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		LogLevel:        logger.Info,
	}
	return NewGormMySQLWithOption(opt)
}

// NewGormMySQLWithOption 使用选项创建数据库连接
func NewGormMySQLWithOption(opt *DatabaseOption) (*GormMySQL, error) {
	if opt == nil {
		opt = DefaultDatabaseOption()
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		opt.Username,
		opt.Password,
		opt.Host,
		opt.Port,
		opt.Database,
	)

	return NewGormMySQLWithDSN(dsn, opt)
}

// NewGormMySQLWithDSN 使用DSN字符串创建数据库连接
// dsn: Data Source Name, 格式: username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
// opt: 可选的连接池配置，传nil使用默认值
func NewGormMySQLWithDSN(dsn string, opt *DatabaseOption) (*GormMySQL, error) {
	if opt == nil {
		opt = DefaultDatabaseOption()
	}

	slog.Info("正在连接 MySQL", "dsn", maskPassword(dsn))

	// GORM 配置
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(opt.LogLevel),
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层 SQL DB 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 SQL DB 失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(opt.MaxOpenConns)
	sqlDB.SetMaxIdleConns(opt.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(opt.ConnMaxLifetime) * time.Second)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	slog.Info("MySQL 连接成功")

	return &GormMySQL{DB: db}, nil
}

// MigrateOption 迁移选项
type MigrateOption struct {
	Models       []interface{} // 要迁移的模型列表
	SkipExisting bool          // 如果表存在是否跳过
}

// Migrate 执行数据库迁移
// 支持传入要迁移的模型列表，如: db.Migrate(&model.User{}, &model.Product{})
func (g *GormMySQL) Migrate(models ...interface{}) error {
	if len(models) == 0 {
		// 默认迁移所有模型
		models = []interface{}{
			&model.User{},
			&model.Product{},
		}
	}

	slog.Info("开始数据库表迁移...", "models", len(models))

	for _, m := range models {
		if err := g.DB.AutoMigrate(m); err != nil {
			return fmt.Errorf("迁移表失败: %w", err)
		}
	}

	slog.Info("数据库表迁移完成")
	return nil
}

// MigrateWithOption 使用选项执行数据库迁移
func (g *GormMySQL) MigrateWithOption(opt *MigrateOption) error {
	if opt == nil || len(opt.Models) == 0 {
		return g.Migrate()
	}

	slog.Info("开始数据库表迁移...", "models", len(opt.Models))

	for _, m := range opt.Models {
		if opt.SkipExisting && g.DB.Migrator().HasTable(m) {
			slog.Info("表已存在，跳过", "model", getModelName(m))
			continue
		}

		if err := g.DB.AutoMigrate(m); err != nil {
			return fmt.Errorf("迁移表 %s 失败: %w", getModelName(m), err)
		}
	}

	slog.Info("数据库表迁移完成")
	return nil
}

// AutoMigrate 自动迁移数据库表结构（保留原有方法兼容）
// 使用 GORM 的 AutoMigrate 方法自动创建或更新表结构
func (g *GormMySQL) AutoMigrate() error {
	return g.Migrate()
}

// MigrateUsersTable 仅迁移 users 表
func (g *GormMySQL) MigrateUsersTable() error {
	return g.Migrate(&model.User{})
}

// MigrateProductsTable 仅迁移 products 表
func (g *GormMySQL) MigrateProductsTable() error {
	return g.Migrate(&model.Product{})
}

// GetSQLDB 获取底层 SQL DB
func (g *GormMySQL) GetSQLDB() (*sql.DB, error) {
	return g.DB.DB()
}

// Close 关闭数据库连接
func (g *GormMySQL) Close() error {
	if g.DB != nil {
		sqlDB, err := g.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// ============ 辅助函数 ============

// maskPassword 隐藏DSN中的密码
func maskPassword(dsn string) string {
	// 简单隐藏密码显示
	return "***:***@tcp(***)/***"
}

// getModelName 获取模型名称
func getModelName(m interface{}) string {
	return fmt.Sprintf("%T", m)
}

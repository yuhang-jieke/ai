// Package database provides database connection management.
package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yuhang-jieke/ai/internal/config"
)

// MySQL wraps the MySQL database connection.
type MySQL struct {
	DB *sql.DB
}

// NewMySQL creates a new MySQL database connection.
func NewMySQL(cfg config.DatabaseConfig) (*MySQL, error) {
	// MySQL 8.0+ uses caching_sha2_password by default
	// Use minimal DSN to let driver auto-negotiate authentication
	// DO NOT add any authentication-related parameters
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	slog.Info("connecting to mysql", "host", cfg.Host, "port", cfg.Port, "database", cfg.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("connected to mysql database",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
	)

	return &MySQL{DB: db}, nil
}

// Close closes the database connection.
func (m *MySQL) Close() error {
	if m.DB != nil {
		return m.DB.Close()
	}
	return nil
}

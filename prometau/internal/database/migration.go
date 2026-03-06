package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log/slog"
	"path/filepath"
	"strings"
)

// Migration represents a database migration.
type Migration struct {
	Name string
	SQL  string
}

// RunMigrations runs all SQL migration files from the migrations directory.
func RunMigrations(db *sql.DB, migrationsDir string) error {
	// Read migration files
	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Only process .sql files
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		filePath := filepath.Join(migrationsDir, file.Name())
		sqlBytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		// Execute migration
		migrationSQL := string(sqlBytes)
		statements := splitSQLStatements(migrationSQL)

		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.HasPrefix(stmt, "--") {
				continue
			}

			_, err := db.Exec(stmt)
			if err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", file.Name(), err)
			}
		}

		slog.Info("migration applied", "file", file.Name())
	}

	slog.Info("all migrations completed successfully")
	return nil
}

// splitSQLStatements splits SQL content into individual statements.
func splitSQLStatements(sqlContent string) []string {
	// Simple split by semicolon
	// For more complex SQL parsing, consider using a proper SQL parser
	statements := strings.Split(sqlContent, ";")
	return statements
}

// CheckAndCreateProductsTable creates the products table if it doesn't exist.
func CheckAndCreateProductsTable(db *sql.DB) error {
	// Check if table exists
	var tableName string
	err := db.QueryRow(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = 'products'
	`).Scan(&tableName)

	if err == sql.ErrNoRows {
		// Table doesn't exist, create it
		slog.Info("products table does not exist, creating...")

		createTableSQL := `
			CREATE TABLE IF NOT EXISTS products (
				id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '商品 ID',
				name VARCHAR(255) NOT NULL COMMENT '商品名称',
				description TEXT COMMENT '商品描述',
				price DECIMAL(10, 2) NOT NULL DEFAULT 0.00 COMMENT '商品价格',
				stock INT NOT NULL DEFAULT 0 COMMENT '库存数量',
				category_id BIGINT DEFAULT 0 COMMENT '分类 ID',
				image_url VARCHAR(500) DEFAULT '' COMMENT '商品图片 URL',
				status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=下架，1=上架',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
				INDEX idx_category_id (category_id),
				INDEX idx_status (status),
				INDEX idx_name (name)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品表'
		`

		_, err := db.Exec(createTableSQL)
		if err != nil {
			return fmt.Errorf("failed to create products table: %w", err)
		}

		slog.Info("products table created successfully")

		// Insert sample data
		return insertSampleData(db)
	} else if err != nil {
		return fmt.Errorf("failed to check products table: %w", err)
	}

	slog.Info("products table already exists")
	return insertSampleData(db)
}

// CheckAndCreateUsersTable creates the users table if it doesn't exist.
func CheckAndCreateUsersTable(db *sql.DB) error {
	// Check if table exists
	var tableName string
	err := db.QueryRow(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = 'users'
	`).Scan(&tableName)

	if err == sql.ErrNoRows {
		// Table doesn't exist, create it
		slog.Info("users table does not exist, creating...")

		createTableSQL := `
			CREATE TABLE IF NOT EXISTS users (
				id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '用户ID',
				account VARCHAR(30) NOT NULL UNIQUE COMMENT '账号',
				password VARCHAR(255) NOT NULL COMMENT '密码',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
				INDEX idx_account (account)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表'
		`

		_, err := db.Exec(createTableSQL)
		if err != nil {
			return fmt.Errorf("failed to create users table: %w", err)
		}

		slog.Info("users table created successfully")

		// Insert sample data
		return insertSampleUsers(db)
	} else if err != nil {
		return fmt.Errorf("failed to check users table: %w", err)
	}

	slog.Info("users table already exists")
	return insertSampleUsers(db)
}

// insertSampleData inserts sample products into the database.
func insertSampleData(db *sql.DB) error {
	// Check if data already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count products: %w", err)
	}

	if count > 0 {
		slog.Info("sample data already exists, skipping")
		return nil
	}

	// Insert sample products
	sampleData := []struct {
		Name        string
		Description string
		Price       float64
		Stock       int
		CategoryID  int64
		ImageURL    string
	}{
		{"iPhone 15 Pro", "Apple iPhone 15 Pro 256GB 钛金属", 8999.00, 100, 1, "/images/iphone15pro.jpg"},
		{"MacBook Pro 14", "Apple MacBook Pro 14 寸 M3 芯片", 12999.00, 50, 2, "/images/macbookpro14.jpg"},
		{"AirPods Pro 2", "Apple AirPods Pro 2 主动降噪", 1899.00, 200, 3, "/images/airpodspro2.jpg"},
		{"iPad Air", "Apple iPad Air 10.9 寸 64GB", 4799.00, 80, 4, "/images/ipadair.jpg"},
		{"Apple Watch S9", "Apple Watch Series 9 GPS 45mm", 3299.00, 120, 5, "/images/applewatches9.jpg"},
	}

	insertSQL := `
		INSERT INTO products (name, description, price, stock, category_id, image_url, status)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`

	for _, product := range sampleData {
		_, err := db.Exec(insertSQL,
			product.Name,
			product.Description,
			product.Price,
			product.Stock,
			product.CategoryID,
			product.ImageURL,
		)
		if err != nil {
			return fmt.Errorf("failed to insert sample product %s: %w", product.Name, err)
		}
	}

	slog.Info("sample data inserted successfully", "count", len(sampleData))
	return nil
}

// insertSampleUsers inserts sample users into the database.
func insertSampleUsers(db *sql.DB) error {
	// Check if data already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if count > 0 {
		slog.Info("sample users already exist, skipping")
		return nil
	}

	// Insert sample users
	sampleUsers := []struct {
		Account  string
		Password string
	}{
		{"admin", "admin123"},
		{"user1", "password123"},
		{"test", "test123"},
	}

	insertSQL := `
		INSERT INTO users (account, password)
		VALUES (?, ?)
	`

	for _, user := range sampleUsers {
		_, err := db.Exec(insertSQL, user.Account, user.Password)
		if err != nil {
			return fmt.Errorf("failed to insert sample user %s: %w", user.Account, err)
		}
	}

	slog.Info("sample users inserted successfully", "count", len(sampleUsers))
	return nil
}

package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/yuhang-jieke/ai/internal/model"
)

// ProductRepository handles database operations for products.
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new ProductRepository.
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// Create creates a new product.
func (r *ProductRepository) Create(ctx context.Context, p *model.Product) (int64, error) {
	query := `
		INSERT INTO products (name, description, price, stock, category_id, image_url, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		p.Name,
		p.Description,
		p.Price,
		p.Stock,
		p.CategoryID,
		p.ImageURL,
		p.Status,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	p.ID = id
	return id, nil
}

// GetByID retrieves a product by ID.
func (r *ProductRepository) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	query := `
		SELECT id, name, description, price, stock, category_id, image_url, status, created_at, updated_at
		FROM products
		WHERE id = ?
	`
	p := &model.Product{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.Price,
		&p.Stock,
		&p.CategoryID,
		&p.ImageURL,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// GetAll retrieves all products.
func (r *ProductRepository) GetAll(ctx context.Context) ([]*model.Product, error) {
	query := `
		SELECT id, name, description, price, stock, category_id, image_url, status, created_at, updated_at
		FROM products
		ORDER BY id DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]*model.Product, 0)
	for rows.Next() {
		p := &model.Product{}
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.Price,
			&p.Stock,
			&p.CategoryID,
			&p.ImageURL,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}

// Update updates an existing product.
func (r *ProductRepository) Update(ctx context.Context, p *model.Product) error {
	query := `
		UPDATE products
		SET name = ?, description = ?, price = ?, stock = ?, category_id = ?, image_url = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		p.Name,
		p.Description,
		p.Price,
		p.Stock,
		p.CategoryID,
		p.ImageURL,
		p.Status,
		time.Now(),
		p.ID,
	)
	return err
}

// Delete deletes a product by ID.
func (r *ProductRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM products WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetByCategory retrieves products by category ID.
func (r *ProductRepository) GetByCategory(ctx context.Context, categoryID int64) ([]*model.Product, error) {
	query := `
		SELECT id, name, description, price, stock, category_id, image_url, status, created_at, updated_at
		FROM products
		WHERE category_id = ?
		ORDER BY id DESC
	`
	rows, err := r.db.QueryContext(ctx, query, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]*model.Product, 0)
	for rows.Next() {
		p := &model.Product{}
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.Price,
			&p.Stock,
			&p.CategoryID,
			&p.ImageURL,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}

// SearchByName searches products by name.
func (r *ProductRepository) SearchByName(ctx context.Context, keyword string) ([]*model.Product, error) {
	query := `
		SELECT id, name, description, price, stock, category_id, image_url, status, created_at, updated_at
		FROM products
		WHERE name LIKE ?
		ORDER BY id DESC
	`
	rows, err := r.db.QueryContext(ctx, query, "%"+keyword+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]*model.Product, 0)
	for rows.Next() {
		p := &model.Product{}
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.Price,
			&p.Stock,
			&p.CategoryID,
			&p.ImageURL,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}

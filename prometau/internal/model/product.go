package model

import (
	"time"
)

// Product represents a product in the database.
type Product struct {
	ID          int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"column:name;type:varchar(255);not null"`
	Description string    `json:"description" gorm:"column:description;type:text"`
	Price       float64   `json:"price" gorm:"column:price;type:decimal(10,2);not null;default:0.00"`
	Stock       int       `json:"stock" gorm:"column:stock;type:int;not null;default:0"`
	CategoryID  int64     `json:"category_id" gorm:"column:category_id;type:bigint;default:0"`
	ImageURL    string    `json:"image_url" gorm:"column:image_url;type:varchar(500);default:''"`
	Status      int       `json:"status" gorm:"column:status;type:tinyint;not null;default:1"` // 1=active, 0=inactive
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the table name for Product.
func (Product) TableName() string {
	return "products"
}

// ProductStatus represents product status.
type ProductStatus int

const (
	ProductStatusInactive ProductStatus = 0
	ProductStatusActive   ProductStatus = 1
)

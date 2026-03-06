package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/yuhang-jieke/ai/internal/httpserver"
	"github.com/yuhang-jieke/ai/internal/model"
	"github.com/yuhang-jieke/ai/internal/repository"
)

// ProductHandler handles product-related HTTP requests.
type ProductHandler struct {
	repo *repository.ProductRepository
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(repo *repository.ProductRepository) *ProductHandler {
	return &ProductHandler{repo: repo}
}

// GetAllProducts handles GET /api/products
func (h *ProductHandler) GetAllProducts(ctx httpserver.Context) error {
	products, err := h.repo.GetAll(context.Background())
	if err != nil {
		return ctx.Error(http.StatusInternalServerError, err.Error())
	}
	return ctx.Success(products)
}

// GetProduct handles GET /api/products/:id
func (h *ProductHandler) GetProduct(ctx httpserver.Context) error {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ctx.Error(http.StatusBadRequest, "Invalid product ID")
	}

	product, err := h.repo.GetByID(context.Background(), id)
	if err != nil {
		return ctx.Error(http.StatusNotFound, "Product not found")
	}

	return ctx.Success(product)
}

// CreateProduct handles POST /api/products
func (h *ProductHandler) CreateProduct(ctx httpserver.Context) error {
	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		Price       float64 `json:"price" binding:"required"`
		Stock       int     `json:"stock"`
		CategoryID  int64   `json:"category_id"`
		ImageURL    string  `json:"image_url"`
		Status      int     `json:"status"`
	}

	if err := ctx.ShouldBind(&req); err != nil {
		return ctx.Error(http.StatusBadRequest, err.Error())
	}

	product := &model.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		CategoryID:  req.CategoryID,
		ImageURL:    req.ImageURL,
		Status:      req.Status,
	}

	if product.Status == 0 {
		product.Status = 1 // Default to active
	}

	id, err := h.repo.Create(context.Background(), product)
	if err != nil {
		return ctx.Error(http.StatusInternalServerError, err.Error())
	}

	product.ID = id
	return ctx.Success(product)
}

// UpdateProduct handles PUT /api/products/:id
func (h *ProductHandler) UpdateProduct(ctx httpserver.Context) error {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ctx.Error(http.StatusBadRequest, "Invalid product ID")
	}

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Stock       int     `json:"stock"`
		CategoryID  int64   `json:"category_id"`
		ImageURL    string  `json:"image_url"`
		Status      int     `json:"status"`
	}

	if err := ctx.ShouldBind(&req); err != nil {
		return ctx.Error(http.StatusBadRequest, err.Error())
	}

	product, err := h.repo.GetByID(context.Background(), id)
	if err != nil {
		return ctx.Error(http.StatusNotFound, "Product not found")
	}

	// Update fields if provided
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.Stock > 0 {
		product.Stock = req.Stock
	}
	product.CategoryID = req.CategoryID
	if req.ImageURL != "" {
		product.ImageURL = req.ImageURL
	}
	if req.Status > 0 {
		product.Status = req.Status
	}

	if err := h.repo.Update(context.Background(), product); err != nil {
		return ctx.Error(http.StatusInternalServerError, err.Error())
	}

	return ctx.Success(product)
}

// DeleteProduct handles DELETE /api/products/:id
func (h *ProductHandler) DeleteProduct(ctx httpserver.Context) error {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ctx.Error(http.StatusBadRequest, "Invalid product ID")
	}

	if err := h.repo.Delete(context.Background(), id); err != nil {
		return ctx.Error(http.StatusInternalServerError, err.Error())
	}

	return ctx.Success(nil)
}

// SearchProducts handles GET /api/products/search?q=keyword
func (h *ProductHandler) SearchProducts(ctx httpserver.Context) error {
	keyword := ctx.Query("q")
	if keyword == "" {
		return ctx.Error(http.StatusBadRequest, "Missing search keyword")
	}

	products, err := h.repo.SearchByName(context.Background(), keyword)
	if err != nil {
		return ctx.Error(http.StatusInternalServerError, err.Error())
	}

	return ctx.Success(products)
}

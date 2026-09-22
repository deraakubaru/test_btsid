package domain

import (
	"strconv"
	"time"
)

// Product represents the database product entity.
type Product struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Price       float64   `json:"price"`
	Description *string   `json:"description"`
	Category    string    `json:"category"`
	Images      []string  `json:"images"`
	CreatedByID int64     `json:"created_by_id"`
	UpdatedByID int64     `json:"updated_by_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
}

// ProductResponse is the API JSON representation matching SPEC.md.
type ProductResponse struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Price       float64    `json:"price"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Images      []string   `json:"images"`
	CreatedAt   CustomTime `json:"created_at"`
	CreatedBy   string     `json:"created_by"`
	CreatedByID string     `json:"created_by_id"`
	UpdatedAt   CustomTime `json:"updated_at"`
	UpdatedBy   string     `json:"updated_by"`
	UpdatedByID string     `json:"updated_by_id"`
}

// CreateProductRequest defines the input payload for creating a product.
type CreateProductRequest struct {
	Title       string   `json:"title"`
	Price       *float64 `json:"price"`
	Description *string  `json:"description"`
	Category    string   `json:"category"`
	Images      []string `json:"images"`
}

// UpdateProductRequest defines the input payload for replacing a product.
type UpdateProductRequest struct {
	Title       string   `json:"title"`
	Price       *float64 `json:"price"`
	Description *string  `json:"description"`
	Category    string   `json:"category"`
	Images      []string `json:"images"`
}

// PaginationInfo represents metadata for paginated API responses.
type PaginationInfo struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// ToResponse converts a Product entity to API ProductResponse DTO.
func (p *Product) ToResponse() ProductResponse {
	desc := ""
	if p.Description != nil {
		desc = *p.Description
	}
	images := p.Images
	if images == nil {
		images = []string{}
	}
	return ProductResponse{
		ID:          p.ID,
		Title:       p.Title,
		Price:       p.Price,
		Description: desc,
		Category:    p.Category,
		Images:      images,
		CreatedAt:   NewCustomTime(p.CreatedAt),
		CreatedBy:   p.CreatedBy,
		CreatedByID: strconv.FormatInt(p.CreatedByID, 10),
		UpdatedAt:   NewCustomTime(p.UpdatedAt),
		UpdatedBy:   p.UpdatedBy,
		UpdatedByID: strconv.FormatInt(p.UpdatedByID, 10),
	}
}

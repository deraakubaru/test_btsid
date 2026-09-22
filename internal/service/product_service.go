package service

import (
	"context"
	"math"
	"net/url"
	"strings"

	"btsid/internal/domain"
	"btsid/internal/repository"
)

type ProductService struct {
	productRepo repository.ProductRepository
}

func NewProductService(productRepo repository.ProductRepository) *ProductService {
	return &ProductService{
		productRepo: productRepo,
	}
}

// ListProducts retrieves paginated products with filter validation and pagination calculations.
func (s *ProductService) ListProducts(ctx context.Context, search, category string, page, limit int) ([]domain.ProductResponse, domain.PaginationInfo, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		return nil, domain.PaginationInfo{}, domain.NewValidationError("limit cannot exceed 100")
	}

	products, totalItems, err := s.productRepo.GetProducts(ctx, search, category, page, limit)
	if err != nil {
		return nil, domain.PaginationInfo{}, err
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(limit)))
	}

	pagination := domain.PaginationInfo{
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}

	responses := make([]domain.ProductResponse, 0, len(products))
	for _, p := range products {
		responses = append(responses, p.ToResponse())
	}

	return responses, pagination, nil
}

// GetProductByID retrieves a single product by ID.
func (s *ProductService) GetProductByID(ctx context.Context, id int64) (*domain.ProductResponse, error) {
	p, err := s.productRepo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := p.ToResponse()
	return &resp, nil
}

// CreateProduct validates payload, binds audit user IDs, and persists a new product.
func (s *ProductService) CreateProduct(ctx context.Context, req domain.CreateProductRequest, userID int64) (*domain.ProductResponse, error) {
	if err := s.validateProductInput(req.Title, req.Price, req.Category, req.Images); err != nil {
		return nil, err
	}

	product := &domain.Product{
		Title:       strings.TrimSpace(req.Title),
		Price:       *req.Price,
		Description: req.Description,
		Category:    strings.TrimSpace(req.Category),
		Images:      req.Images,
		CreatedByID: userID,
		UpdatedByID: userID,
	}

	created, err := s.productRepo.CreateProduct(ctx, product)
	if err != nil {
		return nil, err
	}

	resp := created.ToResponse()
	return &resp, nil
}

// UpdateProduct validates payload and updates an existing product with full replacement.
func (s *ProductService) UpdateProduct(ctx context.Context, id int64, req domain.UpdateProductRequest, userID int64) (*domain.ProductResponse, error) {
	if err := s.validateProductInput(req.Title, req.Price, req.Category, req.Images); err != nil {
		return nil, err
	}

	product := &domain.Product{
		ID:          id,
		Title:       strings.TrimSpace(req.Title),
		Price:       *req.Price,
		Description: req.Description,
		Category:    strings.TrimSpace(req.Category),
		Images:      req.Images,
		UpdatedByID: userID,
	}

	updated, err := s.productRepo.UpdateProduct(ctx, product)
	if err != nil {
		return nil, err
	}

	resp := updated.ToResponse()
	return &resp, nil
}

// DeleteProduct deletes a product by ID.
func (s *ProductService) DeleteProduct(ctx context.Context, id int64) error {
	return s.productRepo.DeleteProduct(ctx, id)
}

func (s *ProductService) validateProductInput(title string, price *float64, category string, images []string) error {
	if strings.TrimSpace(title) == "" {
		return domain.NewValidationError("title is required and cannot be empty")
	}
	if strings.TrimSpace(category) == "" {
		return domain.NewValidationError("category is required and cannot be empty")
	}
	if price == nil {
		return domain.NewValidationError("price is required")
	}
	if *price < 0 {
		return domain.NewValidationError("price must be a non-negative number")
	}
	if len(images) == 0 {
		return domain.NewValidationError("images must contain at least 1 item")
	}
	for _, img := range images {
		if !isValidURL(img) {
			return domain.NewValidationError("image item must be a valid HTTP or HTTPS URL")
		}
	}
	return nil
}

func isValidURL(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	return true
}

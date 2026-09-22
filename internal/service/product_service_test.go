package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"btsid/internal/domain"
)

type mockProductRepository struct {
	mu       sync.RWMutex
	products map[int64]*domain.Product
	idSeq    int64
}

func newMockProductRepository() *mockProductRepository {
	return &mockProductRepository{
		products: make(map[int64]*domain.Product),
	}
}

func (m *mockProductRepository) GetProducts(ctx context.Context, search, category string, page, limit int) ([]*domain.Product, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filtered := []*domain.Product{}
	for _, p := range m.products {
		if search != "" && !strings.Contains(strings.ToLower(p.Title), strings.ToLower(search)) {
			continue
		}
		if category != "" && !strings.EqualFold(p.Category, category) {
			continue
		}
		filtered = append(filtered, p)
	}

	totalItems := int64(len(filtered))
	offset := (page - 1) * limit
	if offset >= len(filtered) {
		return []*domain.Product{}, totalItems, nil
	}

	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[offset:end], totalItems, nil
}

func (m *mockProductRepository) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, exists := m.products[id]
	if !exists {
		return nil, domain.NewNotFoundError("product not found")
	}
	return p, nil
}

func (m *mockProductRepository) CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.idSeq++
	created := &domain.Product{
		ID:          m.idSeq,
		Title:       p.Title,
		Price:       p.Price,
		Description: p.Description,
		Category:    p.Category,
		Images:      p.Images,
		CreatedByID: p.CreatedByID,
		UpdatedByID: p.UpdatedByID,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		CreatedBy:   fmt.Sprintf("user_%d", p.CreatedByID),
		UpdatedBy:   fmt.Sprintf("user_%d", p.UpdatedByID),
	}
	m.products[m.idSeq] = created
	return created, nil
}

func (m *mockProductRepository) UpdateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.products[p.ID]
	if !exists {
		return nil, domain.NewNotFoundError("product not found")
	}

	existing.Title = p.Title
	existing.Price = p.Price
	existing.Description = p.Description
	existing.Category = p.Category
	existing.Images = p.Images
	existing.UpdatedByID = p.UpdatedByID
	existing.UpdatedAt = time.Now().UTC()
	existing.UpdatedBy = fmt.Sprintf("user_%d", p.UpdatedByID)

	return existing, nil
}

func (m *mockProductRepository) DeleteProduct(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.products[id]; !exists {
		return domain.NewNotFoundError("product not found")
	}
	delete(m.products, id)
	return nil
}

func TestProductService_ListProducts(t *testing.T) {
	repo := newMockProductRepository()
	service := NewProductService(repo)
	ctx := context.Background()

	// Populate 25 mock products
	price := 10.0
	for i := 1; i <= 25; i++ {
		_, _ = service.CreateProduct(ctx, domain.CreateProductRequest{
			Title:    fmt.Sprintf("Product %d", i),
			Price:    &price,
			Category: "Clothes",
			Images:   []string{"https://example.com/img.jpg"},
		}, 1)
	}

	t.Run("default page and limit", func(t *testing.T) {
		res, pag, err := service.ListProducts(ctx, "", "", 0, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if pag.Page != 1 || pag.Limit != 10 {
			t.Errorf("expected page 1, limit 10; got page %d, limit %d", pag.Page, pag.Limit)
		}
		if pag.TotalItems != 25 || pag.TotalPages != 3 {
			t.Errorf("expected total_items 25, total_pages 3; got total_items %d, total_pages %d", pag.TotalItems, pag.TotalPages)
		}
		if len(res) != 10 {
			t.Errorf("expected 10 items in page, got %d", len(res))
		}
	})

	t.Run("non-default page and limit", func(t *testing.T) {
		res, pag, err := service.ListProducts(ctx, "", "", 2, 5)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if pag.Page != 2 || pag.Limit != 5 {
			t.Errorf("expected page 2, limit 5; got page %d, limit %d", pag.Page, pag.Limit)
		}
		if pag.TotalPages != 5 {
			t.Errorf("expected total_pages 5, got %d", pag.TotalPages)
		}
		if len(res) != 5 {
			t.Errorf("expected 5 items, got %d", len(res))
		}
	})

	t.Run("limit > 100 validation error", func(t *testing.T) {
		_, _, err := service.ListProducts(ctx, "", "", 1, 101)
		if err == nil {
			t.Fatal("expected error for limit > 100, got nil")
		}
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || !errors.Is(appErr.Code, domain.ErrValidation) {
			t.Errorf("expected validation error, got %v", err)
		}
	})

	t.Run("zero results", func(t *testing.T) {
		emptyRepo := newMockProductRepository()
		emptyService := NewProductService(emptyRepo)
		res, pag, err := emptyService.ListProducts(ctx, "", "", 1, 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if pag.TotalItems != 0 || pag.TotalPages != 0 {
			t.Errorf("expected 0 total items/pages, got items %d, pages %d", pag.TotalItems, pag.TotalPages)
		}
		if len(res) != 0 {
			t.Errorf("expected 0 results, got %d", len(res))
		}
	})
}

func TestProductService_Validation(t *testing.T) {
	repo := newMockProductRepository()
	service := NewProductService(repo)
	ctx := context.Background()

	validPrice := 29.99

	t.Run("empty title", func(t *testing.T) {
		_, err := service.CreateProduct(ctx, domain.CreateProductRequest{
			Title:    "   ",
			Price:    &validPrice,
			Category: "Clothes",
			Images:   []string{"https://example.com/img.jpg"},
		}, 1)
		if err == nil {
			t.Fatal("expected error for empty title, got nil")
		}
	})

	t.Run("empty category", func(t *testing.T) {
		_, err := service.CreateProduct(ctx, domain.CreateProductRequest{
			Title:    "Shirt",
			Price:    &validPrice,
			Category: "   ",
			Images:   []string{"https://example.com/img.jpg"},
		}, 1)
		if err == nil {
			t.Fatal("expected error for empty category, got nil")
		}
	})

	t.Run("negative price", func(t *testing.T) {
		negPrice := -5.0
		_, err := service.CreateProduct(ctx, domain.CreateProductRequest{
			Title:    "Shirt",
			Price:    &negPrice,
			Category: "Clothes",
			Images:   []string{"https://example.com/img.jpg"},
		}, 1)
		if err == nil {
			t.Fatal("expected error for negative price, got nil")
		}
	})

	t.Run("empty images slice", func(t *testing.T) {
		_, err := service.CreateProduct(ctx, domain.CreateProductRequest{
			Title:    "Shirt",
			Price:    &validPrice,
			Category: "Clothes",
			Images:   []string{},
		}, 1)
		if err == nil {
			t.Fatal("expected error for empty images, got nil")
		}
	})

	t.Run("invalid image URL - missing scheme", func(t *testing.T) {
		_, err := service.CreateProduct(ctx, domain.CreateProductRequest{
			Title:    "Shirt",
			Price:    &validPrice,
			Category: "Clothes",
			Images:   []string{"not-a-valid-url"},
		}, 1)
		if err == nil {
			t.Fatal("expected error for invalid image URL, got nil")
		}
	})

	t.Run("invalid image URL - ftp scheme", func(t *testing.T) {
		_, err := service.CreateProduct(ctx, domain.CreateProductRequest{
			Title:    "Shirt",
			Price:    &validPrice,
			Category: "Clothes",
			Images:   []string{"ftp://example.com/file.jpg"},
		}, 1)
		if err == nil {
			t.Fatal("expected error for ftp image URL, got nil")
		}
	})

	t.Run("valid HTTP URL", func(t *testing.T) {
		res, err := service.CreateProduct(ctx, domain.CreateProductRequest{
			Title:    "Shirt HTTP",
			Price:    &validPrice,
			Category: "Clothes",
			Images:   []string{"http://example.com/img.png"},
		}, 1)
		if err != nil {
			t.Fatalf("expected valid HTTP URL to pass, got %v", err)
		}
		if res.Title != "Shirt HTTP" {
			t.Errorf("expected title 'Shirt HTTP', got %s", res.Title)
		}
	})

	t.Run("valid HTTPS URL", func(t *testing.T) {
		res, err := service.CreateProduct(ctx, domain.CreateProductRequest{
			Title:    "Shirt HTTPS",
			Price:    &validPrice,
			Category: "Clothes",
			Images:   []string{"https://example.com/img.png"},
		}, 1)
		if err != nil {
			t.Fatalf("expected valid HTTPS URL to pass, got %v", err)
		}
		if res.Title != "Shirt HTTPS" {
			t.Errorf("expected title 'Shirt HTTPS', got %s", res.Title)
		}
	})
}

func TestProductService_Create(t *testing.T) {
	repo := newMockProductRepository()
	service := NewProductService(repo)
	ctx := context.Background()

	price := 99.99
	desc := "High quality product"
	req := domain.CreateProductRequest{
		Title:       "Awesome Product",
		Price:       &price,
		Description: &desc,
		Category:    "Electronics",
		Images:      []string{"https://example.com/image.jpg"},
	}

	res, err := service.CreateProduct(ctx, req, 42)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.ID <= 0 {
		t.Errorf("expected positive product ID, got %d", res.ID)
	}
	if res.CreatedByID != "42" || res.UpdatedByID != "42" {
		t.Errorf("expected audit IDs to be '42', got created_by_id=%s, updated_by_id=%s", res.CreatedByID, res.UpdatedByID)
	}
	if res.Title != "Awesome Product" || res.Price != 99.99 {
		t.Errorf("expected title 'Awesome Product' and price 99.99, got title=%s, price=%f", res.Title, res.Price)
	}
}

func TestProductService_Update(t *testing.T) {
	repo := newMockProductRepository()
	service := NewProductService(repo)
	ctx := context.Background()

	price := 50.0
	created, _ := service.CreateProduct(ctx, domain.CreateProductRequest{
		Title:    "Original Title",
		Price:    &price,
		Category: "Books",
		Images:   []string{"https://example.com/orig.jpg"},
	}, 1)

	t.Run("valid full replacement", func(t *testing.T) {
		newPrice := 75.0
		desc := "Updated description"
		updateReq := domain.UpdateProductRequest{
			Title:       "Updated Title",
			Price:       &newPrice,
			Description: &desc,
			Category:    "Books",
			Images:      []string{"https://example.com/new.jpg"},
		}

		updated, err := service.UpdateProduct(ctx, created.ID, updateReq, 99)
		if err != nil {
			t.Fatalf("expected no error on update, got %v", err)
		}

		if updated.Title != "Updated Title" || updated.Price != 75.0 {
			t.Errorf("expected updated title and price, got title=%s, price=%f", updated.Title, updated.Price)
		}
		if updated.UpdatedByID != "99" {
			t.Errorf("expected updated_by_id to be '99', got %s", updated.UpdatedByID)
		}
	})

	t.Run("missing product", func(t *testing.T) {
		newPrice := 10.0
		updateReq := domain.UpdateProductRequest{
			Title:    "Title",
			Price:    &newPrice,
			Category: "Books",
			Images:   []string{"https://example.com/new.jpg"},
		}
		_, err := service.UpdateProduct(ctx, 99999, updateReq, 1)
		if err == nil {
			t.Fatal("expected 404 error for non-existent product, got nil")
		}
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || !errors.Is(appErr.Code, domain.ErrNotFound) {
			t.Errorf("expected not found error, got %v", err)
		}
	})
}

func TestProductService_Delete(t *testing.T) {
	repo := newMockProductRepository()
	service := NewProductService(repo)
	ctx := context.Background()

	price := 20.0
	created, _ := service.CreateProduct(ctx, domain.CreateProductRequest{
		Title:    "Product to delete",
		Price:    &price,
		Category: "Tools",
		Images:   []string{"https://example.com/tool.jpg"},
	}, 1)

	t.Run("successful deletion", func(t *testing.T) {
		err := service.DeleteProduct(ctx, created.ID)
		if err != nil {
			t.Fatalf("expected no error on delete, got %v", err)
		}

		// Verify GET returns 404
		_, err = service.GetProductByID(ctx, created.ID)
		if err == nil {
			t.Fatal("expected 404 after deletion, got nil")
		}
	})

	t.Run("missing product deletion", func(t *testing.T) {
		err := service.DeleteProduct(ctx, 99999)
		if err == nil {
			t.Fatal("expected 404 error on deleting missing product, got nil")
		}
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || !errors.Is(appErr.Code, domain.ErrNotFound) {
			t.Errorf("expected not found error, got %v", err)
		}
	})
}

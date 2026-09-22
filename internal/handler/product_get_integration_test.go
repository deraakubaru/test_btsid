package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"btsid/internal/domain"
	"btsid/internal/service"

	"github.com/gin-gonic/gin"
)

// fakeProductRepository provides an in-memory repository for product HTTP tests.
type fakeProductRepository struct {
	mu            sync.RWMutex
	products      map[int64]*domain.Product
	idSeq         int64
	simulateError bool
}

func newFakeProductRepository() *fakeProductRepository {
	return &fakeProductRepository{
		products: make(map[int64]*domain.Product),
	}
}

func (f *fakeProductRepository) GetProducts(ctx context.Context, search, category string, page, limit int) ([]*domain.Product, int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.simulateError {
		return nil, 0, errors.New("db connection failure")
	}

	filtered := []*domain.Product{}
	for _, p := range f.products {
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

func (f *fakeProductRepository) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.simulateError {
		return nil, errors.New("db query error")
	}

	p, exists := f.products[id]
	if !exists {
		return nil, domain.NewNotFoundError("product not found")
	}
	return p, nil
}

func (f *fakeProductRepository) CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.simulateError {
		return nil, errors.New("db error")
	}

	f.idSeq++
	created := &domain.Product{
		ID:          f.idSeq,
		Title:       p.Title,
		Price:       p.Price,
		Description: p.Description,
		Category:    p.Category,
		Images:      p.Images,
		CreatedByID: p.CreatedByID,
		UpdatedByID: p.UpdatedByID,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		CreatedBy:   "john_doe",
		UpdatedBy:   "john_doe",
	}
	f.products[f.idSeq] = created
	return created, nil
}

func (f *fakeProductRepository) UpdateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.simulateError {
		return nil, errors.New("db error")
	}

	existing, exists := f.products[p.ID]
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

	return existing, nil
}

func (f *fakeProductRepository) DeleteProduct(ctx context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.simulateError {
		return errors.New("db error")
	}

	if _, exists := f.products[id]; !exists {
		return domain.NewNotFoundError("product not found")
	}
	delete(f.products, id)
	return nil
}

func setupProductTestApp() (*gin.Engine, *fakeProductRepository) {
	gin.SetMode(gin.TestMode)
	repo := newFakeProductRepository()
	prodService := service.NewProductService(repo)
	prodHandler := NewProductHandler(prodService)

	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group("/api")
	{
		products := api.Group("/products")
		{
			products.GET("", prodHandler.GetProducts)
			products.GET("/:id", prodHandler.GetProductByID)
		}
	}
	return r, repo
}

func TestProductGetIntegration_ListProducts(t *testing.T) {
	r, repo := setupProductTestApp()
	ctx := context.Background()

	// Populate 15 sample products
	desc := "High quality t-shirt"
	for i := 1; i <= 15; i++ {
		category := "Clothes"
		if i > 10 {
			category = "Electronics"
		}
		title := fmt.Sprintf("Shirt %d", i)
		if i > 10 {
			title = fmt.Sprintf("Gadget %d", i)
		}

		_, _ = repo.CreateProduct(ctx, &domain.Product{
			Title:       title,
			Price:       99.99,
			Description: &desc,
			Category:    category,
			Images:      []string{"https://placeimg.com/640/480/any"},
			CreatedByID: 1,
			UpdatedByID: 1,
		})
	}

	t.Run("default request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if resp["success"] != true {
			t.Error("expected success to be true")
		}

		dataList, ok := resp["data"].([]interface{})
		if !ok || len(dataList) != 10 {
			t.Errorf("expected 10 items in default list page, got %v", len(dataList))
		}

		pagination, ok := resp["pagination"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected pagination object")
		}
		if pagination["page"] != float64(1) || pagination["limit"] != float64(10) || pagination["total_items"] != float64(15) || pagination["total_pages"] != float64(2) {
			t.Errorf("unexpected pagination metadata: %v", pagination)
		}
	})

	t.Run("search query", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?search=gadget", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataList := resp["data"].([]interface{})
		if len(dataList) != 5 {
			t.Errorf("expected 5 gadget products, got %d", len(dataList))
		}
	})

	t.Run("category query", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?category=electronics", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataList := resp["data"].([]interface{})
		if len(dataList) != 5 {
			t.Errorf("expected 5 electronics products, got %d", len(dataList))
		}
	})

	t.Run("custom page and limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?page=2&limit=5", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		pagination := resp["pagination"].(map[string]interface{})
		if pagination["page"] != float64(2) || pagination["limit"] != float64(5) {
			t.Errorf("expected page 2 limit 5, got %v", pagination)
		}
	})

	t.Run("malformed page parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?page=abc", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for malformed page, got %d", w.Code)
		}

		var errResp domain.ErrorResponse
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp.Success {
			t.Error("expected success to be false")
		}
	})

	t.Run("malformed limit parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?limit=xyz", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for malformed limit, got %d", w.Code)
		}
	})

	t.Run("limit > 100 validation error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?limit=101", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for limit > 100, got %d", w.Code)
		}
	})

	t.Run("empty result set", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?search=nonexistentproductname", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataList := resp["data"].([]interface{})
		if len(dataList) != 0 {
			t.Errorf("expected empty data array, got %d items", len(dataList))
		}
		pagination := resp["pagination"].(map[string]interface{})
		if pagination["total_items"] != float64(0) || pagination["total_pages"] != float64(0) {
			t.Errorf("expected 0 total items/pages, got %v", pagination)
		}
	})
}

func TestProductGetIntegration_GetProductByID(t *testing.T) {
	r, repo := setupProductTestApp()
	ctx := context.Background()

	desc := "High quality cotton t-shirt"
	created, _ := repo.CreateProduct(ctx, &domain.Product{
		Title:       "Awesome T-Shirt",
		Price:       99.99,
		Description: &desc,
		Category:    "Clothes",
		Images:      []string{"https://placeimg.com/640/480/any"},
		CreatedByID: 1,
		UpdatedByID: 1,
	})

	t.Run("valid product ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/products/%d", created.ID), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if resp["success"] != true {
			t.Error("expected success to be true")
		}

		dataMap, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected data object, got %T", resp["data"])
		}

		if int64(dataMap["id"].(float64)) != created.ID {
			t.Errorf("expected ID %d, got %v", created.ID, dataMap["id"])
		}
		if dataMap["title"] != "Awesome T-Shirt" {
			t.Errorf("expected title Awesome T-Shirt, got %v", dataMap["title"])
		}
		if dataMap["price"] != 99.99 {
			t.Errorf("expected price 99.99, got %v", dataMap["price"])
		}
		if dataMap["created_by"] != "john_doe" || dataMap["created_by_id"] != "1" {
			t.Errorf("unexpected audit fields: created_by=%v, created_by_id=%v", dataMap["created_by"], dataMap["created_by_id"])
		}
	})

	t.Run("malformed product ID - non numeric", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products/abc", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("malformed product ID - negative integer", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products/-5", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("missing product / 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products/99999", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", w.Code)
		}

		var errResp domain.ErrorResponse
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp.Success {
			t.Error("expected success to be false")
		}
	})

	t.Run("unexpected server error / 500", func(t *testing.T) {
		repo.simulateError = true
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products/1", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 Internal Server Error, got %d", w.Code)
		}

		var errResp domain.ErrorResponse
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp.Success {
			t.Error("expected success to be false")
		}
		if errResp.Message != "Internal server error" {
			t.Errorf("expected sanitized error message, got %s", errResp.Message)
		}
		repo.simulateError = false
	})
}

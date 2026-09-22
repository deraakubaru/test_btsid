package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"btsid/internal/domain"
	"btsid/internal/middleware"
	"btsid/internal/security"
	"btsid/internal/service"

	"github.com/gin-gonic/gin"
)

func setupMutationTestApp(secret string) (*gin.Engine, *fakeProductRepository) {
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

			protected := products.Group("")
			protected.Use(middleware.Authenticate(secret))
			{
				protected.POST("", prodHandler.CreateProduct)
				protected.PUT("/:id", prodHandler.UpdateProduct)
				protected.DELETE("/:id", prodHandler.DeleteProduct)
			}
		}
	}
	return r, repo
}

func getValidAuthHeader(t *testing.T, userID int64, username, secret string) string {
	tokens, err := security.GenerateTokens(userID, username, secret, 15, 7)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}
	return "Bearer " + tokens.AuthenticationToken
}

func TestProductMutation_POST(t *testing.T) {
	secret := "mutation-secret"
	router, repo := setupMutationTestApp(secret)
	authHeader := getValidAuthHeader(t, 42, "john_doe", secret)

	validPayload := map[string]interface{}{
		"title":       "Awesome T-Shirt",
		"price":       99.99,
		"description": "High-quality cotton t-shirt",
		"category":    "Clothes",
		"images":      []string{"https://example.com/image.jpg"},
	}

	t.Run("valid authenticated request", func(t *testing.T) {
		jsonBody, _ := json.Marshal(validPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/products", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp ResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if !resp.Success {
			t.Error("expected success to be true")
		}

		dataMap, ok := resp.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected data object, got %T", resp.Data)
		}

		if dataMap["title"] != "Awesome T-Shirt" {
			t.Errorf("expected title 'Awesome T-Shirt', got %v", dataMap["title"])
		}
		if dataMap["created_by_id"] != "42" || dataMap["updated_by_id"] != "42" {
			t.Errorf("expected created_by_id/updated_by_id to be '42', got created=%v, updated=%v", dataMap["created_by_id"], dataMap["updated_by_id"])
		}
	})

	t.Run("missing auth header", func(t *testing.T) {
		jsonBody, _ := json.Marshal(validPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/products", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/products", bytes.NewBufferString("{invalid_json"))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("validation failure - empty title", func(t *testing.T) {
		invalidPayload := map[string]interface{}{
			"title":    "",
			"price":    99.99,
			"category": "Clothes",
			"images":   []string{"https://example.com/image.jpg"},
		}
		jsonBody, _ := json.Marshal(invalidPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/products", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("internal repository error", func(t *testing.T) {
		repo.simulateError = true
		jsonBody, _ := json.Marshal(validPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/products", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 Internal Server Error, got %d", w.Code)
		}
		repo.simulateError = false
	})
}

func TestProductMutation_PUT(t *testing.T) {
	secret := "mutation-secret"
	router, repo := setupMutationTestApp(secret)
	authHeaderUser1 := getValidAuthHeader(t, 1, "john_doe", secret)
	authHeaderUser2 := getValidAuthHeader(t, 99, "alice", secret)

	desc := "Initial description"
	created, _ := repo.CreateProduct(context.Background(), &domain.Product{
		Title:       "Original Product",
		Price:       50.0,
		Description: &desc,
		Category:    "Books",
		Images:      []string{"https://example.com/orig.jpg"},
		CreatedByID: 1,
		UpdatedByID: 1,
	})

	updatePayload := map[string]interface{}{
		"title":       "Updated Product Title",
		"price":       75.50,
		"description": "Updated product description",
		"category":    "Books",
		"images":      []string{"https://example.com/new.jpg"},
	}

	t.Run("valid authenticated request by user 99", func(t *testing.T) {
		jsonBody, _ := json.Marshal(updatePayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/products/%d", created.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", authHeaderUser2)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp ResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataMap := resp.Data.(map[string]interface{})

		if dataMap["title"] != "Updated Product Title" {
			t.Errorf("expected updated title, got %v", dataMap["title"])
		}
		if dataMap["created_by_id"] != "1" {
			t.Errorf("expected created_by_id preserved as '1', got %v", dataMap["created_by_id"])
		}
		if dataMap["updated_by_id"] != "99" {
			t.Errorf("expected updated_by_id to be updated to '99', got %v", dataMap["updated_by_id"])
		}
	})

	t.Run("missing auth header", func(t *testing.T) {
		jsonBody, _ := json.Marshal(updatePayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/products/%d", created.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("malformed ID - non numeric", func(t *testing.T) {
		jsonBody, _ := json.Marshal(updatePayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/products/abc", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", authHeaderUser1)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("negative product ID", func(t *testing.T) {
		jsonBody, _ := json.Marshal(updatePayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/products/-5", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", authHeaderUser1)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/products/%d", created.ID), bytes.NewBufferString("{bad_json"))
		req.Header.Set("Authorization", authHeaderUser1)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("validation failure - negative price", func(t *testing.T) {
		badPayload := map[string]interface{}{
			"title":    "Title",
			"price":    -10.0,
			"category": "Books",
			"images":   []string{"https://example.com/img.jpg"},
		}
		jsonBody, _ := json.Marshal(badPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/products/%d", created.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", authHeaderUser1)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for negative price, got %d", w.Code)
		}
	})

	t.Run("missing product / 404", func(t *testing.T) {
		jsonBody, _ := json.Marshal(updatePayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/products/99999", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", authHeaderUser1)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", w.Code)
		}
	})

	t.Run("internal repository error", func(t *testing.T) {
		repo.simulateError = true
		jsonBody, _ := json.Marshal(updatePayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/products/%d", created.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", authHeaderUser1)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 Internal Server Error, got %d", w.Code)
		}
		repo.simulateError = false
	})
}

func TestProductMutation_DELETE(t *testing.T) {
	secret := "mutation-secret"
	router, repo := setupMutationTestApp(secret)
	authHeader := getValidAuthHeader(t, 1, "john_doe", secret)

	desc := "To be deleted"
	created, _ := repo.CreateProduct(context.Background(), &domain.Product{
		Title:       "Product to Delete",
		Price:       25.0,
		Description: &desc,
		Category:    "Tools",
		Images:      []string{"https://example.com/tool.jpg"},
		CreatedByID: 1,
		UpdatedByID: 1,
	})

	t.Run("valid authenticated deletion", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/products/%d", created.ID), nil)
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp ResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if !resp.Success {
			t.Error("expected success to be true")
		}
		if resp.Message != "Product deleted successfully" {
			t.Errorf("expected message 'Product deleted successfully', got %s", resp.Message)
		}
	})

	t.Run("missing auth header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/products/%d", created.ID), nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("malformed ID - non numeric", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/products/xyz", nil)
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("negative product ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/products/-1", nil)
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("missing product / 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/products/99999", nil)
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", w.Code)
		}
	})

	t.Run("internal repository error", func(t *testing.T) {
		repo.simulateError = true
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/products/1", nil)
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 Internal Server Error, got %d", w.Code)
		}
		repo.simulateError = false
	})
}

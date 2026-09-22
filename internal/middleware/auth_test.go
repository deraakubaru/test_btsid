package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"btsid/internal/domain"
	"btsid/internal/security"

	"github.com/gin-gonic/gin"
)

func setupTestRouter(jwtSecret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	protected := r.Group("/protected")
	protected.Use(Authenticate(jwtSecret))
	protected.GET("/test", func(c *gin.Context) {
		userID, username, ok := GetAuthUser(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "context identity missing"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "username": username})
	})

	return r
}

func TestAuthMiddleware(t *testing.T) {
	secret := "test-secret"
	router := setupTestRouter(secret)

	t.Run("missing authorization header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
		var errResp domain.ErrorResponse
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp.Success {
			t.Error("expected success to be false")
		}
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected/test", nil)
		req.Header.Set("Authorization", "Basic invalidtoken")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid token signature", func(t *testing.T) {
		tokens, err := security.GenerateTokens(1, "john", "different-secret", 15, 7)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AuthenticationToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		tokens, err := security.GenerateTokens(1, "john", secret, -1, 7)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AuthenticationToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("refresh token used as access token", func(t *testing.T) {
		tokens, err := security.GenerateTokens(1, "john", secret, 15, 7)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.RefreshToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for refresh token as access token, got %d", w.Code)
		}
	})

	t.Run("valid access token", func(t *testing.T) {
		tokens, err := security.GenerateTokens(1, "john", secret, 15, 7)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AuthenticationToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["username"] != "john" {
			t.Errorf("expected username john, got %v", resp["username"])
		}
		if int64(resp["user_id"].(float64)) != 1 {
			t.Errorf("expected user_id 1, got %v", resp["user_id"])
		}
	})
}

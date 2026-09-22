package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"btsid/internal/domain"
	"btsid/internal/middleware"
	"btsid/internal/security"
	"btsid/internal/service"

	"github.com/gin-gonic/gin"
)

// fakeUserRepository provides an in-memory implementation of repository.UserRepository for HTTP tests.
type fakeUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User
	idSeq int64
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (f *fakeUserRepository) CreateUser(ctx context.Context, username, passwordHash string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.users[username]; exists {
		return nil, domain.NewConflictError("username already exists")
	}

	f.idSeq++
	u := &domain.User{
		ID:           f.idSeq,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	f.users[username] = u
	return u, nil
}

func (f *fakeUserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	u, exists := f.users[username]
	if !exists {
		return nil, domain.NewNotFoundError("user not found")
	}
	return u, nil
}

func (f *fakeUserRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.NewNotFoundError("user not found")
}

func setupTestApp(secret string) (*gin.Engine, *service.AuthService) {
	gin.SetMode(gin.TestMode)
	repo := newFakeUserRepository()
	authService := service.NewAuthService(repo, secret, 15, 7)
	authHandler := NewAuthHandler(authService)

	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		protected := api.Group("/protected")
		protected.Use(middleware.Authenticate(secret))
		protected.GET("/me", func(c *gin.Context) {
			userID, username, ok := middleware.GetAuthUser(c)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "unauthenticated context"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"user_id":  userID,
					"username": username,
				},
			})
		})
	}

	return r, authService
}

func TestAuthIntegration_Register(t *testing.T) {
	secret := "integration-test-secret"
	router, _ := setupTestApp(secret)

	t.Run("valid registration", func(t *testing.T) {
		body := map[string]string{
			"username":              "john",
			"password":              "secret",
			"password_confirmation": "secret",
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
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

		if dataMap["username"] != "john" {
			t.Errorf("expected username john, got %v", dataMap["username"])
		}

		if _, hasPassword := dataMap["password"]; hasPassword {
			t.Error("password field must not be exposed")
		}
		if _, hasPasswordHash := dataMap["password_hash"]; hasPasswordHash {
			t.Error("password_hash field must not be exposed")
		}
	})

	t.Run("missing username", func(t *testing.T) {
		body := map[string]string{
			"username":              "",
			"password":              "secret",
			"password_confirmation": "secret",
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("missing password", func(t *testing.T) {
		body := map[string]string{
			"username":              "john2",
			"password":              "",
			"password_confirmation": "secret",
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("mismatched password confirmation", func(t *testing.T) {
		body := map[string]string{
			"username":              "john3",
			"password":              "secret1",
			"password_confirmation": "secret2",
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		body := map[string]string{
			"username":              "john",
			"password":              "secret",
			"password_confirmation": "secret",
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request for duplicate username, got %d", w.Code)
		}

		var errResp domain.ErrorResponse
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp.Success {
			t.Error("expected success to be false")
		}
	})
}

func TestAuthIntegration_Login(t *testing.T) {
	secret := "integration-test-secret"
	router, _ := setupTestApp(secret)

	// First register a valid user
	regBody := map[string]string{
		"username":              "alice",
		"password":              "supersecret",
		"password_confirmation": "supersecret",
	}
	regJson, _ := json.Marshal(regBody)
	wReg := httptest.NewRecorder()
	reqReg, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(regJson))
	reqReg.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wReg, reqReg)

	t.Run("valid login", func(t *testing.T) {
		body := map[string]string{
			"username": "alice",
			"password": "supersecret",
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
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

		if dataMap["authentication_token"] == "" || dataMap["refresh_token"] == "" {
			t.Error("expected non-empty tokens in login response")
		}
		if dataMap["token_type"] != "Bearer" {
			t.Errorf("expected token_type Bearer, got %v", dataMap["token_type"])
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		body := map[string]string{
			"username": "alice",
			"password": "wrongpassword",
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("unknown username", func(t *testing.T) {
		body := map[string]string{
			"username": "nobody",
			"password": "supersecret",
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBufferString("{invalid_json"))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d", w.Code)
		}
	})
}

func TestAuthIntegration_Middleware(t *testing.T) {
	secret := "integration-test-secret"
	router, _ := setupTestApp(secret)

	// Register and login to get valid tokens
	regBody := map[string]string{
		"username":              "bob",
		"password":              "supersecret",
		"password_confirmation": "supersecret",
	}
	regJson, _ := json.Marshal(regBody)
	wReg := httptest.NewRecorder()
	reqReg, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(regJson))
	reqReg.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wReg, reqReg)

	loginBody := map[string]string{
		"username": "bob",
		"password": "supersecret",
	}
	loginJson, _ := json.Marshal(loginBody)
	wLogin := httptest.NewRecorder()
	reqLogin, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(loginJson))
	reqLogin.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wLogin, reqLogin)

	var loginResp ResponseEnvelope
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	tokenData := loginResp.Data.(map[string]interface{})
	accessToken := tokenData["authentication_token"].(string)
	refreshToken := tokenData["refresh_token"].(string)

	t.Run("missing Authorization header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected/me", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("malformed Bearer header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected/me", nil)
		req.Header.Set("Authorization", "Token "+accessToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected/me", nil)
		req.Header.Set("Authorization", "Bearer invalid.jwt.string")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		expiredTokenPair, _ := security.GenerateTokens(1, "bob", secret, -1, 7)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected/me", nil)
		req.Header.Set("Authorization", "Bearer "+expiredTokenPair.AuthenticationToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("refresh token used as access token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected/me", nil)
		req.Header.Set("Authorization", "Bearer "+refreshToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("valid access token reaches protected handler", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/protected/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp ResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp.Data.(map[string]interface{})
		if data["username"] != "bob" {
			t.Errorf("expected username bob, got %v", data["username"])
		}
	})
}

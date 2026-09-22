package handler

import (
	"net/http"

	"btsid/internal/domain"
	"btsid/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register handles user registration POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.NewErrorResponse("Invalid request payload", "malformed JSON body"))
		return
	}

	userResp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, http.StatusCreated, "User registered successfully", userResp)
}

// Login handles user authentication POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.NewErrorResponse("Invalid request payload", "malformed JSON body"))
		return
	}

	tokenPair, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		RespondError(c, err)
		return
	}

	RespondSuccess(c, http.StatusOK, "Login successful", tokenPair)
}

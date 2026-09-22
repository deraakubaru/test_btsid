package middleware

import (
	"net/http"
	"strings"

	"btsid/internal/domain"
	"btsid/internal/security"

	"github.com/gin-gonic/gin"
)

const (
	UserIDKey   = "userID"
	UsernameKey = "username"
)

// Authenticate returns a Gin middleware function that enforces JWT Bearer access token authentication.
func Authenticate(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, domain.NewErrorResponse("Unauthorized", "missing Authorization header"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			c.JSON(http.StatusUnauthorized, domain.NewErrorResponse("Unauthorized", "malformed Authorization header, expected 'Bearer <token>'"))
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])

		claims, err := security.ValidateToken(tokenString, jwtSecret, security.TokenTypeAccess)
		if err != nil {
			c.JSON(http.StatusUnauthorized, domain.NewErrorResponse("Unauthorized", err.Error()))
			c.Abort()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)

		c.Next()
	}
}

// GetAuthUser extracts the authenticated user ID and username from the Gin context.
func GetAuthUser(c *gin.Context) (int64, string, bool) {
	userIDVal, exists1 := c.Get(UserIDKey)
	usernameVal, exists2 := c.Get(UsernameKey)
	if !exists1 || !exists2 {
		return 0, "", false
	}

	userID, ok1 := userIDVal.(int64)
	username, ok2 := usernameVal.(string)
	if !ok1 || !ok2 {
		return 0, "", false
	}

	return userID, username, true
}

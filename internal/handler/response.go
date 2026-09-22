package handler

import (
	"errors"
	"net/http"

	"btsid/internal/domain"

	"github.com/gin-gonic/gin"
)

// ResponseEnvelope represents a successful JSON response matching SPEC.md.
type ResponseEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// RespondSuccess sends a 2xx JSON response with standard envelope.
func RespondSuccess(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, ResponseEnvelope{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// RespondError maps errors to HTTP status code and renders standard ErrorResponse envelope.
func RespondError(c *gin.Context, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		errs := appErr.Details
		if len(errs) == 0 {
			errs = []string{appErr.Message}
		}

		switch {
		case errors.Is(appErr.Code, domain.ErrValidation):
			c.JSON(http.StatusBadRequest, domain.NewErrorResponse(appErr.Message, errs...))
		case errors.Is(appErr.Code, domain.ErrConflict):
			c.JSON(http.StatusBadRequest, domain.NewErrorResponse(appErr.Message, errs...))
		case errors.Is(appErr.Code, domain.ErrUnauthorized):
			c.JSON(http.StatusUnauthorized, domain.NewErrorResponse(appErr.Message, errs...))
		case errors.Is(appErr.Code, domain.ErrNotFound):
			c.JSON(http.StatusNotFound, domain.NewErrorResponse(appErr.Message, errs...))
		case errors.Is(appErr.Code, domain.ErrRateLimited):
			c.JSON(http.StatusTooManyRequests, domain.NewErrorResponse(appErr.Message, errs...))
		default:
			c.JSON(http.StatusInternalServerError, domain.NewErrorResponse("Internal server error"))
		}
		return
	}

	if errors.Is(err, domain.ErrConflict) {
		c.JSON(http.StatusBadRequest, domain.NewErrorResponse(err.Error(), err.Error()))
		return
	}
	if errors.Is(err, domain.ErrUnauthorized) {
		c.JSON(http.StatusUnauthorized, domain.NewErrorResponse(err.Error(), err.Error()))
		return
	}
	if errors.Is(err, domain.ErrNotFound) {
		c.JSON(http.StatusNotFound, domain.NewErrorResponse(err.Error(), err.Error()))
		return
	}

	c.JSON(http.StatusInternalServerError, domain.NewErrorResponse("Internal server error"))
}

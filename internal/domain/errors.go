package domain

import (
	"errors"
	"fmt"
)

// Sentinel domain errors
var (
	ErrValidation   = errors.New("validation failed")
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("resource conflict")
	ErrRateLimited  = errors.New("too many requests")
	ErrInternal     = errors.New("internal server error")
)

// ErrorResponse represents the standardized Error Envelope matching SPEC.md.
type ErrorResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Errors  []string `json:"errors"`
}

func NewErrorResponse(message string, errs ...string) ErrorResponse {
	if errs == nil {
		errs = []string{}
	}
	return ErrorResponse{
		Success: false,
		Message: message,
		Errors:  errs,
	}
}

// AppError carries domain error classification and detailed message.
type AppError struct {
	Code    error
	Message string
	Details []string
}

func (e *AppError) Error() string {
	if len(e.Details) > 0 {
		return fmt.Sprintf("%s: %v", e.Message, e.Details)
	}
	return e.Message
}

func NewValidationError(msg string, details ...string) *AppError {
	return &AppError{
		Code:    ErrValidation,
		Message: msg,
		Details: details,
	}
}

func NewNotFoundError(msg string) *AppError {
	return &AppError{
		Code:    ErrNotFound,
		Message: msg,
	}
}

func NewConflictError(msg string) *AppError {
	return &AppError{
		Code:    ErrConflict,
		Message: msg,
	}
}

func NewUnauthorizedError(msg string) *AppError {
	return &AppError{
		Code:    ErrUnauthorized,
		Message: msg,
	}
}

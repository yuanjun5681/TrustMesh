package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ListMeta struct {
	Count int `json:"count"`
}

type AppError struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

func NewError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Details: map[string]any{}}
}

func WriteData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

func WriteList(c *gin.Context, items any, count int) {
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"items": items},
		"meta": ListMeta{Count: count},
	})
}

func WriteError(c *gin.Context, err *AppError) {
	details := err.Details
	if details == nil {
		details = map[string]any{}
	}
	c.JSON(err.Status, ErrorResponse{Error: ErrorBody{
		Code:    err.Code,
		Message: err.Message,
		Details: details,
	}})
}

func BadRequest(code, message string) *AppError {
	return NewError(http.StatusBadRequest, code, message)
}

func Unauthorized(message string) *AppError {
	return NewError(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(message string) *AppError {
	return NewError(http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(message string) *AppError {
	return NewError(http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(code, message string) *AppError {
	return NewError(http.StatusConflict, code, message)
}

func Validation(message string, details map[string]any) *AppError {
	err := NewError(http.StatusUnprocessableEntity, "VALIDATION_ERROR", message)
	err.Details = details
	return err
}

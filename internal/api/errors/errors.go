package errors

import (
	"fmt"
	"net/http"
)

// ErrorKind represents different types of API errors
type ErrorKind string

const (
	KindValidation         ErrorKind = "validation"
	KindNotFound           ErrorKind = "not_found"
	KindUnauthorized       ErrorKind = "unauthorized"
	KindForbidden          ErrorKind = "forbidden"
	KindConflict           ErrorKind = "conflict"
	KindInternal           ErrorKind = "internal"
	KindServiceUnavailable ErrorKind = "service_unavailable"
	KindBadRequest         ErrorKind = "bad_request"
)

// APIError represents a structured API error response
type APIError struct {
	Kind      ErrorKind         `json:"kind"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
	Code      string            `json:"code,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// HTTPStatus returns the appropriate HTTP status code for the error kind
func (e *APIError) HTTPStatus() int {
	switch e.Kind {
	case KindValidation:
		return http.StatusUnprocessableEntity
	case KindBadRequest:
		return http.StatusBadRequest
	case KindNotFound:
		return http.StatusNotFound
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindConflict:
		return http.StatusConflict
	case KindServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// NewValidationError creates a validation error with field details
func NewValidationError(message string, fields map[string]string) *APIError {
	return &APIError{
		Kind:    KindValidation,
		Message: message,
		Details: fields,
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *APIError {
	return &APIError{
		Kind:    KindNotFound,
		Message: fmt.Sprintf("%s not found", resource),
	}
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string) *APIError {
	return &APIError{
		Kind:    KindUnauthorized,
		Message: message,
	}
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(message string) *APIError {
	return &APIError{
		Kind:    KindForbidden,
		Message: message,
	}
}

// NewConflictError creates a conflict error
func NewConflictError(message string) *APIError {
	return &APIError{
		Kind:    KindConflict,
		Message: message,
	}
}

// NewInternalError creates an internal server error
func NewInternalError(message string) *APIError {
	return &APIError{
		Kind:    KindInternal,
		Message: message,
	}
}

// NewBadRequestError creates a bad request error
func NewBadRequestError(message string) *APIError {
	return &APIError{
		Kind:    KindBadRequest,
		Message: message,
	}
}

// NewServiceUnavailableError creates a service unavailable error
func NewServiceUnavailableError(message string) *APIError {
	return &APIError{
		Kind:    KindServiceUnavailable,
		Message: message,
	}
}

// WrapError wraps an existing error with API error context
func WrapError(err error, kind ErrorKind, message string) *APIError {
	if err == nil {
		return nil
	}
	
	apiErr := &APIError{
		Kind:    kind,
		Message: message,
	}
	
	// If the original error is already an APIError, preserve details
	if origAPIErr, ok := err.(*APIError); ok {
		if origAPIErr.Details != nil {
			apiErr.Details = origAPIErr.Details
		}
		if origAPIErr.Code != "" {
			apiErr.Code = origAPIErr.Code
		}
	}
	
	return apiErr
}
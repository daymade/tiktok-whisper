package errors

import (
	"fmt"
	"strings"
)

// Common error types
var (
	// Configuration errors
	ErrMissingAPIKey       = New("API key is required")
	ErrInvalidAPIKey       = New("invalid API key format")
	ErrMissingConfig       = New("configuration is required")
	ErrInvalidConfig       = New("invalid configuration")
	
	// Provider errors
	ErrProviderNotFound    = New("provider not found")
	ErrProviderDisabled    = New("provider is disabled")
	ErrProviderTimeout     = New("provider timeout")
	
	// Database errors
	ErrDatabaseConnection  = New("database connection failed")
	ErrQueryFailed         = New("query failed")
	ErrScanFailed          = New("scan failed")
	ErrInsertFailed        = New("insert failed")
	ErrUpdateFailed        = New("update failed")
	
	// File errors
	ErrFileNotFound        = New("file not found")
	ErrFileReadFailed      = New("file read failed")
	ErrFileWriteFailed     = New("file write failed")
	
	// Network errors
	ErrConnectionFailed    = New("connection failed")
	ErrRequestFailed       = New("request failed")
	ErrResponseInvalid     = New("invalid response")
)

// Error represents a standardized error
type Error struct {
	message string
	cause   error
}

// New creates a new error
func New(message string) *Error {
	return &Error{message: message}
}

// Newf creates a new formatted error
func Newf(format string, args ...interface{}) *Error {
	return &Error{message: fmt.Sprintf(format, args...)}
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return &Error{
		message: message,
		cause:   err,
	}
}

// Wrapf wraps an error with formatted context
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &Error{
		message: fmt.Sprintf(format, args...),
		cause:   err,
	}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.cause
}

// Is checks if the error matches target
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.message == t.message
}

// Helper functions for common patterns

// RequiredField returns an error for missing required fields
func RequiredField(field string) error {
	return Newf("%s is required", field)
}

// InvalidField returns an error for invalid field values
func InvalidField(field string, reason string) error {
	return Newf("%s is invalid: %s", field, reason)
}

// InvalidFormat returns an error for invalid format
func InvalidFormat(field string, expected string) error {
	return Newf("%s format invalid: expected %s", field, expected)
}

// TooShort returns an error for values that are too short
func TooShort(field string, minLength int) error {
	return Newf("%s too short (minimum %d characters)", field, minLength)
}

// TooLong returns an error for values that are too long
func TooLong(field string, maxLength int) error {
	return Newf("%s too long (maximum %d characters)", field, maxLength)
}

// OutOfRange returns an error for values outside acceptable range
func OutOfRange(field string, min, max interface{}) error {
	return Newf("%s out of range (must be between %v and %v)", field, min, max)
}

// NotFound returns an error for items that were not found
func NotFound(itemType string, identifier string) error {
	return Newf("%s not found: %s", itemType, identifier)
}

// AlreadyExists returns an error for items that already exist
func AlreadyExists(itemType string, identifier string) error {
	return Newf("%s already exists: %s", itemType, identifier)
}

// Timeout returns a timeout error
func Timeout(operation string, duration string) error {
	return Newf("%s timeout after %s", operation, duration)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "too short") ||
		strings.Contains(msg, "too long") ||
		strings.Contains(msg, "out of range")
}
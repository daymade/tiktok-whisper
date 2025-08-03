package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"tiktok-whisper/internal/api/errors"
)

// Validator interface for domain validation
type Validator interface {
	Validate() error
}

// ValidateRequest validates both struct tags and domain rules
func ValidateRequest(c *gin.Context, req interface{}) error {
	// First, perform struct tag validation
	if err := c.ShouldBindJSON(req); err != nil {
		validationErrors := make(map[string]string)

		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, fieldError := range validationErrs {
				field := strings.ToLower(fieldError.Field())

				switch fieldError.Tag() {
				case "required":
					validationErrors[field] = "is required"
				case "email":
					validationErrors[field] = "must be a valid email"
				case "min":
					validationErrors[field] = "is too short"
				case "max":
					validationErrors[field] = "is too long"
				case "oneof":
					validationErrors[field] = "must be one of the allowed values"
				default:
					validationErrors[field] = "is invalid"
				}
			}
		} else {
			validationErrors["request"] = "invalid JSON format"
		}

		return errors.NewValidationError("Validation failed", validationErrors)
	}

	// Then, perform domain validation if the struct implements Validator
	if validator, ok := req.(Validator); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ValidateQuery validates query parameters
func ValidateQuery(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindQuery(req); err != nil {
		validationErrors := make(map[string]string)

		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, fieldError := range validationErrs {
				field := strings.ToLower(fieldError.Field())
				validationErrors[field] = "invalid query parameter"
			}
		} else {
			validationErrors["query"] = "invalid query parameters"
		}

		return errors.NewBadRequestError("Invalid query parameters")
	}

	// Perform domain validation if available
	if validator, ok := req.(Validator); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return nil
}
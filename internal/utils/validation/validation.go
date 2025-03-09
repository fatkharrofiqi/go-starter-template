package validation

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationError represents a structured validation error
type ValidationError struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
}

// Error makes ValidationError implement the error interface
func (v *ValidationError) Error() string {
	return v.Message
}

// ValidateStruct validates a struct and returns all validation errors
func ValidateStruct(validate *validator.Validate, data interface{}) error {
	errors := make(map[string][]string)

	// Validate each field separately
	val := validate.Struct(data)
	if val != nil {
		validationErrors := val.(validator.ValidationErrors)

		// Iterate over each validation error
		for _, err := range validationErrors {
			field := strings.ToLower(err.Field()) // Convert field name to lowercase
			message := ""

			// Generate meaningful error messages
			switch err.Tag() {
			case "required":
				message = fmt.Sprintf("%s is required", field)
			case "email":
				message = fmt.Sprintf("%s must be a valid email address", field)
			case "min":
				message = fmt.Sprintf("%s must be at least %s characters long", field, err.Param())
			case "max":
				message = fmt.Sprintf("%s must not exceed %s characters", field, err.Param())
			case "alpha":
				message = fmt.Sprintf("%s must contain only alphabetic characters", field)
			default:
				message = fmt.Sprintf("%s is invalid (%s)", field, err.Tag())
			}

			// Append multiple messages for the same field
			errors[field] = append(errors[field], message)
		}

		// Return structured validation errors
		return &ValidationError{
			Message: "Validation failed",
			Errors:  errors,
		}
	}

	return nil
}

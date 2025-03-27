package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
}

func (v *ValidationError) Error() string {
	return v.Message
}

type Validation struct {
	Validator *validator.Validate
}

func NewValidation() *Validation {
	return &Validation{
		Validator: validator.New(),
	}
}

func (v *Validation) Validate(data interface{}) error {
	errors := make(map[string][]string)

	// Validate each field separately
	val := v.Validator.Struct(data)
	if val != nil {
		validationErrors := val.(validator.ValidationErrors)

		// Iterate over each validation error
		for _, err := range validationErrors {
			// Use reflection to get the JSON tag
			field, _ := reflect.TypeOf(data).Elem().FieldByName(err.StructField())
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" {
				jsonTag = strings.ToLower(err.StructField()) // Fallback to lowercase field name
			}

			message := ""
			switch err.Tag() {
			case "required":
				message = fmt.Sprintf("%s is required", jsonTag)
			case "email":
				message = fmt.Sprintf("%s must be a valid email address", jsonTag)
			case "min":
				message = fmt.Sprintf("%s must be at least %s characters long", jsonTag, err.Param())
			case "max":
				message = fmt.Sprintf("%s must not exceed %s characters", jsonTag, err.Param())
			case "alpha":
				message = fmt.Sprintf("%s must contain only alphabetic characters", jsonTag)
			default:
				message = fmt.Sprintf("%s is invalid (%s)", jsonTag, err.Tag())
			}

			// Append multiple messages for the same jsonTag
			errors[jsonTag] = append(errors[jsonTag], message)
		}

		// Return structured validation errors
		return &ValidationError{
			Message: "Validation failed",
			Errors:  errors,
		}
	}

	return nil
}

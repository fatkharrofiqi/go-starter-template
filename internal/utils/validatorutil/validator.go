package validatorutil

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidateStruct validates a struct and returns a map of error messages grouped by field
func ValidateStruct(validate *validator.Validate, data interface{}) map[string][]string {
	errors := make(map[string][]string)
	err := validate.Struct(data)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			field := strings.ToLower(err.Field())
			message := ""

			switch err.Tag() {
			case "required":
				message = fmt.Sprintf("%s is required", field)
			case "email":
				message = fmt.Sprintf("%s must be a valid email address", field)
			case "min":
				message = fmt.Sprintf("%s must be at least %s characters long", field, err.Param())
			case "max":
				message = fmt.Sprintf("%s must not exceed %s characters", field, err.Param())
			default:
				message = fmt.Sprintf("%s is invalid (%s)", field, err.Tag())
			}

			// Append the error message to the field's error list
			errors[field] = append(errors[field], message)
		}
	}
	return errors
}

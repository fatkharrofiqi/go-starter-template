package validation

import (
	"fmt"
	"go-starter-template/internal/utils/errcode"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ValidationError struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
}

func (v *ValidationError) Error() string {
	return v.Message
}

type Validation struct {
	validator *validator.Validate
}

func NewValidation() *Validation {
	return &Validation{validator.New()}
}

// ParseAndValidate parses the request body into the provided struct and validates it.
func (v *Validation) ParseAndValidate(c *fiber.Ctx, data interface{}) error {
	// Parse the request body into the struct
	if err := c.BodyParser(data); err != nil {
		return errcode.ErrBadRequest
	}

	// Validate the parsed struct
	if err := v.Validate(data); err != nil {
		return err
	}

	return nil
}

func (v *Validation) Validate(data interface{}) error {
	errors := make(map[string][]string)

	// Validate the struct
	if err := v.validator.Struct(data); err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)
		if !ok {
			return fmt.Errorf("unexpected validation error: %v", err)
		}

		// Get the type of the struct (dereference the pointer)
		rt := reflect.TypeOf(data).Elem()

		// Iterate over each validation error
		for _, err := range validationErrors {
			// Get the field by name
			field, found := rt.FieldByName(err.StructField())
			if !found {
				continue // Skip if field not found
			}

			// Get the JSON tag or fallback to lowercase field name
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" {
				jsonTag = strings.ToLower(err.StructField())
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

		if len(errors) > 0 {
			return &ValidationError{
				Message: "Validation failed",
				Errors:  errors,
			}
		}
	}

	return nil
}

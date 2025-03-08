package config

import (
	"github.com/go-playground/validator/v10"
)

// NewValidator initializes a new Validator instance
func NewValidator() *validator.Validate {
	validate := validator.New()

	return validate
}

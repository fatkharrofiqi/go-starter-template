package test

import (
	"fmt"
	"go-starter-template/internal/config/validation"
	"go-starter-template/internal/dto"
	"testing"
)

func TestValidation(t *testing.T) {
	// Create an instance of LogoutRequest with invalid data
	logoutRequest := dto.LogoutRequest{
		AccessToken:  "abc1", // This should trigger both "min" and "alpha" validation errors
		RefreshToken: "ab",   // This should trigger a "min" validation error
	}

	// Initialize the validator
	validator := validation.NewValidation()

	// Validate the LogoutRequest
	err := validator.Validate(&logoutRequest)
	if err != nil {
		// Print the validation errors
		if validationErr, ok := err.(*validation.ValidationError); ok {
			fmt.Println("Validation failed with the following errors:")
			for field, messages := range validationErr.Errors {
				for _, message := range messages {
					fmt.Printf("Field: %s, Error: %s\n", field, message)
				}
			}
		} else {
			fmt.Println("An unexpected error occurred:", err)
		}
	} else {
		fmt.Println("Validation passed.")
	}
}

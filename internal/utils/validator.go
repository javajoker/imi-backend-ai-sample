// internal/utils/validator.go
package utils

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	validate.RegisterValidation("strong_password", validateStrongPassword)
	validate.RegisterValidation("username", validateUsername)
}

func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 {
		return false
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
}

func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()

	// Username should be alphanumeric and underscores, 3-50 characters
	if len(username) < 3 || len(username) > 50 {
		return false
	}

	matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", username)
	return matched
}

// Validation tags for common fields
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

func GetValidationErrors(err error) []ValidationError {
	var validationErrors []ValidationError

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			validationErrors = append(validationErrors, ValidationError{
				Field:   strings.ToLower(e.Field()),
				Tag:     e.Tag(),
				Message: getValidationMessage(e),
			})
		}
	}

	return validationErrors
}

func getValidationMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return e.Field() + " is required"
	case "email":
		return "Invalid email format"
	case "min":
		return e.Field() + " must be at least " + e.Param() + " characters"
	case "max":
		return e.Field() + " must be at most " + e.Param() + " characters"
	case "strong_password":
		return "Password must contain at least 8 characters with uppercase, lowercase, number, and special character"
	case "username":
		return "Username must be 3-50 characters and contain only letters, numbers, and underscores"
	default:
		return e.Field() + " is invalid"
	}
}

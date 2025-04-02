package runtime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ErrValidationEmail is the sentinel error returned when an email fails validation
var (
	ErrValidationEmail = errors.New("email: failed to pass regex validation")
)

// ClientAPIError represents type for client API errors.
type ClientAPIError struct {
	err error
}

// Error implements the error interface.
func (e *ClientAPIError) Error() string {
	if e.err == nil {
		return "client api error"
	}
	return e.err.Error()
}

// Unwrap returns the underlying error.
func (e *ClientAPIError) Unwrap() error {
	return e.err
}

// NewClientAPIError creates a new ClientAPIError from the given error.
func NewClientAPIError(err error) error {
	return &ClientAPIError{err: err}
}

type ValidationError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	var messages []string
	for _, e := range ve {
		messages = append(messages, fmt.Sprintf("%s %s", e.Field, e.Error))
	}
	return strings.Join(messages, "\n")
}

func FormatValidationErrors(err error) ValidationErrors {
	var result ValidationErrors
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, ve := range validationErrors {
			result = append(result, ValidationError{
				Field: ve.Field(),
				Error: convertFieldErrorMessage(ve),
			})
		}
	}
	return result
}

func convertFieldErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lt":
		return fmt.Sprintf("must be less than %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	default:
		return fmt.Sprintf("is not valid (%s)", fe.Tag())
	}
}

package runtime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ErrValidationEmail is the sentinel error returned when an email fails validation
var (
	ErrValidationEmail         = errors.New("email: failed to pass regex validation")
	ErrFailedToUnmarshalAsAOrB = errors.New("failed to unmarshal as either A or B")
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

// NewValidationErrorsFromError creates a new ValidationErrors from a single error.
func NewValidationErrorsFromError(err error) ValidationErrors {
	return NewValidationErrorsFromErrors("", []error{err})
}

// NewValidationErrorsFromErrors creates a new ValidationErrors from a list of errors.
// If prefix is provided, it will be prepended to each field name with a dot.
func NewValidationErrorsFromErrors(prefix string, errs []error) ValidationErrors {
	var result ValidationErrors
	var validationErrors validator.ValidationErrors
	if prefix != "" {
		prefix = fmt.Sprintf("%s.", prefix)
	}

	for _, err := range errs {
		if errors.As(err, &validationErrors) {
			for _, ve := range validationErrors {
				result = append(result, ValidationError{
					Field: prefix + ve.Field(),
					Error: convertFieldErrorMessage(ve),
				})
			}
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
	case "min":
		return fmt.Sprintf("length must be greater than or equal to %s", fe.Param())
	case "max":
		return fmt.Sprintf("length must be less than or equal to %s", fe.Param())
	default:
		return fmt.Sprintf("is not valid (%s)", fe.Tag())
	}
}

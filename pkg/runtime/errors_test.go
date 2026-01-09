// Copyright 2025 DoorDash, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package runtime

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientAPIError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		err := NewClientAPIError(nil)
		assert.Equal(t, "client api error", err.Error())
	})

	t.Run("non-nil error", func(t *testing.T) {
		err := NewClientAPIError(ErrValidationEmail)
		assert.Equal(t, ErrValidationEmail.Error(), err.Error())
	})

	t.Run("non-nil error with status code", func(t *testing.T) {
		err := NewClientAPIError(ErrValidationEmail, WithStatusCode(400))
		assert.Equal(t, ErrValidationEmail.Error(), err.Error())

		var apiErr *ClientAPIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 400, apiErr.StatusCode())
	})
}

func TestNewValidationError(t *testing.T) {
	t.Run("empty field", func(t *testing.T) {
		err := NewValidationError("", "is required")
		assert.Equal(t, "is required", err.Error())
	})

	t.Run("non-empty field", func(t *testing.T) {
		err := NewValidationError("foo", "is required")
		assert.Equal(t, "foo is required", err.Error())
	})
}

func TestNewValidationErrorsFromError(t *testing.T) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	type Foo struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
		Age   int    `validate:"gt=18"`
		Rate  int    `validate:"lt=100"`
		Price int    `validate:"lte=20"`
		Qty   int    `validate:"gte=10"`
		Range int    `validate:"gte=10,lt=20"`
	}

	t.Run("multiple errors", func(t *testing.T) {
		foo := Foo{
			Email: "email",
			Rate:  101,
			Price: 21,
			Qty:   9,
			Range: 20,
		}
		err := validate.Struct(foo)
		validationErrors := NewValidationErrorsFromError(err)

		assert.Len(t, validationErrors, 7)
		assert.Equal(t, ValidationError{"Name", "is required", nil}, validationErrors[0])
		assert.Equal(t, ValidationError{"Email", "must be a valid email", nil}, validationErrors[1])
		assert.Equal(t, ValidationError{"Age", "must be greater than 18", nil}, validationErrors[2])
		assert.Equal(t, ValidationError{"Rate", "must be less than 100", nil}, validationErrors[3])
		assert.Equal(t, ValidationError{"Price", "must be less than or equal to 20", nil}, validationErrors[4])
		assert.Equal(t, ValidationError{"Qty", "must be greater than or equal to 10", nil}, validationErrors[5])
		assert.Equal(t, ValidationError{"Range", "must be less than 20", nil}, validationErrors[6])
	})
}

func TestNewValidationErrorsFromErrors(t *testing.T) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	type Foo struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	foo := Foo{
		Name:  "",
		Email: "email",
	}
	err := validate.Struct(foo)
	validationErrors := NewValidationErrorsFromErrors("headers", []error{err})
	validationErrors = append(validationErrors, NewValidationError("Foo", "is required"))

	assert.Len(t, validationErrors, 3)
	assert.Equal(t, "headers.Name", validationErrors[0].Field)
	assert.Equal(t, "is required", validationErrors[0].Message)
	assert.Equal(t, "headers.Email", validationErrors[1].Field)
	assert.Equal(t, "must be a valid email", validationErrors[1].Message)
	assert.Equal(t, "Foo", validationErrors[2].Field)
	assert.Equal(t, "is required", validationErrors[2].Message)
}

func TestNewValidationErrorFromError(t *testing.T) {
	t.Run("wraps error and preserves it", func(t *testing.T) {
		originalErr := errors.New("min length is 3")
		validationErr := NewValidationErrorFromError("Name", originalErr)

		// Check field and message
		assert.Equal(t, "Name", validationErr.Field)
		assert.Equal(t, "min length is 3", validationErr.Message)
		assert.Equal(t, "Name min length is 3", validationErr.Error())

		// Check that original error is preserved
		assert.Equal(t, originalErr, validationErr.Err)
		assert.True(t, errors.Is(validationErr, originalErr))
	})

	t.Run("works with validator errors", func(t *testing.T) {
		validate := validator.New()
		err := validate.Var("ab", "min=3")
		require.Error(t, err)

		validationErr := NewValidationErrorFromError("Name", err)

		// Check that message is converted using convertFieldErrorMessage
		assert.Equal(t, "Name", validationErr.Field)
		assert.Equal(t, "length must be greater than or equal to 3", validationErr.Message)
		assert.Equal(t, "Name length must be greater than or equal to 3", validationErr.Error())

		// Check that we can unwrap to the original validator error
		assert.NotNil(t, validationErr.Err)
		assert.Equal(t, err, validationErr.Unwrap())

		// Check that we can use errors.As to get the validator error
		var validatorErr validator.ValidationErrors
		assert.True(t, errors.As(validationErr, &validatorErr))
	})

	t.Run("empty field", func(t *testing.T) {
		originalErr := errors.New("required")
		validationErr := NewValidationErrorFromError("", originalErr)

		assert.Equal(t, "", validationErr.Field)
		assert.Equal(t, "required", validationErr.Message)
		assert.Equal(t, "required", validationErr.Error())
		assert.Equal(t, originalErr, validationErr.Err)
	})

	t.Run("converts validator.FieldError messages", func(t *testing.T) {
		validate := validator.New()

		testCases := []struct {
			name            string
			value           any
			tag             string
			expectedMessage string
		}{
			{"required", "", "required", "is required"},
			{"email", "invalid", "email", "must be a valid email"},
			{"gt", 5, "gt=10", "must be greater than 10"},
			{"gte", 5, "gte=10", "must be greater than or equal to 10"},
			{"lt", 15, "lt=10", "must be less than 10"},
			{"lte", 15, "lte=10", "must be less than or equal to 10"},
			{"min", "ab", "min=3", "length must be greater than or equal to 3"},
			{"max", "abcdef", "max=3", "length must be less than or equal to 3"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := validate.Var(tc.value, tc.tag)
				require.Error(t, err)

				validationErr := NewValidationErrorFromError("TestField", err)

				assert.Equal(t, "TestField", validationErr.Field)
				assert.Equal(t, tc.expectedMessage, validationErr.Message)
				assert.Equal(t, "TestField "+tc.expectedMessage, validationErr.Error())
				assert.Equal(t, err, validationErr.Err)
			})
		}
	})

	t.Run("validates Amount field with gte=0", func(t *testing.T) {
		validate := validator.New()

		type Payment struct {
			Amount int64 `json:"amount" validate:"gte=0"`
		}

		t.Run("valid - positive amount", func(t *testing.T) {
			payment := Payment{Amount: 100}
			err := validate.Var(payment.Amount, "gte=0")
			assert.NoError(t, err)
		})

		t.Run("valid - zero amount", func(t *testing.T) {
			payment := Payment{Amount: 0}
			err := validate.Var(payment.Amount, "gte=0")
			assert.NoError(t, err)
		})

		t.Run("invalid - negative amount", func(t *testing.T) {
			payment := Payment{Amount: -10}
			err := validate.Var(payment.Amount, "gte=0")
			require.Error(t, err)

			validationErr := NewValidationErrorFromError("Amount", err)

			assert.Equal(t, "Amount", validationErr.Field)
			assert.Equal(t, "must be greater than or equal to 0", validationErr.Message)
			assert.Equal(t, "Amount must be greater than or equal to 0", validationErr.Error())
			assert.Equal(t, err, validationErr.Err)
		})
	})
}

func TestValidationError_Unwrap(t *testing.T) {
	t.Run("returns underlying error", func(t *testing.T) {
		originalErr := errors.New("test error")
		validationErr := NewValidationErrorFromError("Field", originalErr)

		unwrapped := validationErr.Unwrap()
		assert.Equal(t, originalErr, unwrapped)
	})

	t.Run("returns nil when no underlying error", func(t *testing.T) {
		validationErr := NewValidationError("Field", "message")

		unwrapped := validationErr.Unwrap()
		assert.Nil(t, unwrapped)
	})

	t.Run("works with errors.Is", func(t *testing.T) {
		sentinelErr := errors.New("sentinel")
		validationErr := NewValidationErrorFromError("Field", sentinelErr)

		assert.True(t, errors.Is(validationErr, sentinelErr))
	})

	t.Run("works with errors.As", func(t *testing.T) {
		validate := validator.New()
		validatorErr := validate.Var("", "required")
		validationErr := NewValidationErrorFromError("Field", validatorErr)

		var ve validator.ValidationErrors
		assert.True(t, errors.As(validationErr, &ve))
	})
}

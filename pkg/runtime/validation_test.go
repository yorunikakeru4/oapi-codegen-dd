// Copyright 2025 DoorDash, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterCustomTypeFunc_WithEither(t *testing.T) {
	v := validator.New(validator.WithRequiredStructEnabled())
	RegisterCustomTypeFunc(v)

	type TestStruct struct {
		Value Either[string, int] `validate:"required"`
	}

	t.Run("validates active A variant", func(t *testing.T) {
		ts := TestStruct{
			Value: NewEitherFromA[string, int]("hello"),
		}
		err := v.Struct(ts)
		assert.NoError(t, err)
	})

	t.Run("validates active B variant", func(t *testing.T) {
		ts := TestStruct{
			Value: NewEitherFromB[string, int](42),
		}
		err := v.Struct(ts)
		assert.NoError(t, err)
	})

	t.Run("fails validation when no variant is active", func(t *testing.T) {
		ts := TestStruct{
			Value: Either[string, int]{}, // N=0, no active variant
		}
		err := v.Struct(ts)
		assert.Error(t, err)
	})
}

func TestRegisterCustomTypeFunc_WithValidateVar(t *testing.T) {
	v := validator.New(validator.WithRequiredStructEnabled())
	RegisterCustomTypeFunc(v)

	t.Run("validates active A variant with Var", func(t *testing.T) {
		either := NewEitherFromA[string, int]("hello")
		err := v.Var(either, "required")
		assert.NoError(t, err)
	})

	t.Run("validates active B variant with Var", func(t *testing.T) {
		either := NewEitherFromB[string, int](42)
		err := v.Var(either, "required")
		assert.NoError(t, err)
	})

	t.Run("returns nil when no variant is active with Var", func(t *testing.T) {
		either := Either[string, int]{} // N=0, no active variant

		// The custom type function returns nil for inactive variants,
		// which the validator treats as a zero value, not a validation error
		err := v.Var(either, "required")

		// This doesn't error because the validator sees nil from Value()
		// Actual validation of inactive variants should be done via Validate() method
		assert.NoError(t, err)
	})
}

func TestConvertValidatorError(t *testing.T) {
	v := validator.New(validator.WithRequiredStructEnabled())

	t.Run("returns nil for nil error", func(t *testing.T) {
		result := ConvertValidatorError(nil)
		assert.Nil(t, result)
	})

	t.Run("converts ValidationError to ValidationErrors", func(t *testing.T) {
		ve := NewValidationError("field", "message")
		result := ConvertValidatorError(ve)

		// Should be converted to ValidationErrors for consistency
		ves, ok := result.(ValidationErrors)
		require.True(t, ok, "expected ValidationErrors")
		require.Len(t, ves, 1)
		assert.Equal(t, "field", ves[0].Field)
		assert.Equal(t, "message", ves[0].Message)
	})

	t.Run("returns ValidationErrors as-is", func(t *testing.T) {
		ves := ValidationErrors{
			NewValidationError("field1", "message1"),
			NewValidationError("field2", "message2"),
		}
		result := ConvertValidatorError(ves)
		assert.Equal(t, ves, result)
	})

	t.Run("converts wrapped ValidationError to ValidationErrors", func(t *testing.T) {
		ve := NewValidationError("field", "message")
		wrapped := fmt.Errorf("wrapped: %w", ve)

		result := ConvertValidatorError(wrapped)

		// Should be converted to ValidationErrors for consistency
		ves, ok := result.(ValidationErrors)
		require.True(t, ok, "expected ValidationErrors")
		require.Len(t, ves, 1)
		// The wrapped error message becomes the message
		assert.Contains(t, ves[0].Message, "wrapped")
		assert.Contains(t, ves[0].Message, "field")
		assert.Contains(t, ves[0].Message, "message")
	})

	t.Run("handles wrapped ValidationErrors", func(t *testing.T) {
		ves := ValidationErrors{
			NewValidationError("field1", "message1"),
		}
		wrapped := fmt.Errorf("wrapped: %w", ves)

		result := ConvertValidatorError(wrapped)

		assert.Equal(t, wrapped, result)

		var unwrapped ValidationErrors
		assert.True(t, errors.As(result, &unwrapped))
		assert.Len(t, unwrapped, 1)
	})

	t.Run("converts validator.ValidationErrors", func(t *testing.T) {
		type TestStruct struct {
			Name  string `validate:"required"`
			Email string `validate:"required,email"`
		}

		ts := TestStruct{}
		err := v.Struct(ts)
		require.Error(t, err)

		result := ConvertValidatorError(err)

		var ves ValidationErrors
		require.True(t, errors.As(result, &ves), "expected ValidationErrors type")
		assert.NotEmpty(t, ves)
	})

	t.Run("converts wrapped validator.ValidationErrors", func(t *testing.T) {
		type TestStruct struct {
			Name string `validate:"required"`
		}

		ts := TestStruct{}
		validatorErr := v.Struct(ts)
		require.Error(t, validatorErr)

		wrapped := fmt.Errorf("validation failed: %w", validatorErr)

		result := ConvertValidatorError(wrapped)

		var ves ValidationErrors
		require.True(t, errors.As(result, &ves), "expected ValidationErrors type")
		assert.NotEmpty(t, ves)
	})

	t.Run("converts generic error", func(t *testing.T) {
		genericErr := errors.New("some error")

		result := ConvertValidatorError(genericErr)

		var ves ValidationErrors
		require.True(t, errors.As(result, &ves), "expected ValidationErrors type")
		assert.Len(t, ves, 1, "generic errors are now wrapped in ValidationError")
		assert.Equal(t, "", ves[0].Field)
		assert.Equal(t, "some error", ves[0].Message)
		assert.Equal(t, genericErr, ves[0].Err)
	})

	t.Run("converts deeply wrapped ValidationError to ValidationErrors", func(t *testing.T) {
		ve := NewValidationError("field", "message")
		wrapped1 := fmt.Errorf("layer1: %w", ve)
		wrapped2 := fmt.Errorf("layer2: %w", wrapped1)
		wrapped3 := fmt.Errorf("layer3: %w", wrapped2)

		result := ConvertValidatorError(wrapped3)

		// Should be converted to ValidationErrors for consistency
		ves, ok := result.(ValidationErrors)
		require.True(t, ok, "expected ValidationErrors")
		require.Len(t, ves, 1)
		// The wrapped error message becomes the message
		assert.Contains(t, ves[0].Message, "layer3")
		assert.Contains(t, ves[0].Message, "layer2")
		assert.Contains(t, ves[0].Message, "layer1")
	})

	t.Run("handles deeply wrapped ValidationErrors", func(t *testing.T) {
		ves := ValidationErrors{
			NewValidationError("field1", "message1"),
			NewValidationError("field2", "message2"),
		}
		wrapped1 := fmt.Errorf("layer1: %w", ves)
		wrapped2 := fmt.Errorf("layer2: %w", wrapped1)

		result := ConvertValidatorError(wrapped2)

		assert.Equal(t, wrapped2, result)

		var unwrapped ValidationErrors
		assert.True(t, errors.As(result, &unwrapped))
		assert.Len(t, unwrapped, 2)
	})
}

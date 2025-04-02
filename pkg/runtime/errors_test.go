package runtime

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestFormatValidationErrors(t *testing.T) {
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
		validationErrors := FormatValidationErrors(err)

		assert.Len(t, validationErrors, 7)
		assert.Equal(t, ValidationError{"Name", "is required"}, validationErrors[0])
		assert.Equal(t, ValidationError{"Email", "must be a valid email"}, validationErrors[1])
		assert.Equal(t, ValidationError{"Age", "must be greater than 18"}, validationErrors[2])
		assert.Equal(t, ValidationError{"Rate", "must be less than 100"}, validationErrors[3])
		assert.Equal(t, ValidationError{"Price", "must be less than or equal to 20"}, validationErrors[4])
		assert.Equal(t, ValidationError{"Qty", "must be greater than or equal to 10"}, validationErrors[5])
		assert.Equal(t, ValidationError{"Range", "must be less than 20"}, validationErrors[6])
	})
}

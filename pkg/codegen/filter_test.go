package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterOperationsByTag(t *testing.T) {
	packageName := "testswagger"
	t.Run("include tags", func(t *testing.T) {
		cfg := Configuration{
			PackageName: packageName,
			Filter: FilterConfig{
				Include: FilterParamsConfig{
					Tags: []string{"hippo", "giraffe", "cat"},
				},
			},
			UseSingleOutput: true,
		}

		// Run our code generation:
		code, err := Generate([]byte(testDocument), cfg)
		assert.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.Contains(t, code, `type CatDeadCause string`)
	})

	t.Run("exclude tags", func(t *testing.T) {
		opts := Configuration{
			PackageName: packageName,
			Filter: FilterConfig{
				Exclude: FilterParamsConfig{
					Tags: []string{"hippo", "giraffe", "cat"},
				},
			},
			UseSingleOutput: true,
		}

		// Run our code generation:
		code, err := Generate([]byte(testDocument), opts)
		assert.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.NotContains(t, code, `"/cat"`)
	})
}

func TestFilterOperationsByOperationID(t *testing.T) {
	packageName := "testswagger"

	t.Run("include operation ids", func(t *testing.T) {
		opts := Configuration{
			PackageName: packageName,
			Filter: FilterConfig{
				Include: FilterParamsConfig{
					OperationIDs: []string{"getCatStatus"},
				},
			},
			UseSingleOutput: true,
		}

		// Run our code generation:
		code, err := Generate([]byte(testDocument), opts)
		assert.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.Contains(t, code, `type CatDeadCause string`)
	})

	t.Run("exclude operation ids", func(t *testing.T) {
		opts := Configuration{
			PackageName: packageName,
			Filter: FilterConfig{
				Exclude: FilterParamsConfig{
					OperationIDs: []string{"getCatStatus"},
				},
			},
		}

		// Run our code generation:
		code, err := Generate([]byte(testDocument), opts)
		assert.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.NotContains(t, code, `"/cat"`)
	})
}

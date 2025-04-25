package codegen

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			Output: &Output{
				UseSingleFile: true,
			},
		}

		// Run our code generation:
		code, err := Generate([]byte(testDocument), cfg)
		require.NoError(t, err)
		assert.NotEmpty(t, code)

		assert.Contains(t, code.GetCombined(), `type CatDeadCause string`)
	})

	t.Run("exclude tags", func(t *testing.T) {
		opts := Configuration{
			PackageName: packageName,
			Filter: FilterConfig{
				Exclude: FilterParamsConfig{
					Tags: []string{"hippo", "giraffe", "cat"},
				},
			},
			Output: &Output{
				UseSingleFile: true,
			},
		}

		// Run our code generation:
		code, err := Generate([]byte(testDocument), opts)
		require.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.NotContains(t, code.GetCombined(), `"/cat"`)
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
			Output: &Output{
				UseSingleFile: true,
			},
		}

		// Run our code generation:
		code, err := Generate([]byte(testDocument), opts)
		require.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.Contains(t, code.GetCombined(), `type CatDeadCause string`)
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
		require.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.NotContains(t, code.GetCombined(), `"/cat"`)
	})

	t.Run("examples removed", func(t *testing.T) {
		contents, err := os.ReadFile("testdata/with-examples.yml")
		require.NoError(t, err)

		opts := Configuration{
			PackageName: packageName,
			Output: &Output{
				UseSingleFile: true,
			},
		}

		// Run our code generation:
		code, err := Generate(contents, opts)
		require.NoError(t, err)
		assert.NotEmpty(t, code)

		combined := code.GetCombined()
		assert.Contains(t, combined, `type ExtraA = string`)
		assert.Contains(t, combined, `type ExtraB = string`)
	})
}

//go:build integration
// +build integration

package integration

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/doordash/oapi-codegen/v3/pkg/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_fromURLs(t *testing.T) {
	urls := map[string]string{
		"adyen":  "https://raw.githubusercontent.com/Adyen/adyen-openapi/main/yaml/CheckoutService-v71.yaml",
		"stripe": "https://raw.githubusercontent.com/stripe/openapi/refs/heads/master/openapi/spec3.yaml",
	}
	cfg := codegen.NewDefaultConfiguration()

	for name, url := range urls {
		t.Run(fmt.Sprintf("test-%s", name), func(t *testing.T) {
			t.Parallel()

			fmt.Printf("[%s] Downloading file from %s\n", name, url)
			contents, err := downloadFile(url)
			if err != nil {
				t.Fatalf("failed to download file: %s", err)
			}

			fmt.Printf("[%s] Generating code\n", name)
			res, err := codegen.Generate(contents, cfg)
			require.NoError(t, err, "failed to generate code")
			require.NotNil(t, res, "result should not be nil")

			assert.NotNil(t, res["client"])
			assert.NotNil(t, res["client_options"])
			assert.NotNil(t, res["types"])

		})
	}

	files := map[string]string{
		"train-travel-api": "../testdata/train-travel-api.yml",
	}
	for name, filePath := range files {
		t.Run(fmt.Sprintf("test-%s", name), func(t *testing.T) {
			t.Parallel()

			fmt.Printf("[%s] Opening file from %s\n", name, filePath)
			contents, err := getFileContents(filePath)
			if err != nil {
				t.Fatalf("failed to download file: %s", err)
			}

			fmt.Printf("[%s] Generating code\n", name)
			res, err := codegen.Generate(contents, cfg)
			require.NoError(t, err, "failed to generate code")
			require.NotNil(t, res, "result should not be nil")

			assert.NotNil(t, res["client"])
			assert.NotNil(t, res["client_options"])
			assert.NotNil(t, res["types"])

		})
	}
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: %s (status code: %d)", url, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func getFileContents(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return contents, nil
}

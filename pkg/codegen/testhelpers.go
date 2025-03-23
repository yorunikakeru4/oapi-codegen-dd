package codegen

import (
	"fmt"
	"net/url"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pb33f/libopenapi"
)

func LoadSwagger(filePath string) (swagger *openapi3.T, err error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	u, err := url.Parse(filePath)
	if err == nil && u.Scheme != "" && u.Host != "" {
		return loader.LoadFromURI(u)
	} else {
		return loader.LoadFromFile(filePath)
	}
}

func LoadDocument(filepath string) (libopenapi.Document, error) {
	contents, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	doc, err := libopenapi.NewDocument(contents)
	if err != nil {
		return nil, fmt.Errorf("error creating document: %w", err)
	}
	return doc, nil
}

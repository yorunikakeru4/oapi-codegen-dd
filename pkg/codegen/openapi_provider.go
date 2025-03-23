package codegen

import (
	"fmt"
	"os"

	"github.com/pb33f/libopenapi"
)

func loadDocumentFromFile(filepath string) (libopenapi.Document, error) {
	contents, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return loadDocumentFromContents(contents)
}

func loadDocumentFromContents(contents []byte) (libopenapi.Document, error) {
	doc, err := libopenapi.NewDocument(contents)
	if err != nil {
		return nil, fmt.Errorf("error creating document: %w", err)
	}
	return doc, nil
}

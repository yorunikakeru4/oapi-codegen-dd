package codegen

import (
	"fmt"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
)

func loadDocumentFromContents(contents []byte) (libopenapi.Document, error) {
	docConfig := &datamodel.DocumentConfiguration{
		SkipCircularReferenceCheck: true,
	}
	doc, err := libopenapi.NewDocumentWithConfiguration(contents, docConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating document: %w", err)
	}
	return doc, nil
}

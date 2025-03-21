package codegen

import (
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

// ParseContext holds the OpenAPI models.
type ParseContext struct {
	Operations               []OperationDefinition
	TypeDefinitions          map[SpecLocation][]TypeDefinition
	Enums                    []EnumDefinition
	UnionTypes               []TypeDefinition
	AdditionalTypes          []TypeDefinition
	UnionWithAdditionalTypes []TypeDefinition
	Imports                  []string
}

// CreateParseContext creates a ParseContext from an OpenAPI file and a ParseConfig.
func CreateParseContext(file string, cfg *Configuration) (*ParseContext, []error) {
	if cfg == nil {
		cfg = NewDefaultConfiguration()
	}

	contents, err := os.ReadFile(file)
	if err != nil {
		return nil, []error{err}
	}

	doc, err := openapi3.NewLoader().LoadFromData(contents)
	if err != nil {
		return nil, []error{err}
	}

	res, err := createParseContextFromDocument(doc, cfg)
	if err != nil {
		return nil, []error{err}
	}

	return res, nil
}

func createParseContextFromDocument(doc *openapi3.T, cfg *Configuration) (*ParseContext, error) {
	_, err := filterDocument(doc, cfg)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

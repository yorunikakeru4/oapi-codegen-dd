package codegen

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// goImport represents a go package to be imported in the generated code
type goImport struct {
	Name string // package name
	Path string // package path
}

// String returns a go import statement
func (gi goImport) String() string {
	if gi.Name != "" {
		return fmt.Sprintf("%s %q", gi.Name, gi.Path)
	}
	return fmt.Sprintf("%q", gi.Path)
}

// importMap maps external OpenAPI specifications files/urls to external go packages
type importMap map[string]goImport

// importMappingCurrentPackage allows an Import Mapping to map to the current package, rather than an external package.
// This allows users to split their OpenAPI specification across multiple files, but keep them in the same package, which can reduce a bit of the overhead for users.
// We use `-` to indicate that this is a bit of a special case
const importMappingCurrentPackage = "-"

// GoImports returns a slice of go import statements
func (im importMap) GoImports() []string {
	goImports := make([]string, 0, len(im))
	for _, v := range im {
		if v.Path == importMappingCurrentPackage {
			continue
		}
		goImports = append(goImports, v.String())
	}
	return goImports
}

func collectSchemaImports(s GoSchema) (map[string]goImport, error) {
	res := map[string]goImport{}

	for _, p := range s.Properties {
		imprts, err := GoSchemaImports(p.Schema.OpenAPISchema)
		if err != nil {
			return nil, err
		}
		MergeImports(res, imprts)
	}

	imprts, err := GoSchemaImports(s.OpenAPISchema)
	if err != nil {
		return nil, err
	}
	MergeImports(res, imprts)
	return res, nil
}

func parseGoImportExtension(v *openapi3.Schema) (*goImport, error) {
	if v.Extensions[extPropGoImport] == nil || v.Extensions[extPropGoType] == nil {
		return nil, nil
	}

	goTypeImportExt := v.Extensions[extPropGoImport]

	importI, ok := goTypeImportExt.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to convert type: %T", goTypeImportExt)
	}

	gi := goImport{}
	// replicate the case-insensitive field mapping json.Unmarshal would do
	for k, v := range importI {
		if strings.EqualFold(k, "name") {
			if vs, ok := v.(string); ok {
				gi.Name = vs
			} else {
				return nil, fmt.Errorf("failed to convert type: %T", v)
			}
		} else if strings.EqualFold(k, "path") {
			if vs, ok := v.(string); ok {
				gi.Path = vs
			} else {
				return nil, fmt.Errorf("failed to convert type: %T", v)
			}
		}
	}

	return &gi, nil
}

func GoSchemaImports(schema *openapi3.Schema) (map[string]goImport, error) {
	res := map[string]goImport{}
	if schema == nil {
		return nil, nil
	}

	if gi, err := parseGoImportExtension(schema); err != nil {
		return nil, err
	} else {
		if gi != nil {
			res[gi.String()] = *gi
		}
	}

	t := schema.Type
	if t.Slice() == nil || t.Is("object") {
		for _, v := range schema.Properties {
			imprts, err := GoSchemaImports(v.Value)
			if err != nil {
				return nil, err
			}
			MergeImports(res, imprts)
		}
	} else if t.Is("array") && schema.Items != nil {
		imprts, err := GoSchemaImports(schema.Items.Value)
		if err != nil {
			return nil, err
		}
		MergeImports(res, imprts)
	}
	return res, nil
}

func MergeImports(dst, src map[string]goImport) {
	for k, v := range src {
		dst[k] = v
	}
}

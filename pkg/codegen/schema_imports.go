package codegen

import (
	"fmt"
	"strings"

	"github.com/doordash/oapi-codegen/v2/pkg/util"
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

func OperationSchemaImports(s *Schema) (map[string]goImport, error) {
	res := map[string]goImport{}

	for _, p := range s.Properties {
		imprts, err := GoSchemaImports(&openapi3.SchemaRef{Value: p.Schema.OAPISchema})
		if err != nil {
			return nil, err
		}
		MergeImports(res, imprts)
	}

	imprts, err := GoSchemaImports(&openapi3.SchemaRef{Value: s.OAPISchema})
	if err != nil {
		return nil, err
	}
	MergeImports(res, imprts)
	return res, nil
}

func ParseGoImportExtension(v *openapi3.SchemaRef) (*goImport, error) {
	if v.Value.Extensions[extPropGoImport] == nil || v.Value.Extensions[extPropGoType] == nil {
		return nil, nil
	}

	goTypeImportExt := v.Value.Extensions[extPropGoImport]

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

func GoSchemaImports(schemas ...*openapi3.SchemaRef) (map[string]goImport, error) {
	res := map[string]goImport{}
	for _, sref := range schemas {
		if sref == nil || sref.Value == nil || IsGoTypeReference(sref.Ref) {
			return nil, nil
		}
		if gi, err := ParseGoImportExtension(sref); err != nil {
			return nil, err
		} else {
			if gi != nil {
				res[gi.String()] = *gi
			}
		}
		schemaVal := sref.Value

		t := schemaVal.Type
		if t.Slice() == nil || t.Is("object") {
			for _, v := range schemaVal.Properties {
				imprts, err := GoSchemaImports(v)
				if err != nil {
					return nil, err
				}
				MergeImports(res, imprts)
			}
		} else if t.Is("array") {
			imprts, err := GoSchemaImports(schemaVal.Items)
			if err != nil {
				return nil, err
			}
			MergeImports(res, imprts)
		}
	}
	return res, nil
}

func OperationImports(ops []OperationDefinition) (map[string]goImport, error) {
	res := map[string]goImport{}
	for _, op := range ops {
		for _, pd := range [][]ParameterDefinition{op.PathParams, op.QueryParams} {
			for _, p := range pd {
				imprts, err := OperationSchemaImports(&p.Schema)
				if err != nil {
					return nil, err
				}
				MergeImports(res, imprts)
			}
		}

		for _, b := range op.Bodies {
			imprts, err := OperationSchemaImports(&b.Schema)
			if err != nil {
				return nil, err
			}
			MergeImports(res, imprts)
		}

		for _, b := range op.Responses {
			for _, c := range b.Contents {
				imprts, err := OperationSchemaImports(&c.Schema)
				if err != nil {
					return nil, err
				}
				MergeImports(res, imprts)
			}
		}

	}
	return res, nil
}

func GetTypeDefinitionsImports(swagger *openapi3.T) (map[string]goImport, error) {
	res := map[string]goImport{}
	if swagger.Components == nil {
		return res, nil
	}

	schemaImports, err := GetSchemaImports(swagger.Components.Schemas)
	if err != nil {
		return nil, err
	}

	reqBodiesImports, err := GetRequestBodiesImports(swagger.Components.RequestBodies)
	if err != nil {
		return nil, err
	}

	responsesImports, err := GetResponsesImports(swagger.Components.Responses)
	if err != nil {
		return nil, err
	}

	parametersImports, err := GetParametersImports(swagger.Components.Parameters)
	if err != nil {
		return nil, err
	}

	for _, imprts := range []map[string]goImport{schemaImports, reqBodiesImports, responsesImports, parametersImports} {
		MergeImports(res, imprts)
	}
	return res, nil
}

func GetSchemaImports(schemas map[string]*openapi3.SchemaRef) (map[string]goImport, error) {
	res := map[string]goImport{}
	for _, schema := range schemas {
		imprts, err := GoSchemaImports(schema)
		if err != nil {
			return nil, err
		}
		MergeImports(res, imprts)
	}
	return res, nil
}

func GetRequestBodiesImports(bodies map[string]*openapi3.RequestBodyRef) (map[string]goImport, error) {
	res := map[string]goImport{}
	for _, r := range bodies {
		response := r.Value
		for mediaType, body := range response.Content {
			if !util.IsMediaTypeJson(mediaType) {
				continue
			}

			imprts, err := GoSchemaImports(body.Schema)
			if err != nil {
				return nil, err
			}
			MergeImports(res, imprts)
		}
	}
	return res, nil
}

func GetResponsesImports(responses map[string]*openapi3.ResponseRef) (map[string]goImport, error) {
	res := map[string]goImport{}
	for _, r := range responses {
		response := r.Value
		for mediaType, body := range response.Content {
			if !util.IsMediaTypeJson(mediaType) {
				continue
			}

			imprts, err := GoSchemaImports(body.Schema)
			if err != nil {
				return nil, err
			}
			MergeImports(res, imprts)
		}
	}
	return res, nil
}

func GetParametersImports(params map[string]*openapi3.ParameterRef) (map[string]goImport, error) {
	res := map[string]goImport{}
	for _, param := range params {
		if param.Value == nil {
			continue
		}
		imprts, err := GoSchemaImports(param.Value.Schema)
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

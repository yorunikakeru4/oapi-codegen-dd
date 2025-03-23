package codegen

import (
	"fmt"
	"strconv"

	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

const (
	// extPropGoType overrides the generated type definition.
	extPropGoType = "x-go-type"
	// extPropGoTypeSkipOptionalPointer specifies that optional fields should
	// be the type itself instead of a pointer to the type.
	extPropGoTypeSkipOptionalPointer = "x-go-type-skip-optional-pointer"
	// extPropGoImport specifies the module to import which provides above type
	extPropGoImport = "x-go-type-import"
	// extGoName is used to override a field name
	extGoName = "x-go-name"
	// extGoTypeName is used to override a generated typename for something.
	extGoTypeName        = "x-go-type-name"
	extPropGoJsonIgnore  = "x-go-json-ignore"
	extPropOmitEmpty     = "x-omitempty"
	extPropExtraTags     = "x-oapi-codegen-extra-tags"
	extEnumVarNames      = "x-enum-varnames"
	extEnumNames         = "x-enumNames"
	extDeprecationReason = "x-deprecated-reason"
)

func extString(extPropValue any) (string, error) {
	str, ok := extPropValue.(string)
	if !ok {
		return "", fmt.Errorf("failed to convert type: %T", extPropValue)
	}
	return str, nil
}

func extParsePropGoTypeSkipOptionalPointer(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, fmt.Errorf("failed to convert type: %T", value)
		}
		return b, nil
	}
	return false, fmt.Errorf("failed to convert type: %T", value)
}

func extParseGoFieldName(extPropValue any) (string, error) {
	return extString(extPropValue)
}

func extParseOmitEmpty(extPropValue any) (bool, error) {
	omitEmpty, ok := extPropValue.(bool)
	if !ok {
		return false, fmt.Errorf("failed to convert type: %T", extPropValue)
	}
	return omitEmpty, nil
}

func extExtraTags(extPropValue any) (map[string]string, error) {
	tagsI, ok := extPropValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to convert type: %T", extPropValue)
	}

	tags := make(map[string]string, len(tagsI))
	for k, v := range tagsI {
		vs, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("failed to convert type: %T", v)
		}
		tags[k] = vs
	}
	return tags, nil
}

func extParseGoJsonIgnore(extPropValue interface{}) (bool, error) {
	goJsonIgnore, ok := extPropValue.(bool)
	if !ok {
		return false, fmt.Errorf("failed to convert type: %T", extPropValue)
	}
	return goJsonIgnore, nil
}

func extParseEnumVarNames(extPropValue interface{}) ([]string, error) {
	namesI, ok := extPropValue.([]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to convert type: %T", extPropValue)
	}
	names := make([]string, len(namesI))
	for i, v := range namesI {
		vs, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("failed to convert type: %T", v)
		}
		names[i] = vs
	}
	return names, nil
}

func extParseDeprecationReason(extPropValue interface{}) (string, error) {
	return extString(extPropValue)
}

func extractExtensions(schemaExtensions *orderedmap.Map[string, *yaml.Node]) map[string]any {
	if schemaExtensions == nil || schemaExtensions.Len() == 0 {
		return nil
	}

	res := make(map[string]any)

	for extType, node := range schemaExtensions.FromOldest() {
		res[extType] = make(map[string]any)
		if node.Kind == yaml.ScalarNode {
			res[extType] = node.Value
			continue
		}

		if node.Kind != yaml.MappingNode {
			continue
		}

		var k string
		inner := make(map[string]any)
		for i, n := range node.Content {
			if i%2 == 0 {
				k = n.Value
			} else {
				if k == "" {
					continue
				}
				v := n.Value
				println(v)
				inner[k] = v
				k = ""
			}
		}
		res[extType] = inner
	}
	return res
}

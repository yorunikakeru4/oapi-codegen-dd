package codegen

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Schema describes an OpenAPI schema, with lots of helper fields to use in the
// templating engine.
type Schema struct {
	GoType  string // The Go type needed to represent the schema
	RefType string // If the type has a type name, this is set

	ArrayType *Schema // The schema of array element

	EnumValues map[string]string // Enum values

	Properties               []Property       // For an object, the fields with names
	HasAdditionalProperties  bool             // Whether we support additional properties
	AdditionalPropertiesType *Schema          // And if we do, their type
	AdditionalTypes          []TypeDefinition // We may need to generate auxiliary helper types, stored here

	SkipOptionalPointer bool // Some types don't need a * in front when they're optional

	Description string // The description of the element

	UnionElements []UnionElement // Possible elements of oneOf/anyOf union
	Discriminator *Discriminator // Describes which value is stored in a union

	// If this is set, the schema will declare a type via alias, eg,
	// `type Foo = bool`. If this is not set, we will define this type via
	// type definition `type Foo bool`
	//
	// Can be overriden by the config#DisableTypeAliasesForType field
	DefineViaAlias bool

	// The original OpenAPIv3 Schema.
	OAPISchema *openapi3.Schema
}

func (s Schema) IsRef() bool {
	return s.RefType != ""
}

func (s Schema) IsExternalRef() bool {
	if !s.IsRef() {
		return false
	}
	return strings.Contains(s.RefType, ".")
}

func (s Schema) TypeDecl() string {
	if s.IsRef() {
		return s.RefType
	}
	return s.GoType
}

// AddProperty adds a new property to the current Schema, and returns an error
// if it collides. Two identical fields will not collide, but two properties by
// the same name, but different definition, will collide. It's safe to merge the
// fields of two schemas with overlapping properties if those properties are
// identical.
func (s Schema) AddProperty(p Property) error {
	// Scan all existing properties for a conflict
	for _, e := range s.Properties {
		if e.JsonFieldName == p.JsonFieldName && !PropertiesEqual(e, p) {
			return fmt.Errorf("property '%s' already exists with a different type", e.JsonFieldName)
		}
	}
	s.Properties = append(s.Properties, p)
	return nil
}

func (s Schema) GetAdditionalTypeDefs() []TypeDefinition {
	return s.AdditionalTypes
}

func (s Schema) createGoStruct(fields []string) string {
	// Start out with struct {
	objectParts := []string{"struct {"}

	// Append all the field definitions
	objectParts = append(objectParts, fields...)

	// Close the struct
	if s.HasAdditionalProperties {
		objectParts = append(objectParts,
			fmt.Sprintf("AdditionalProperties map[string]%s `json:\"-\"`",
				additionalPropertiesType(s)))
	}

	if len(s.UnionElements) != 0 {
		objectParts = append(objectParts, "union json.RawMessage")
	}

	objectParts = append(objectParts, "}")
	return strings.Join(objectParts, "\n")
}

type Constants struct {
	// EnumDefinitions holds type and value information for all enums
	EnumDefinitions []EnumDefinition
}

type Discriminator struct {
	// maps discriminator value to go type
	Mapping map[string]string

	// JSON property name that holds the discriminator
	Property string
}

func (d *Discriminator) JSONTag() string {
	return fmt.Sprintf("`json:\"%s\"`", d.Property)
}

func (d *Discriminator) PropertyName() string {
	return SchemaNameToTypeName(d.Property)
}

func PropertiesEqual(a, b Property) bool {
	return a.JsonFieldName == b.JsonFieldName && a.Schema.TypeDecl() == b.Schema.TypeDecl() && a.Required == b.Required
}

// SchemaDescriptor describes a Schema, a type definition.
type SchemaDescriptor struct {
	Fields                   []FieldDescriptor
	HasAdditionalProperties  bool
	AdditionalPropertiesType string
}

type FieldDescriptor struct {
	Required bool   // Is the schema required? If not, we'll pass by pointer
	GoType   string // The Go type needed to represent the json type.
	GoName   string // The Go compatible type name for the type
	JsonName string // The json type name for the type
	IsRef    bool   // Is this schema a reference to predefined object?
}

func additionalPropertiesType(schema Schema) string {
	addPropsType := schema.AdditionalPropertiesType.GoType
	if schema.AdditionalPropertiesType.RefType != "" {
		addPropsType = schema.AdditionalPropertiesType.RefType
	}
	if schema.AdditionalPropertiesType.OAPISchema != nil && schema.AdditionalPropertiesType.OAPISchema.Nullable {
		addPropsType = "*" + addPropsType
	}
	return addPropsType
}

func GenStructFromSchema(schema Schema) string {
	// Start out with struct {
	objectParts := []string{"struct {"}
	// Append all the field definitions
	objectParts = append(objectParts, GenFieldsFromProperties(schema.Properties)...)
	// Close the struct
	if schema.HasAdditionalProperties {
		objectParts = append(objectParts,
			fmt.Sprintf("AdditionalProperties map[string]%s `json:\"-\"`",
				additionalPropertiesType(schema)))
	}
	if len(schema.UnionElements) != 0 {
		objectParts = append(objectParts, "union json.RawMessage")
	}
	objectParts = append(objectParts, "}")
	return strings.Join(objectParts, "\n")
}

// This constructs a Go type for a parameter, looking at either the schema or
// the content, whichever is available
func paramToGoType(param *openapi3.Parameter, path []string) (Schema, error) {
	if param.Content == nil && param.Schema == nil {
		return Schema{}, fmt.Errorf("parameter '%s' has no schema or content", param.Name)
	}

	// We can process the schema through the generic schema processor
	if param.Schema != nil {
		return GenerateGoSchema(param.Schema, path)
	}

	// At this point, we have a content type. We know how to deal with
	// application/json, but if multiple formats are present, we can't do anything,
	// so we'll return the parameter as a string, not bothering to decode it.
	if len(param.Content) > 1 {
		return Schema{
			GoType:      "string",
			Description: StringToGoComment(param.Description),
		}, nil
	}

	// Otherwise, look for application/json in there
	mt, found := param.Content["application/json"]
	if !found {
		// If we don't have json, it's a string
		return Schema{
			GoType:      "string",
			Description: StringToGoComment(param.Description),
		}, nil
	}

	// For json, we go through the standard schema mechanism
	return GenerateGoSchema(mt.Schema, path)
}

package codegen

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// GoSchema describes an OpenAPI schema, with lots of helper fields to use in the templating engine.
// GoType is the Go type that represents the schema.
// RefType is the type name of the schema, if it has one.
// ArrayType is the schema of the array element, if it's an array.
// EnumValues is a map of enum values.
// Properties is a list of fields for an object.
// HasAdditionalProperties is true if the object has additional properties.
// AdditionalPropertiesType is the type of additional properties.
// AdditionalTypes is a list of auxiliary types that may be needed.
// SkipOptionalPointer is true if the type doesn't need a * in front when it's optional.
// Description is the description of the element.
// Constraints is a struct that holds constraints for the schema.
// UnionElements is a list of possible elements in a oneOf/anyOf union.
// Discriminator describes which value is stored in a union.
// DefineViaAlias is true if the schema should be declared via alias.
type GoSchema struct {
	GoType                   string
	RefType                  string
	ArrayType                *GoSchema
	EnumValues               map[string]string
	Properties               []Property
	HasAdditionalProperties  bool
	AdditionalPropertiesType *GoSchema
	AdditionalTypes          []TypeDefinition
	SkipOptionalPointer      bool
	Description              string
	Constraints              Constraints

	UnionElements []UnionElement
	Discriminator *Discriminator

	DefineViaAlias bool
	OpenAPISchema  *openapi3.Schema
}

func (s GoSchema) IsRef() bool {
	return s.RefType != ""
}

func (s GoSchema) IsExternalRef() bool {
	if !s.IsRef() {
		return false
	}
	return strings.Contains(s.RefType, ".")
}

func (s GoSchema) TypeDecl() string {
	if s.IsRef() {
		return s.RefType
	}
	return s.GoType
}

// AddProperty adds a new property to the current GoSchema, and returns an error
// if it collides. Two identical fields will not collide, but two properties by
// the same name, but different definition, will collide. It's safe to merge the
// fields of two schemas with overlapping properties if those properties are
// identical.
func (s GoSchema) AddProperty(p Property) error {
	// Scan all existing properties for a conflict
	for _, e := range s.Properties {
		if e.JsonFieldName == p.JsonFieldName && !PropertiesEqual(e, p) {
			return fmt.Errorf("property '%s' already exists with a different type", e.JsonFieldName)
		}
	}
	s.Properties = append(s.Properties, p)
	return nil
}

func (s GoSchema) GetAdditionalTypeDefs() []TypeDefinition {
	return s.AdditionalTypes
}

func (s GoSchema) createGoStruct(fields []string) string {
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

func GenerateGoSchema(sref *openapi3.SchemaRef, path []string) (GoSchema, error) {
	// Add a fallback value in case the sref is nil.
	// i.e. the parent schema defines a type:array, but the array has
	// no items defined. Therefore, we have at least valid Go-Code.
	if sref == nil {
		return GoSchema{GoType: "any"}, nil
	}

	schema := sref.Value

	// If Ref is set on the SchemaRef, it means that this type is actually a reference to
	// another type. We're not de-referencing, so simply use the referenced type.
	if IsGoTypeReference(sref.Ref) {
		// Convert the reference path to Go type
		refType, err := RefPathToGoType(sref.Ref)
		if err != nil {
			return GoSchema{}, fmt.Errorf("error turning reference (%s) into a Go type: %s",
				sref.Ref, err)
		}
		return GoSchema{
			GoType:         refType,
			Description:    schema.Description,
			DefineViaAlias: true,
			OpenAPISchema:  schema,
		}, nil
	}

	outSchema := GoSchema{
		Description:   schema.Description,
		OpenAPISchema: schema,
	}

	// AllOf is interesting, and useful. It's the union of a number of other
	// schemas. A common usage is to create a union of an object with an ID,
	// so that in a RESTful paradigm, the Create operation can return
	// (object, id), so that other operations can refer to (id)
	if schema.AllOf != nil {
		mergedSchema, err := MergeSchemas(schema.AllOf, path)
		if err != nil {
			return GoSchema{}, fmt.Errorf("error merging schemas: %w", err)
		}
		mergedSchema.OpenAPISchema = schema
		return mergedSchema, nil
	}

	// Check x-go-type, which will completely override the definition of this
	// schema with the provided type.
	if extension, ok := schema.Extensions[extPropGoType]; ok {
		typeName, err := extTypeName(extension)
		if err != nil {
			return outSchema, fmt.Errorf("invalid value for %q: %w", extPropGoType, err)
		}
		outSchema.GoType = typeName
		outSchema.DefineViaAlias = true

		return outSchema, nil
	}

	// Check x-go-type-skip-optional-pointer, which will override if the type
	// should be a pointer or not when the field is optional.
	if extension, ok := schema.Extensions[extPropGoTypeSkipOptionalPointer]; ok {
		skipOptionalPointer, err := extParsePropGoTypeSkipOptionalPointer(extension)
		if err != nil {
			return outSchema, fmt.Errorf("invalid value for %q: %w", extPropGoTypeSkipOptionalPointer, err)
		}
		outSchema.SkipOptionalPointer = skipOptionalPointer
	}

	// GoSchema type and format, eg. string / binary
	t := schema.Type
	// Handle objects and empty schemas first as a special case
	if t.Slice() == nil || t.Is("object") {
		return createObjectSchema(schema, sref.Ref, path)
	}

	if len(schema.Enum) > 0 {
		return createEnumsSchema(schema, path)
	}

	res, err := oapiSchemaToGoType(schema, path)
	if err != nil {
		return GoSchema{}, fmt.Errorf("error resolving primitive type: %w", err)
	}
	return res, nil
}

func PropertiesEqual(a, b Property) bool {
	return a.JsonFieldName == b.JsonFieldName && a.Schema.TypeDecl() == b.Schema.TypeDecl() && a.Required == b.Required
}

// SchemaDescriptor describes a GoSchema, a type definition.
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

func additionalPropertiesType(schema GoSchema) string {
	addPropsType := schema.AdditionalPropertiesType.GoType
	if schema.AdditionalPropertiesType.RefType != "" {
		addPropsType = schema.AdditionalPropertiesType.RefType
	}
	if schema.AdditionalPropertiesType.OpenAPISchema != nil && schema.AdditionalPropertiesType.OpenAPISchema.Nullable {
		addPropsType = "*" + addPropsType
	}
	return addPropsType
}

func GenStructFromSchema(schema GoSchema) string {
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

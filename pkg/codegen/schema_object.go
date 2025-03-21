package codegen

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func GenerateGoSchema(sref *openapi3.SchemaRef, path []string) (Schema, error) {
	// Add a fallback value in case the sref is nil.
	// i.e. the parent schema defines a type:array, but the array has
	// no items defined. Therefore, we have at least valid Go-Code.
	if sref == nil {
		return Schema{GoType: "interface{}"}, nil
	}

	schema := sref.Value

	// If Ref is set on the SchemaRef, it means that this type is actually a reference to
	// another type. We're not de-referencing, so simply use the referenced type.
	if IsGoTypeReference(sref.Ref) {
		// Convert the reference path to Go type
		refType, err := RefPathToGoType(sref.Ref)
		if err != nil {
			return Schema{}, fmt.Errorf("error turning reference (%s) into a Go type: %s",
				sref.Ref, err)
		}
		return Schema{
			GoType:         refType,
			Description:    schema.Description,
			DefineViaAlias: true,
			OAPISchema:     schema,
		}, nil
	}

	outSchema := Schema{
		Description: schema.Description,
		OAPISchema:  schema,
	}

	// AllOf is interesting, and useful. It's the union of a number of other
	// schemas. A common usage is to create a union of an object with an ID,
	// so that in a RESTful paradigm, the Create operation can return
	// (object, id), so that other operations can refer to (id)
	if schema.AllOf != nil {
		mergedSchema, err := MergeSchemas(schema.AllOf, path)
		if err != nil {
			return Schema{}, fmt.Errorf("error merging schemas: %w", err)
		}
		mergedSchema.OAPISchema = schema
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

	// Schema type and format, eg. string / binary
	t := schema.Type
	// Handle objects and empty schemas first as a special case
	if t.Slice() == nil || t.Is("object") {
		var outType string

		if len(schema.Properties) == 0 && !SchemaHasAdditionalProperties(schema) && schema.AnyOf == nil && schema.OneOf == nil {
			// If the object has no properties or additional properties, we
			// have some special cases for its type.
			if t.Is("object") {
				// We have an object with no properties. This is a generic object
				// expressed as a map.
				outType = "map[string]interface{}"
			} else { // t == ""
				// If we don't even have the object designator, we're a completely
				// generic type.
				outType = "interface{}"
			}
			outSchema.GoType = outType
			outSchema.DefineViaAlias = true
		} else {
			// When we define an object, we want it to be a type definition,
			// not a type alias, eg, "type Foo struct {...}"
			outSchema.DefineViaAlias = false

			// If the schema has additional properties, we need to special case
			// a lot of behaviors.
			outSchema.HasAdditionalProperties = SchemaHasAdditionalProperties(schema)

			// Until we have a concrete additional properties type, we default to
			// any schema.
			outSchema.AdditionalPropertiesType = &Schema{
				GoType: "interface{}",
			}

			// If additional properties are defined, we will override the default
			// above with the specific definition.
			if schema.AdditionalProperties.Schema != nil {
				additionalSchema, err := GenerateGoSchema(schema.AdditionalProperties.Schema, path)
				if err != nil {
					return Schema{}, fmt.Errorf("error generating type for additional properties: %w", err)
				}
				if additionalSchema.HasAdditionalProperties || len(additionalSchema.UnionElements) != 0 {
					// If we have fields present which have additional properties or union values,
					// but are not a pre-defined type, we need to define a type
					// for them, which will be based on the field names we followed
					// to get to the type.
					typeName := PathToTypeName(append(path, "AdditionalProperties"))

					typeDef := TypeDefinition{
						TypeName: typeName,
						JsonName: strings.Join(append(path, "AdditionalProperties"), "."),
						Schema:   additionalSchema,
					}
					additionalSchema.RefType = typeName
					additionalSchema.AdditionalTypes = append(additionalSchema.AdditionalTypes, typeDef)
				}
				outSchema.AdditionalPropertiesType = &additionalSchema
				outSchema.AdditionalTypes = append(outSchema.AdditionalTypes, additionalSchema.AdditionalTypes...)
			}

			// If the schema has no properties, and only additional properties, we will
			// early-out here and generate a map[string]<schema> instead of an object
			// that contains this map. We skip over anyOf/oneOf here because they can
			// introduce properties. allOf was handled above.
			if len(schema.Properties) == 0 && schema.AnyOf == nil && schema.OneOf == nil {
				// We have a dictionary here. Returns the goType to be just a map from
				// string to the property type. HasAdditionalProperties=false means
				// that we won't generate custom json.Marshaler and json.Unmarshaler functions,
				// since we don't need them for a simple map.
				outSchema.HasAdditionalProperties = false
				outSchema.GoType = fmt.Sprintf("map[string]%s", additionalPropertiesType(outSchema))
				return outSchema, nil
			}

			// We've got an object with some properties.
			for _, pName := range SortedSchemaKeys(schema.Properties) {
				p := schema.Properties[pName]
				propertyPath := append(path, pName)
				pSchema, err := GenerateGoSchema(p, propertyPath)
				if err != nil {
					return Schema{}, fmt.Errorf("error generating Go schema for property '%s': %w", pName, err)
				}

				required := StringInArray(pName, schema.Required)

				if (pSchema.HasAdditionalProperties || len(pSchema.UnionElements) != 0) && pSchema.RefType == "" {
					// If we have fields present which have additional properties or union values,
					// but are not a pre-defined type, we need to define a type
					// for them, which will be based on the field names we followed
					// to get to the type.
					typeName := PathToTypeName(propertyPath)

					typeDef := TypeDefinition{
						TypeName: typeName,
						JsonName: strings.Join(propertyPath, "."),
						Schema:   pSchema,
					}
					pSchema.AdditionalTypes = append(pSchema.AdditionalTypes, typeDef)

					pSchema.RefType = typeName
				}
				description := ""
				if p.Value != nil {
					description = p.Value.Description
				}
				prop := Property{
					// GoName: createPropertyGoFieldName(pName, extensions),
					JsonFieldName: pName,
					Schema:        pSchema,
					Required:      required,
					Description:   description,
					Nullable:      p.Value.Nullable,
					ReadOnly:      p.Value.ReadOnly,
					WriteOnly:     p.Value.WriteOnly,
					Extensions:    p.Value.Extensions,
					Deprecated:    p.Value.Deprecated,
				}
				outSchema.Properties = append(outSchema.Properties, prop)
				if len(pSchema.AdditionalTypes) > 0 {
					outSchema.AdditionalTypes = append(outSchema.AdditionalTypes, pSchema.AdditionalTypes...)
				}
			}

			if schema.AnyOf != nil {
				if err := generateUnion(&outSchema, schema.AnyOf, schema.Discriminator, path); err != nil {
					return Schema{}, fmt.Errorf("error generating type for anyOf: %w", err)
				}
			}
			if schema.OneOf != nil {
				if err := generateUnion(&outSchema, schema.OneOf, schema.Discriminator, path); err != nil {
					return Schema{}, fmt.Errorf("error generating type for oneOf: %w", err)
				}
			}

			outSchema.GoType = GenStructFromSchema(outSchema)
		}

		// Check for x-go-type-name. It behaves much like x-go-type, however, it will
		// create a type definition for the named type, and use the named type in place
		// of this schema.
		if extension, ok := schema.Extensions[extGoTypeName]; ok {
			typeName, err := extTypeName(extension)
			if err != nil {
				return outSchema, fmt.Errorf("invalid value for %q: %w", extGoTypeName, err)
			}

			newTypeDef := TypeDefinition{
				TypeName: typeName,
				Schema:   outSchema,
			}
			outSchema = Schema{
				Description:     newTypeDef.Schema.Description,
				GoType:          typeName,
				DefineViaAlias:  true,
				AdditionalTypes: append(outSchema.AdditionalTypes, newTypeDef),
			}
		}

		return outSchema, nil
	}

	if len(schema.Enum) > 0 {
		return createEnumsSchema(schema, path)
	}

	res, err := oapiSchemaToGoType(schema, path)
	if err != nil {
		return Schema{}, fmt.Errorf("error resolving primitive type: %w", err)
	}
	return res, nil
}

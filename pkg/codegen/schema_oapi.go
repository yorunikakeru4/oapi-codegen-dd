package codegen

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// oapiSchemaToGoType converts an OpenApi schema into a Go type definition for
// all non-object types.
func oapiSchemaToGoType(schema *base.Schema, ref string, path []string) (GoSchema, error) {
	f := schema.Format
	t := schema.Type
	constraints := getSchemaConstraints(schema, ConstraintsContext{
		name:       "",
		hasNilType: slices.Contains(t, "null"),
	})

	if slices.Contains(t, "array") {
		// For arrays, we'll get the type of the Items and throw a
		// [] in front of it.
		var items *base.SchemaProxy
		if schema != nil && schema.Items != nil && schema.Items.IsA() {
			items = schema.Items.A
		}
		arrayType, err := GenerateGoSchema(items, ref, path)
		if err != nil {
			return GoSchema{}, fmt.Errorf("error generating type for array: %w", err)
		}
		if (arrayType.HasAdditionalProperties || len(arrayType.UnionElements) != 0) && arrayType.RefType == "" {
			// If we have items which have additional properties or union values,
			// but are not a pre-defined type, we need to define a type
			// for them, which will be based on the field names we followed
			// to get to the type.
			typeName := pathToTypeName(append(path, "Item"))

			typeDef := TypeDefinition{
				Name:     typeName,
				JsonName: strings.Join(append(path, "Item"), "."),
				Schema:   arrayType,
			}
			arrayType.AdditionalTypes = append(arrayType.AdditionalTypes, typeDef)
			arrayType.RefType = typeName
		}

		defineViaAlias := true
		if disableTypeAliasesForArray {
			defineViaAlias = false
		}

		return GoSchema{
			GoType:          "[]" + arrayType.TypeDecl(),
			ArrayType:       &arrayType,
			AdditionalTypes: arrayType.AdditionalTypes,
			Properties:      arrayType.Properties,
			DefineViaAlias:  defineViaAlias,
			Description:     schema.Description,
			OpenAPISchema:   schema,
			Constraints:     constraints,
		}, nil
	}

	if slices.Contains(t, "integer") {
		// We default to int if format doesn't ask for something else.
		goType := "int"
		switch f {
		case "int64":
			goType = "int64"
		case "int32":
			goType = "int32"
		case "int16":
			goType = "int16"
		case "int8":
			goType = "int8"
		case "uint64":
			goType = "uint64"
		case "uint32":
			goType = "uint32"
		case "uint16":
			goType = "uint16"
		case "uint8":
			goType = "uint8"
		case "uint":
			goType = "uint"
		}

		return GoSchema{
			GoType:         goType,
			DefineViaAlias: true,
			Description:    schema.Description,
			OpenAPISchema:  schema,
			Constraints:    constraints,
		}, nil
	}

	if slices.Contains(t, "number") {
		// We default to float for "number"
		goType := "float32"
		switch f {
		case "double":
			goType = "float64"
		case "float":
			goType = "float32"
		case "":
			goType = "float32"
		default:
			return GoSchema{}, fmt.Errorf("invalid number format: %s", f)
		}

		return GoSchema{
			GoType:         goType,
			DefineViaAlias: true,
			Description:    schema.Description,
			OpenAPISchema:  schema,
			Constraints:    constraints,
		}, nil
	}

	if slices.Contains(t, "boolean") {
		if f != "" {
			return GoSchema{}, fmt.Errorf("invalid format (%s) for boolean", f)
		}
		return GoSchema{
			GoType:         "bool",
			DefineViaAlias: true,
			Description:    schema.Description,
			OpenAPISchema:  schema,
			Constraints:    constraints,
		}, nil
	}

	if slices.Contains(t, "string") {
		// Special case string formats here.
		// All unrecognized formats are simply a regular string.
		goType := "string"
		skipOptionalPointer := false
		switch f {
		case "byte":
			goType = "[]byte"
		case "email":
			goType = "oapi_codegen.Email"
		case "date":
			goType = "oapi_codegen.Date"
		case "date-time":
			goType = "time.Time"
		case "json":
			goType = "runtime.RawMessage"
			skipOptionalPointer = true
		case "binary":
			goType = "oapi_codegen.File"
		case "uuid":
			goType = "uuid.UUID"
		}
		return GoSchema{
			GoType:              goType,
			DefineViaAlias:      true,
			SkipOptionalPointer: skipOptionalPointer,
			Description:         schema.Description,
			OpenAPISchema:       schema,
			Constraints:         constraints,
		}, nil
	}

	if slices.Contains(t, "null") {
		return GoSchema{}, nil
	}

	return GoSchema{}, fmt.Errorf("unhandled GoSchema type: %v", t)
}

func getSchemaConstraints(schema *base.Schema, opts ConstraintsContext) Constraints {
	if schema == nil {
		return Constraints{}
	}

	name := opts.name
	hasNilType := opts.hasNilType

	required := opts.required
	if !required && name != "" {
		required = slices.Contains(schema.Required, name)
	}

	nullable := false
	if !required || hasNilType {
		nullable = true
	} else if schema.Nullable != nil {
		nullable = *schema.Nullable
	}

	if required && nullable {
		nullable = true
	}

	readOnly := false
	if schema.ReadOnly != nil {
		readOnly = *schema.ReadOnly
	}

	writeOnly := false
	if schema.WriteOnly != nil {
		writeOnly = *schema.WriteOnly
	}

	minValue := float64(0)
	if schema.Minimum != nil {
		minValue = *schema.Minimum
	}

	maxValue := float64(0)
	if schema.Maximum != nil {
		maxValue = *schema.Maximum
	}

	minLength := int64(0)
	if schema.MinLength != nil {
		minLength = *schema.MinLength
	}

	maxLength := int64(0)
	if schema.MaxLength != nil {
		maxLength = *schema.MaxLength
	}

	return Constraints{
		Nullable:  nullable,
		Required:  required,
		ReadOnly:  readOnly,
		WriteOnly: writeOnly,
		Min:       minValue,
		Max:       maxValue,
		MinLength: minLength,
		MaxLength: maxLength,
	}
}

package codegen

import (
	"fmt"
	"reflect"
)

// TypeDefinition describes a Go type definition in generated code.
//
// Let's use this example schema:
// components:
//
//	schemas:
//	  Person:
//	    type: object
//	    properties:
//	    name:
//	      type: string
type TypeDefinition struct {
	// The name of the type, eg, type <...> Person
	TypeName string

	// The name of the corresponding JSON description, as it will sometimes
	// differ due to invalid characters.
	JsonName string

	// This is the Schema wrapper is used to populate the type description
	Schema Schema
}

// ResponseTypeDefinition is an extension of TypeDefinition, specifically for
// response unmarshaling in ClientWithResponses.
type ResponseTypeDefinition struct {
	TypeDefinition
	// The content type name where this is used, eg, application/json
	ContentTypeName string

	// The type name of a response model.
	ResponseName string

	AdditionalTypeDefinitions []TypeDefinition
}

func (t *TypeDefinition) IsAlias() bool {
	return t.Schema.DefineViaAlias
}

func checkDuplicates(types []TypeDefinition) ([]TypeDefinition, error) {
	m := map[string]TypeDefinition{}
	var ts []TypeDefinition

	for _, typ := range types {
		if other, found := m[typ.TypeName]; found {
			// If type names collide, we need to see if they refer to the same
			// exact type definition, in which case, we can de-dupe.
			// If they don't match, we error out.
			if TypeDefinitionsEquivalent(other, typ) {
				continue
			}
			// We want to create an error when we try to define the same type twice.
			return nil, fmt.Errorf("duplicate typename '%s' detected, can't auto-rename, "+
				"please use x-go-name to specify your own name for one of them", typ.TypeName)
		}

		m[typ.TypeName] = typ

		ts = append(ts, typ)
	}

	return ts, nil
}

// TypeDefinitionsEquivalent checks for equality between two type definitions, but
// not every field is considered. We only want to know if they are fundamentally
// the same type.
func TypeDefinitionsEquivalent(t1, t2 TypeDefinition) bool {
	if t1.TypeName != t2.TypeName {
		return false
	}
	return reflect.DeepEqual(t1.Schema.OAPISchema, t2.Schema.OAPISchema)
}

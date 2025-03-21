package codegen

import (
	"fmt"

	"github.com/doordash/oapi-codegen/v2/pkg/util"
	"github.com/getkin/kin-openapi/openapi3"
)

// GenerateTypesForSchemas generates type definitions for any custom types defined in the
// components/schemas section of the Swagger spec.
func GenerateTypesForSchemas(schemas map[string]*openapi3.SchemaRef) ([]TypeDefinition, error) {
	types := make([]TypeDefinition, 0)
	// We're going to define Go types for every object under components/schemas
	for _, schemaName := range SortedSchemaKeys(schemas) {
		schemaRef := schemas[schemaName]

		goSchema, err := GenerateGoSchema(schemaRef, []string{schemaName})
		if err != nil {
			return nil, fmt.Errorf("error converting Schema %s to Go type: %w", schemaName, err)
		}

		goTypeName, err := renameSchema(schemaName, schemaRef)
		if err != nil {
			return nil, fmt.Errorf("error making name for components/schemas/%s: %w", schemaName, err)
		}

		types = append(types, TypeDefinition{
			JsonName: schemaName,
			TypeName: goTypeName,
			Schema:   goSchema,
		})

		types = append(types, goSchema.AdditionalTypes...)
	}
	return types, nil
}

// GenerateTypesForParameters generates type definitions for any custom types defined in the
// components/parameters section of the Swagger spec.
func GenerateTypesForParameters(params map[string]*openapi3.ParameterRef) ([]TypeDefinition, error) {
	var types []TypeDefinition
	for _, paramName := range SortedMapKeys(params) {
		paramOrRef := params[paramName]

		goType, err := paramToGoType(paramOrRef.Value, nil)
		if err != nil {
			return nil, fmt.Errorf("error generating Go type for schema in parameter %s: %w", paramName, err)
		}

		goTypeName, err := renameParameter(paramName, paramOrRef)
		if err != nil {
			return nil, fmt.Errorf("error making name for components/parameters/%s: %w", paramName, err)
		}

		typeDef := TypeDefinition{
			JsonName: paramName,
			Schema:   goType,
			TypeName: goTypeName,
		}

		if paramOrRef.Ref != "" {
			// Generate a reference type for referenced parameters
			refType, err := RefPathToGoType(paramOrRef.Ref)
			if err != nil {
				return nil, fmt.Errorf("error generating Go type for (%s) in parameter %s: %w", paramOrRef.Ref, paramName, err)
			}
			typeDef.TypeName = SchemaNameToTypeName(refType)
		}

		types = append(types, typeDef)
	}
	return types, nil
}

// GenerateTypesForRequestBodies generates type definitions for any custom types defined in the
// components/requestBodies section of the Swagger spec.
func GenerateTypesForRequestBodies(bodies map[string]*openapi3.RequestBodyRef) ([]TypeDefinition, error) {
	var types []TypeDefinition

	for _, requestBodyName := range SortedMapKeys(bodies) {
		requestBodyRef := bodies[requestBodyName]

		// As for responses, we will only generate Go code for JSON bodies,
		// the other body formats are up to the user.
		response := requestBodyRef.Value
		for mediaType, body := range response.Content {
			if !util.IsMediaTypeJson(mediaType) {
				continue
			}

			goType, err := GenerateGoSchema(body.Schema, []string{requestBodyName})
			if err != nil {
				return nil, fmt.Errorf("error generating Go type for schema in body %s: %w", requestBodyName, err)
			}

			goTypeName, err := renameRequestBody(requestBodyName, requestBodyRef)
			if err != nil {
				return nil, fmt.Errorf("error making name for components/schemas/%s: %w", requestBodyName, err)
			}

			typeDef := TypeDefinition{
				JsonName: requestBodyName,
				Schema:   goType,
				TypeName: goTypeName,
			}

			if requestBodyRef.Ref != "" {
				// Generate a reference type for referenced bodies
				refType, err := RefPathToGoType(requestBodyRef.Ref)
				if err != nil {
					return nil, fmt.Errorf("error generating Go type for (%s) in body %s: %w", requestBodyRef.Ref, requestBodyName, err)
				}
				typeDef.TypeName = SchemaNameToTypeName(refType)
			}
			types = append(types, typeDef)
		}
	}
	return types, nil
}

// GenerateTypesForResponses generates type definitions for any custom types defined in the
// components/responses section of the Swagger spec.
func GenerateTypesForResponses(responses openapi3.ResponseBodies) ([]TypeDefinition, error) {
	var types []TypeDefinition

	for _, responseName := range SortedMapKeys(responses) {
		responseOrRef := responses[responseName]

		// We have to generate the response object. We're only going to
		// handle media types that conform to JSON. Other responses should
		// simply be specified as strings or byte arrays.
		response := responseOrRef.Value

		jsonCount := 0
		for mediaType := range response.Content {
			if util.IsMediaTypeJson(mediaType) {
				jsonCount++
			}
		}

		SortedMapKeys := SortedMapKeys(response.Content)
		for _, mediaType := range SortedMapKeys {
			response := response.Content[mediaType]
			if !util.IsMediaTypeJson(mediaType) {
				continue
			}

			goType, err := GenerateGoSchema(response.Schema, []string{responseName})
			if err != nil {
				return nil, fmt.Errorf("error generating Go type for schema in response %s: %w", responseName, err)
			}

			goTypeName, err := renameResponse(responseName, responseOrRef)
			if err != nil {
				return nil, fmt.Errorf("error making name for components/responses/%s: %w", responseName, err)
			}

			typeDef := TypeDefinition{
				JsonName: responseName,
				Schema:   goType,
				TypeName: goTypeName,
			}

			if responseOrRef.Ref != "" {
				// Generate a reference type for referenced parameters
				refType, err := RefPathToGoType(responseOrRef.Ref)
				if err != nil {
					return nil, fmt.Errorf("error generating Go type for (%s) in parameter %s: %w", responseOrRef.Ref, responseName, err)
				}
				typeDef.TypeName = SchemaNameToTypeName(refType)
			}

			if jsonCount > 1 {
				typeDef.TypeName = typeDef.TypeName + mediaTypeToCamelCase(mediaType)
			}

			types = append(types, typeDef)
		}
	}
	return types, nil
}

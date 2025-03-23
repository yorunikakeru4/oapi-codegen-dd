package codegen

import (
	"fmt"

	"github.com/pb33f/libopenapi"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
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

// Generate uses the Go templating engine to generate all of our server wrappers from
// the descriptions we've built up above from the schema objects.
func Generate(doc libopenapi.Document, cfg *Configuration) (string, error) {
	parseCtx, err := createParseContextFromDocument(doc, cfg)
	if err != nil {
		return "", fmt.Errorf("error creating parse context: %w", err)
	}

	parser, err := NewParser(cfg, parseCtx)
	if err != nil {
		return "", fmt.Errorf("error creating parser: %w", err)
	}

	codes, err := parser.Parse()
	if err != nil {
		return "", fmt.Errorf("error parsing: %w", err)
	}

	// original behavior is single file output
	return FormatCode(codes["all"]), nil
}

// CreateParseContext creates a ParseContext from an OpenAPI file and a ParseConfig.
func CreateParseContext(file string, cfg *Configuration) (*ParseContext, []error) {
	if cfg == nil {
		cfg = NewDefaultConfiguration()
	}

	doc, err := loadDocumentFromFile(file)
	if err != nil {
		return nil, []error{err}
	}

	res, err := createParseContextFromDocument(doc, cfg)
	if err != nil {
		return nil, []error{err}
	}

	return res, nil
}

func createParseContextFromDocument(doc libopenapi.Document, cfg *Configuration) (*ParseContext, error) {
	doc, err := filterOutDocument(doc, cfg.Filter)
	if err != nil {
		return nil, fmt.Errorf("error filtering document: %w", err)
	}

	if !cfg.SkipPrune {
		doc, err = pruneSchema(doc)
		if err != nil {
			return nil, fmt.Errorf("error pruning unused components: %w", err)
		}
	}

	builtModel, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		return nil, errs[0]
	}
	model := &builtModel.Model

	var (
		operations    []OperationDefinition
		importSchemas []GoSchema
		typeDefs      []TypeDefinition
	)

	if model == nil || model.Paths == nil {
		return nil, nil
	}

	for path, pathItem := range model.Paths.PathItems.FromOldest() {
		// These are parameters defined for all methods on a given path. They
		// are shared by all methods.
		globalParams, err := describeOperationParameters(pathItem.Parameters, nil)
		if err != nil {
			return nil, fmt.Errorf("error describing global parameters for %s: %s", path, err)
		}

		for method, operation := range pathItem.GetOperations().FromOldest() {
			var (
				headerDef     *TypeDefinition
				queryDef      *TypeDefinition
				pathParamsDef *TypeDefinition
			)

			operationID, err := createOperationID(method, path, operation.OperationId)
			if err != nil {
				return nil, fmt.Errorf("error creating operation ID: %w", err)
			}

			// These are parameters defined for the specific path method that we're iterating over.
			localParams, err := describeOperationParameters(operation.Parameters, []string{operationID})
			if err != nil {
				return nil, fmt.Errorf("error describing local parameters for %s/%s: %s", method, path, err)
			}

			// All the parameters required by a handler are the union of the
			// global parameters and the local parameters.
			allParams, err := combineOperationParameters(globalParams, localParams)
			if err != nil {
				return nil, err
			}
			for _, param := range allParams {
				importSchemas = append(importSchemas, param.Schema)
			}

			// Order the path parameters to match the order as specified in
			// the path, not in the openapi spec, and validate that the parameter
			// names match, as downstream code depends on that.
			pathParameters := filterParameterDefinitionByType(allParams, "path")
			pathDefs, pathSchemas := generateParamsTypes(pathParameters, operationID+"Path")
			if len(pathDefs) > 0 {
				pathParamsDef = &pathDefs[len(pathDefs)-1]
				typeDefs = append(typeDefs, pathDefs...)
			}
			if len(pathSchemas) > 0 {
				importSchemas = append(importSchemas, pathSchemas...)
			}

			queryParams := filterParameterDefinitionByType(allParams, "query")
			queryDefs, querySchemas := generateParamsTypes(queryParams, operationID+"Query")
			if len(queryDefs) > 0 {
				queryDef = &queryDefs[len(queryDefs)-1]
				typeDefs = append(typeDefs, queryDefs...)
			}
			if len(querySchemas) > 0 {
				importSchemas = append(importSchemas, querySchemas...)
			}

			headerParams := filterParameterDefinitionByType(allParams, "header")
			headerDefs, headerSchemas := generateParamsTypes(headerParams, operationID+"Headers")
			if len(headerDefs) > 0 {
				headerDef = &headerDefs[len(headerDefs)-1]
				typeDefs = append(typeDefs, headerDefs...)
			}
			if len(headerSchemas) > 0 {
				importSchemas = append(importSchemas, headerSchemas...)
			}

			// Process Request Body
			bodyDefinition, bodyTypeDef, err := createBodyDefinition(operationID, operation.RequestBody)
			if err != nil {
				return nil, fmt.Errorf("error generating body definitions: %w", err)
			}
			if bodyTypeDef != nil {
				typeDefs = append(typeDefs, *bodyTypeDef)
				importSchemas = append(importSchemas, bodyTypeDef.Schema)
			}
			if bodyDefinition != nil {
				typeDefs = append(typeDefs, bodyDefinition.Schema.AdditionalTypes...)
			}

			// Process Responses
			responseDef, responseTypes, err := getOperationResponses(operationID, operation.Responses)
			if err != nil {
				return nil, fmt.Errorf("error getting operation responses: %w", err)
			}
			typeDefs = append(typeDefs, responseTypes...)
			for _, responseType := range responseTypes {
				importSchemas = append(importSchemas, responseType.Schema)
			}

			operations = append(operations, OperationDefinition{
				ID:          operationID,
				Summary:     operation.Summary,
				Description: operation.Description,
				Method:      method,
				Path:        path,
				PathParams:  pathParamsDef,
				Header:      headerDef,
				Query:       queryDef,
				Response:    *responseDef,
				Body:        bodyDefinition,
			})
		}
	}

	// Process Components
	componentDefs, err := collectComponentDefinitions(model)
	if err != nil {
		return nil, fmt.Errorf("error collecting component definitions: %s", err)
	}
	typeDefs = append(typeDefs, componentDefs...)

	// Collect Schemas from components
	for _, componentDef := range componentDefs {
		importSchemas = append(importSchemas, componentDef.Schema)
	}

	// Collect Imports
	imprts := map[string]goImport{}
	for _, schema := range importSchemas {
		importRes, err := collectSchemaImports(schema)
		if err != nil {
			return nil, fmt.Errorf("error getting schema imports: %w", err)
		}
		mergeImports(imprts, importRes)
	}

	typeDefs, err = checkDuplicates(typeDefs)
	if err != nil {
		return nil, fmt.Errorf("error checking for duplicate type definitions: %w", err)
	}

	enums, typeDefs := filterOutEnums(typeDefs)

	groupedTypeDefs := make(map[SpecLocation][]TypeDefinition)
	var (
		additionalTypes          []TypeDefinition
		unionTypes               []TypeDefinition
		unionWithAdditionalTypes []TypeDefinition
	)

	for _, td := range typeDefs {
		collected := false
		if td.Schema.HasAdditionalProperties {
			additionalTypes = append(additionalTypes, td)
			collected = true
		}

		if len(td.Schema.UnionElements) != 0 {
			td.SpecLocation = SpecLocationUnion
			unionTypes = append(unionTypes, td)
			collected = true
		} else if td.SpecLocation == SpecLocationUnion {
			unionTypes = append(unionTypes, td)
			collected = true
		}

		if len(additionalTypes) != 0 && len(unionTypes) != 0 {
			unionWithAdditionalTypes = append(unionWithAdditionalTypes, td)
			collected = true
		}

		if !collected && td.Name != "" && td.SpecLocation != "" {
			if _, found := groupedTypeDefs[td.SpecLocation]; !found {
				groupedTypeDefs[td.SpecLocation] = []TypeDefinition{}
			}
			groupedTypeDefs[td.SpecLocation] = append(groupedTypeDefs[td.SpecLocation], td)
		}
	}

	return &ParseContext{
		Operations:               operations,
		TypeDefinitions:          groupedTypeDefs,
		Enums:                    enums,
		UnionTypes:               unionTypes,
		AdditionalTypes:          additionalTypes,
		UnionWithAdditionalTypes: unionWithAdditionalTypes,
		Imports:                  importMap(imprts).GoImports(),
	}, nil
}

// collectComponentDefinitions collects all the components from the model and returns them as a list of TypeDefinition.
func collectComponentDefinitions(model *v3high.Document) ([]TypeDefinition, error) {
	if model.Components == nil {
		return nil, nil
	}

	var typeDefs []TypeDefinition

	// Parameters
	if model.Components.Parameters != nil {
		res, err := getComponentParameters(model.Components.Parameters)
		if err != nil {
			return nil, err
		}
		typeDefs = append(typeDefs, res...)
	}

	// Schemas
	if model.Components.Schemas != nil {
		schemas, err := getComponentsSchemas(model.Components.Schemas)
		if err != nil {
			return nil, fmt.Errorf("error getting components schemas: %w", err)
		}
		typeDefs = append(typeDefs, schemas...)
	}

	// RequestBodies
	if model.Components.RequestBodies != nil {
		bodyTypes, err := getComponentsRequestBodies(model.Components.RequestBodies)
		if err != nil {
			return nil, fmt.Errorf("error getting components request bodies: %w", err)
		}
		typeDefs = append(typeDefs, bodyTypes...)
	}

	// Responses
	if model.Components.Responses != nil {
		componentResponses, err := getContentResponses(model.Components.Responses)
		if err != nil {
			return nil, fmt.Errorf("error getting content responses: %w", err)
		}
		typeDefs = append(typeDefs, componentResponses...)
	}

	return typeDefs, nil
}

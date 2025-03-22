// Copyright 2019 DeepMap, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codegen

import (
	"bufio"
	"bytes"
	"fmt"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
	"golang.org/x/tools/imports"
)

// Generate uses the Go templating engine to generate all of our server wrappers from
// the descriptions we've built up above from the schema objects.
func Generate(spec *openapi3.T, cfg *Configuration) (string, error) {
	spec, err := filterDocument(spec, cfg)
	if err != nil {
		return "", fmt.Errorf("error filtering document: %w", err)
	}

	var (
		operations    []OperationDefinition
		importSchemas []Schema
		typeDefs      []TypeDefinition
	)

	if spec == nil || spec.Paths == nil {
		return "", nil
	}

	for _, requestPath := range SortedMapKeys(spec.Paths.Map()) {
		pathItem := spec.Paths.Value(requestPath)
		// These are parameters defined for all methods on a given path. They
		// are shared by all methods.
		globalParams, err := DescribeParameters(pathItem.Parameters, nil)
		if err != nil {
			return "", fmt.Errorf("error describing global parameters for %s: %s",
				requestPath, err)
		}

		// Each path can have a number of operations, POST, GET, OPTIONS, etc.
		pathOps := pathItem.Operations()
		for _, method := range SortedMapKeys(pathOps) {
			var (
				headerDef     *TypeDefinition
				queryDef      *TypeDefinition
				pathParamsDef *TypeDefinition
			)

			op := pathOps[method]
			operationID, err := createOperationID(method, requestPath, op.OperationID)
			if err != nil {
				return "", fmt.Errorf("error creating operation ID: %w", err)
			}

			// These are parameters defined for the specific path method that we're iterating over.
			localParams, err := DescribeParameters(op.Parameters, []string{op.OperationID + "Params"})
			if err != nil {
				return "", fmt.Errorf("error describing global parameters for %s/%s: %s",
					method, requestPath, err)
			}
			// All the parameters required by a handler are the union of the
			// global parameters and the local parameters.
			allParams, err := CombineOperationParameters(globalParams, localParams)
			if err != nil {
				return "", err
			}
			for _, param := range allParams {
				importSchemas = append(importSchemas, param.Schema)
			}

			// Order the path parameters to match the order as specified in
			// the path, not in the swagger spec, and validate that the parameter
			// names match, as downstream code depends on that.
			pathParams := FilterParameterDefinitionByType(allParams, "path")
			pathParams, err = SortParamsByPath(requestPath, pathParams)
			if err != nil {
				return "", err
			}
			pathDefs, pathSchemas := generateParamsTypes(pathParams, operationID+"Path")
			if len(pathDefs) > 0 {
				pathParamsDef = &pathDefs[0]
			}
			typeDefs = append(typeDefs, pathDefs...)
			importSchemas = append(importSchemas, pathSchemas...)

			queryParams := FilterParameterDefinitionByType(allParams, "query")
			queryDefs, querySchemas := generateParamsTypes(queryParams, operationID+"Query")
			if len(queryDefs) > 0 {
				queryDef = &queryDefs[0]
			}
			typeDefs = append(typeDefs, queryDefs...)
			importSchemas = append(importSchemas, querySchemas...)

			headerParams := FilterParameterDefinitionByType(allParams, "header")
			headerDefs, headerSchemas := generateParamsTypes(headerParams, operationID+"Headers")
			if len(headerDefs) > 0 {
				headerDef = &headerDefs[0]
			}
			typeDefs = append(typeDefs, headerDefs...)
			importSchemas = append(importSchemas, headerSchemas...)

			// Process Request Body
			bodyDefinition, bodyTypeDef, err := createBodyDefinition(op.OperationID, op.RequestBody)
			if err != nil {
				return "", fmt.Errorf("error generating body definitions: %w", err)
			}
			if bodyTypeDef != nil {
				typeDefs = append(typeDefs, *bodyTypeDef)
				importSchemas = append(importSchemas, bodyTypeDef.Schema)
			}
			if bodyDefinition != nil {
				typeDefs = append(typeDefs, bodyDefinition.Schema.AdditionalTypes...)
			}

			// Process Responses
			responseDef, responseTypes, err := getOperationResponses(op.OperationID, op.Responses.Map())
			if err != nil {
				return "", fmt.Errorf("error getting operation responses: %w", err)
			}
			typeDefs = append(typeDefs, responseTypes...)
			for _, responseType := range responseTypes {
				importSchemas = append(importSchemas, responseType.Schema)
			}

			opDef := OperationDefinition{
				ID:          operationID,
				Summary:     op.Summary,
				Description: op.Description,
				Method:      method,
				Path:        requestPath,
				PathParams:  pathParamsDef,
				Header:      headerDef,
				Query:       queryDef,
				Response:    *responseDef,
				Body:        bodyDefinition,
			}

			if op.RequestBody != nil {
				opDef.BodyRequired = op.RequestBody.Value.Required
			}

			operations = append(operations, opDef)
		}
	}

	// Process Components
	componentDefs, err := collectComponentDefinitions(spec)
	if err != nil {
		return "", fmt.Errorf("error collecting component definitions: %s", err)
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
			return "", fmt.Errorf("error getting schema imports: %w", err)
		}
		MergeImports(imprts, importRes)
	}

	typeDefs, err = checkDuplicates(typeDefs)
	if err != nil {
		return "", fmt.Errorf("error checking for duplicate type definitions: %w", err)
	}

	enums, typeDefs := filterOutEnums(typeDefs)

	groupedTypeDefs := make(map[SpecLocation][]TypeDefinition)
	var (
		additionalTypes          []TypeDefinition
		unionTypes               []TypeDefinition
		unionWithAdditionalTypes []TypeDefinition
	)

	for _, td := range typeDefs {
		if _, found := groupedTypeDefs[td.SpecLocation]; !found {
			groupedTypeDefs[td.SpecLocation] = []TypeDefinition{}
		}

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

		if !collected {
			groupedTypeDefs[td.SpecLocation] = append(groupedTypeDefs[td.SpecLocation], td)
		}
	}

	parseCtx := &ParseContext{
		Operations:               operations,
		TypeDefinitions:          groupedTypeDefs,
		Enums:                    enums,
		UnionTypes:               unionTypes,
		AdditionalTypes:          additionalTypes,
		UnionWithAdditionalTypes: unionWithAdditionalTypes,
		Imports:                  importMap(imprts).GoImports(),
	}

	var typeDefinitions, constantDefinitions string

	parser, err := NewParser(cfg, parseCtx)
	if err != nil {
		return "", fmt.Errorf("error creating parser: %w", err)
	}

	// temporary pass parser.tpl
	clientOut, err := GenerateClient(parser.tpl, operations)
	if err != nil {
		return "", fmt.Errorf("error generating client: %w", err)
	}

	var clientWithResponsesOut string
	clientWithResponsesOut, err = GenerateClientWithResponses(parser.tpl, operations)
	if err != nil {
		return "", fmt.Errorf("error generating client with responses: %w", err)
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	externalImports := importMap(imprts).GoImports()
	importsOut, err := GenerateImports(
		parser.tpl,
		externalImports,
		cfg.PackageName,
	)
	if err != nil {
		return "", fmt.Errorf("error generating imports: %w", err)
	}

	_, err = w.WriteString(importsOut)
	if err != nil {
		return "", fmt.Errorf("error writing imports: %w", err)
	}

	_, err = w.WriteString(constantDefinitions)
	if err != nil {
		return "", fmt.Errorf("error writing constants: %w", err)
	}

	_, err = w.WriteString(typeDefinitions)
	if err != nil {
		return "", fmt.Errorf("error writing type definitions: %w", err)
	}

	_, err = w.WriteString(clientOut)
	if err != nil {
		return "", fmt.Errorf("error writing client: %w", err)
	}
	_, err = w.WriteString(clientWithResponsesOut)
	if err != nil {
		return "", fmt.Errorf("error writing client: %w", err)
	}

	err = w.Flush()
	if err != nil {
		return "", fmt.Errorf("error flushing output buffer: %w", err)
	}

	// remove any byte-order-marks which break Go-Code
	goCode := sanitizeCode(buf.String())

	outBytes, err := imports.Process(cfg.PackageName+".go", []byte(goCode), nil)
	if err != nil {
		return "", fmt.Errorf("error formatting Go code %s: %w", goCode, err)
	}
	return string(outBytes), nil
}

// collectComponentDefinitions collects all the components from the model and returns them as a list of TypeDefinition.
func collectComponentDefinitions(model *openapi3.T) ([]TypeDefinition, error) {
	if model.Components == nil {
		return nil, nil
	}

	var typeDefs []TypeDefinition
	schemaTypes, err := GenerateTypesForSchemas(model.Components.Schemas)
	if err != nil {
		return nil, fmt.Errorf("error generating Go types for component schemas: %w", err)
	}

	paramTypes, err := GenerateTypesForParameters(model.Components.Parameters)
	if err != nil {
		return nil, fmt.Errorf("error generating Go types for component parameters: %w", err)
	}
	typeDefs = append(schemaTypes, paramTypes...)

	responseTypes, err := GenerateTypesForResponses(model.Components.Responses)
	if err != nil {
		return nil, fmt.Errorf("error generating Go types for component responses: %w", err)
	}
	typeDefs = append(typeDefs, responseTypes...)

	bodyTypes, err := GenerateTypesForRequestBodies(model.Components.RequestBodies)
	if err != nil {
		return nil, fmt.Errorf("error generating Go types for component request bodies: %w", err)
	}
	typeDefs = append(typeDefs, bodyTypes...)

	return typeDefs, nil
}

// GenerateImports generates our import statements and package definition.
func GenerateImports(t *template.Template, externalImports []string, packageName string) (string, error) {
	context := struct {
		ExternalImports []string
		PackageName     string
	}{
		ExternalImports: externalImports,
		PackageName:     packageName,
	}

	return GenerateTemplates([]string{"imports.tmpl"}, t, context)
}

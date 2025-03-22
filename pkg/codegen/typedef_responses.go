package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/doordash/oapi-codegen/v2/pkg/util"
	"github.com/getkin/kin-openapi/openapi3"
)

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

type ResponseDefinition struct {
	StatusCode  string
	Description string
	Contents    []ResponseContentDefinition
	Ref         string

	SuccessStatusCode int
	Success           *ResponseContentDefinition
	Error             *ResponseContentDefinition

	// TODO: remove after migration
	TypeDefinitions []ResponseTypeDefinition
}

func (r ResponseDefinition) GoName() string {
	return SchemaNameToTypeName(r.StatusCode)
}

func (r ResponseDefinition) IsRef() bool {
	return r.Ref != ""
}

// ResponseContentDefinition describes Operation response.
// GoSchema is the schema describing this content.
// ContentType is the content type corresponding to the body, eg, application/json.
// NameTag is the tag for the type name, such as JSON, in which case we will produce "Response200JSONContent".
// ResponseName is the name of the response.
// Description is the description of the response.
// Ref is the reference to the response.
// IsSuccess is true if the response is a success response.
type ResponseContentDefinition struct {
	Schema      GoSchema
	ContentType string
	NameTag     string

	ResponseName string
	Description  string
	Ref          string
	IsSuccess    bool
}

// TypeDef returns the Go type definition for a request body
func (r ResponseContentDefinition) TypeDef(opID string, statusCode int) *TypeDefinition {
	return &TypeDefinition{
		Name:   fmt.Sprintf("%s%v%sResponse", opID, statusCode, r.NameTagOrContentType()),
		Schema: r.Schema,
	}
}

func (r ResponseContentDefinition) IsSupported() bool {
	return r.NameTag != ""
}

// HasFixedContentType returns true if content type has fixed content type, i.e. contains no "*" symbol
func (r ResponseContentDefinition) HasFixedContentType() bool {
	return !strings.Contains(r.ContentType, "*")
}

func (r ResponseContentDefinition) NameTagOrContentType() string {
	if r.NameTag != "" {
		return r.NameTag
	}
	return SchemaNameToTypeName(r.ContentType)
}

// IsJSON returns whether this is a JSON media type, for instance:
// - application/json
// - application/vnd.api+json
// - application/*+json
func (r ResponseContentDefinition) IsJSON() bool {
	return util.IsMediaTypeJson(r.ContentType)
}

type ResponseHeaderDefinition struct {
	Name   string
	GoName string
	Schema GoSchema
}

func getOperationResponses(operationID string, responses map[string]*openapi3.ResponseRef) (*ResponseDefinition, []TypeDefinition, error) {
	var (
		successDefinition *ResponseContentDefinition
		errorDefinition   *ResponseContentDefinition
		successCode       int
	)

	var typeDefinitions []TypeDefinition
	var resTypeDefs []ResponseTypeDefinition

	for _, statusCode := range SortedMapKeys(responses) {
		responseOrRef := responses[statusCode]
		if responseOrRef == nil {
			continue
		}
		response := responseOrRef.Value

		isSuccess := false
		refType := ""
		var err error
		if responseOrRef.Ref != "" {
			refType, err = RefPathToGoType(responseOrRef.Ref)
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing ref %s: %w", responseOrRef.Ref, err)
			}
		}

		status, err := strconv.Atoi(statusCode)
		if err != nil {
			if statusCode == "default" || strings.ToLower(statusCode) == "2xx" {
				status = 200
			} else if strings.ToLower(statusCode) == "4xx" || strings.ToLower(statusCode) == "5xx" {
				status = 400
			} else {
				return nil, nil, fmt.Errorf("error parsing status code %s: %w", statusCode, err)
			}
		}

		if status >= 200 && status < 300 {
			isSuccess = true
			successCode = status
		} else if status >= 400 && status < 600 {
			isSuccess = false
		} else {
			continue
		}

		// pick one schema
		// TODO: prefer json content type
		if isSuccess && successDefinition != nil {
			continue
		}
		if !isSuccess && errorDefinition != nil {
			continue
		}

		var responseContentDefinitions []ResponseContentDefinition

		typeSuffix := "Response"
		if !isSuccess {
			typeSuffix = "ErrorResponse"
		}

		for _, contentType := range SortedMapKeys(response.Content) {
			content := response.Content[contentType]
			var tag string
			switch {
			case contentType == "application/json":
				tag = "JSON"
			case util.IsMediaTypeJson(contentType):
				tag = mediaTypeToCamelCase(contentType)
			case contentType == "application/x-www-form-urlencoded":
				tag = "Formdata"
			case strings.HasPrefix(contentType, "multipart/"):
				tag = "Multipart"
			case contentType == "text/plain":
				tag = "Text"
			default:
				rcd := ResponseContentDefinition{
					ContentType: contentType,
				}
				responseContentDefinitions = append(responseContentDefinitions, rcd)
				continue
			}

			responseTypeName := operationID + typeSuffix
			contentSchema, err := GenerateGoSchema(content.Schema, []string{responseTypeName})
			if err != nil {
				return nil, nil, fmt.Errorf("error generating request body definition: %w", err)
			}

			contentSchema.RefType = refType

			td := TypeDefinition{
				Name:         responseTypeName,
				Schema:       contentSchema,
				SpecLocation: SpecLocationResponse,
			}

			typeDefinitions = append(typeDefinitions, td)
			typeDefinitions = append(typeDefinitions, contentSchema.AdditionalTypes...)

			// TODO: remove after migration
			resTypeDefs = append(resTypeDefs, ResponseTypeDefinition{
				TypeDefinition:            td,
				ContentTypeName:           contentType,
				ResponseName:              responseTypeName,
				AdditionalTypeDefinitions: contentSchema.AdditionalTypes,
			})

			description := ""
			if response.Description != nil {
				description = *response.Description
			}

			rcd := &ResponseContentDefinition{
				ResponseName: responseTypeName,
				IsSuccess:    isSuccess,
				Description:  description,
				Schema:       contentSchema,
				Ref:          refType,
				ContentType:  contentType,
				NameTag:      tag,
			}

			if isSuccess {
				successDefinition = rcd
			} else {
				errorDefinition = rcd
			}
		}
	}

	res := &ResponseDefinition{
		SuccessStatusCode: successCode,
		Success:           successDefinition,
		Error:             errorDefinition,
		TypeDefinitions:   resTypeDefs,
	}

	return res, typeDefinitions, nil
}

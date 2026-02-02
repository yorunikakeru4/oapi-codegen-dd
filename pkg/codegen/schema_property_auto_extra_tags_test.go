// Copyright 2025 DoorDash, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPropertyFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		property  Property
		fieldName string
		want      string
	}{
		{
			name: "extract description",
			property: Property{
				Description: "This is a test description",
			},
			fieldName: "description",
			want:      "This is a test description",
		},
		{
			name: "extract x-validation extension",
			property: Property{
				Extensions: map[string]any{
					"x-validation": "required,email",
				},
			},
			fieldName: "x-validation",
			want:      "required,email",
		},
		{
			name: "extract x-custom extension",
			property: Property{
				Extensions: map[string]any{
					"x-custom": "custom-value",
				},
			},
			fieldName: "x-custom",
			want:      "custom-value",
		},
		{
			name: "non-existent extension returns empty",
			property: Property{
				Extensions: map[string]any{},
			},
			fieldName: "x-missing",
			want:      "",
		},
		{
			name: "non-string extension returns empty",
			property: Property{
				Extensions: map[string]any{
					"x-numeric": 123,
				},
			},
			fieldName: "x-numeric",
			want:      "",
		},
		{
			name:      "empty description returns empty",
			property:  Property{},
			fieldName: "description",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPropertyFieldValue(tt.property, tt.fieldName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenFieldsFromProperties_AutoExtraTags(t *testing.T) {
	tests := []struct {
		name       string
		properties []Property
		options    ParseOptions
		wantTags   map[string][]string // field name -> expected tags
	}{
		{
			name: "auto-extra-tags from description",
			properties: []Property{
				{
					GoName:        "Email",
					JsonFieldName: "email",
					Description:   "User email address",
					Schema: GoSchema{
						GoType: "string",
					},
					Constraints: Constraints{},
				},
			},
			options: ParseOptions{
				AutoExtraTags: map[string]string{
					"jsonschema": "description",
				},
			},
			wantTags: map[string][]string{
				"Email": {"json:", "jsonschema:"},
			},
		},
		{
			name: "auto-extra-tags from x-validation",
			properties: []Property{
				{
					GoName:        "Username",
					JsonFieldName: "username",
					Description:   "Username",
					Schema: GoSchema{
						GoType: "string",
					},
					Extensions: map[string]any{
						"x-validation": "required,min=3",
					},
					Constraints: Constraints{},
				},
			},
			options: ParseOptions{
				AutoExtraTags: map[string]string{
					"customvalidate": "x-validation",
				},
			},
			wantTags: map[string][]string{
				"Username": {"json:", "customvalidate:"},
			},
		},
		{
			name: "multiple auto-extra-tags",
			properties: []Property{
				{
					GoName:        "Email",
					JsonFieldName: "email",
					Description:   "User email",
					Schema: GoSchema{
						GoType: "string",
					},
					Extensions: map[string]any{
						"x-validation": "required,email",
					},
					Constraints: Constraints{},
				},
			},
			options: ParseOptions{
				AutoExtraTags: map[string]string{
					"jsonschema":     "description",
					"customvalidate": "x-validation",
				},
			},
			wantTags: map[string][]string{
				"Email": {"json:", "jsonschema:", "customvalidate:"},
			},
		},
		{
			name: "auto-extra-tags doesn't override existing tags",
			properties: []Property{
				{
					GoName:        "ID",
					JsonFieldName: "id",
					Description:   "ID field",
					Schema: GoSchema{
						GoType: "int64",
					},
					Extensions: map[string]any{
						"x-oapi-codegen-extra-tags": map[string]any{
							"jsonschema": "existing-value",
						},
					},
					Constraints: Constraints{},
				},
			},
			options: ParseOptions{
				AutoExtraTags: map[string]string{
					"jsonschema": "description",
				},
			},
			wantTags: map[string][]string{
				"ID": {"json:", `jsonschema:"existing-value"`}, // Should keep existing value
			},
		},
		{
			name: "auto-extra-tags skips empty values",
			properties: []Property{
				{
					GoName:        "Field",
					JsonFieldName: "field",
					Description:   "", // Empty description
					Schema: GoSchema{
						GoType: "string",
					},
					Constraints: Constraints{},
				},
			},
			options: ParseOptions{
				AutoExtraTags: map[string]string{
					"jsonschema": "description",
				},
			},
			wantTags: map[string][]string{
				"Field": {"json:"}, // No jsonschema tag
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := genFieldsFromProperties(tt.properties, tt.options)
			require.Len(t, fields, len(tt.properties))

			for i, field := range fields {
				propName := tt.properties[i].GoName
				expectedTags := tt.wantTags[propName]

				for _, expectedTag := range expectedTags {
					assert.Contains(t, field, expectedTag, "field should contain tag: %s", expectedTag)
				}
			}
		})
	}
}

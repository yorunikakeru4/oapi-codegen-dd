package union

import (
	"encoding/json"

	"testing"

	"github.com/yorunikakeru4/oapi-codegen-dd/v3/pkg/runtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTarget_UnmarshalJSON_EmailTarget(t *testing.T) {
	input := `{"id": 1, "type": "email", "email": "test@example.com", "custom": "value"}`

	var target Target
	err := json.Unmarshal([]byte(input), &target)
	require.NoError(t, err)

	// Base fields
	assert.Equal(t, 1, target.TargetBase.ID)
	assert.Equal(t, "email", target.TargetBase.Type)

	// Union variant
	require.NotNil(t, target.Target_AllOf1)
	require.NotNil(t, target.Target_AllOf1.Target_AllOf1_AnyOf)
	assert.True(t, target.Target_AllOf1.Target_AllOf1_AnyOf.IsA())
	assert.Equal(t, "test@example.com", target.Target_AllOf1.Target_AllOf1_AnyOf.A.Email)

	// AdditionalProperties - only custom field
	assert.Equal(t, map[string]string{"custom": "value"}, target.AdditionalProperties)
}

func TestTarget_UnmarshalJSON_WebhookTarget(t *testing.T) {
	input := `{"id": 2, "type": "webhook", "url": "https://example.com", "extra": "data"}`

	var target Target
	err := json.Unmarshal([]byte(input), &target)
	require.NoError(t, err)

	assert.Equal(t, 2, target.TargetBase.ID)
	assert.True(t, target.Target_AllOf1.Target_AllOf1_AnyOf.IsB())
	assert.Equal(t, "https://example.com", target.Target_AllOf1.Target_AllOf1_AnyOf.B.URL)
	assert.Equal(t, map[string]string{"extra": "data"}, target.AdditionalProperties)
}

func TestTarget_UnmarshalJSON_NoAdditionalProperties(t *testing.T) {
	input := `{"id": 1, "type": "email", "email": "test@example.com"}`

	var target Target
	err := json.Unmarshal([]byte(input), &target)
	require.NoError(t, err)

	assert.Nil(t, target.AdditionalProperties)
}

func TestTarget_MarshalJSON(t *testing.T) {
	target := Target{
		TargetBase: TargetBase{ID: 1, Type: "email"},
		Target_AllOf1: &Target_AllOf1{
			Target_AllOf1_AnyOf: &Target_AllOf1_AnyOf{},
		},
		AdditionalProperties: map[string]string{"custom": "value"},
	}
	target.Target_AllOf1.Target_AllOf1_AnyOf.Either = runtime.NewEitherFromA[EmailTarget, WebhookTarget](EmailTarget{Email: "test@example.com"})

	data, err := json.Marshal(target)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))

	assert.Equal(t, float64(1), result["id"])
	assert.Equal(t, "email", result["type"])
	assert.Equal(t, "test@example.com", result["email"])
	assert.Equal(t, "value", result["custom"])
}

func TestTarget_Roundtrip(t *testing.T) {
	original := `{"id":1,"type":"email","email":"test@example.com","custom":"value"}`

	var target Target
	require.NoError(t, json.Unmarshal([]byte(original), &target))

	data, err := json.Marshal(target)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))

	assert.Equal(t, float64(1), result["id"])
	assert.Equal(t, "email", result["type"])
	assert.Equal(t, "test@example.com", result["email"])
	assert.Equal(t, "value", result["custom"])
}

func TestTargetWithExtra_UnmarshalJSON_EmailTarget(t *testing.T) {
	input := `{"email": "test@example.com", "extra": "data"}`

	var target TargetWithExtra
	err := json.Unmarshal([]byte(input), &target)
	require.NoError(t, err)

	require.NotNil(t, target.TargetWithExtra_AnyOf)
	assert.True(t, target.TargetWithExtra_AnyOf.IsA())
	assert.Equal(t, "test@example.com", target.TargetWithExtra_AnyOf.A.Email)
	assert.Equal(t, map[string]string{"extra": "data"}, target.AdditionalProperties)
}

func TestTargetWithExtra_UnmarshalJSON_WebhookTarget(t *testing.T) {
	input := `{"url": "https://example.com", "custom": "value"}`

	var target TargetWithExtra
	err := json.Unmarshal([]byte(input), &target)
	require.NoError(t, err)

	require.NotNil(t, target.TargetWithExtra_AnyOf)
	assert.True(t, target.TargetWithExtra_AnyOf.IsB())
	assert.Equal(t, "https://example.com", target.TargetWithExtra_AnyOf.B.URL)
	assert.Equal(t, map[string]string{"custom": "value"}, target.AdditionalProperties)
}

func TestTargetWithExtra_UnmarshalJSON_NoAdditionalProperties(t *testing.T) {
	input := `{"email": "test@example.com"}`

	var target TargetWithExtra
	err := json.Unmarshal([]byte(input), &target)
	require.NoError(t, err)

	assert.Nil(t, target.AdditionalProperties)
}

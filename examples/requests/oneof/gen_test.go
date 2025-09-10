package gen

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessPaymentBody_MarshalJSON(t *testing.T) {
	b := PayloadB{
		B: func() *string {
			s := "b-value"
			return &s
		}(),
	}
	oneOf := &ProcessPaymentBody_OneOf{}
	_ = oneOf.FromPayloadB(b)

	expected := `{"b": "b-value"}`

	payload := &ProcessPaymentBody{
		ProcessPaymentBody_OneOf: oneOf,
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	require.JSONEq(t, expected, string(data))
}

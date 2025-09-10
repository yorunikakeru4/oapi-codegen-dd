package gen

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessPaymentBody_MarshalJSON(t *testing.T) {
	b := PayloadC{
		C: func() *string {
			s := "c-value"
			return &s
		}(),
	}
	oneOf := &Payload_OneOf{}
	_ = oneOf.FromPayloadC(b)

	expected := `{"c": "c-value"}`

	payload := &Payload{
		Payload_OneOf: oneOf,
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	require.JSONEq(t, expected, string(data))
}

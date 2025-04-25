package anyofallofoneof

import (
	"encoding/json"
	"testing"

	"github.com/doordash/oapi-codegen/v3/pkg/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientOrID_UnmarshalJSON(t *testing.T) {
	t.Run("unmarshal as client", func(t *testing.T) {
		type responseWithClient struct {
			Client *ClientOrID `json:"client"`
		}

		res := &responseWithClient{}
		err := json.Unmarshal([]byte(`{"client": {"name": "foo"}}`), res)
		require.NoError(t, err)
		assert.Equal(t, Client{Name: "foo"}, res.Client.A)
	})

	t.Run("unmarshal as string", func(t *testing.T) {
		type responseWithClient struct {
			Client ClientOrID `json:"client"`
		}

		res := &responseWithClient{}
		err := json.Unmarshal([]byte(`{"client": "abc-123"}`), res)
		require.NoError(t, err)
		assert.Equal(t, "abc-123", res.Client.B)
	})
}

func TestClientOrID_MarshalJSON(t *testing.T) {
	t.Run("marshal as client", func(t *testing.T) {
		type responseWithClient struct {
			Client ClientOrID `json:"client"`
		}

		res := &responseWithClient{Client: ClientOrID{
			Either: runtime.NewEitherFromA[Client, string](Client{Name: "foo"}),
		}}
		b, err := json.Marshal(res)
		require.NoError(t, err)

		assert.Equal(t, `{"client":{"name":"foo"}}`, string(b))
	})

	t.Run("marshal as string", func(t *testing.T) {
		type responseWithClient struct {
			Client ClientOrID `json:"client"`
		}

		res := &responseWithClient{Client: ClientOrID{
			Either: runtime.NewEitherFromB[Client, string]("abc-123"),
		}}

		b, err := json.Marshal(res)
		require.NoError(t, err)

		assert.Equal(t, `{"client":"abc-123"}`, string(b))
	})
}

func TestClientOrIdentityWithDiscriminator_UnmarshalJSON(t *testing.T) {
	t.Run("unmarshal as client", func(t *testing.T) {
		type responseWithClientOrIdentity struct {
			Data ClientOrIdentityWithDiscriminator `json:"data"`
		}

		res := &responseWithClientOrIdentity{}
		err := json.Unmarshal([]byte(`{"data": {"type": "client", "name": "foo"}}`), res)
		require.NoError(t, err)
		assert.Equal(t, Client{Name: "foo"}, res.Data.A)
	})

	t.Run("unmarshal as identity", func(t *testing.T) {
		type responseWithClientOrIdentity struct {
			Data ClientOrIdentityWithDiscriminator `json:"data"`
		}

		res := &responseWithClientOrIdentity{}
		err := json.Unmarshal([]byte(`{"data": {"type": "identity", "issuer": "abc-123"}}`), res)
		require.NoError(t, err)
		assert.Equal(t, Identity{Issuer: "abc-123"}, res.Data.B)
	})
}

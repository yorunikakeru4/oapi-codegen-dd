package gen

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusinessGroupResponse_UnmarshalJSON(t *testing.T) {
	data := []byte(`[{"name":"Group A","id":1},{"name":"Group B","id":2}]`)

	var resp BusinessGroupResponse
	err := json.Unmarshal(data, &resp)
	require.NoError(t, err)

	require.Len(t, resp, 2)
	assert.Equal(t, "Group A", *resp[0].Name)
	assert.Equal(t, 1, *resp[0].ID)
	assert.Equal(t, "Group B", *resp[1].Name)
	assert.Equal(t, 2, *resp[1].ID)
}

func TestBusinessGroupResponse_MarshalJSON(t *testing.T) {
	name1, name2 := "Group A", "Group B"
	id1, id2 := 1, 2

	resp := BusinessGroupResponse{
		{Name: &name1, ID: &id1},
		{Name: &name2, ID: &id2},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	expected := `[{"name":"Group A","id":1},{"name":"Group B","id":2}]`
	assert.JSONEq(t, expected, string(data))
}

func TestGetBusinessGroupsResponse_IsAlias(t *testing.T) {
	// GetBusinessGroupsResponse should be an alias to BusinessGroupResponse
	var resp GetBusinessGroupsResponse
	var bgResp BusinessGroupResponse

	// They should be the same type
	resp = bgResp
	bgResp = resp

	assert.Equal(t, resp, bgResp)
}

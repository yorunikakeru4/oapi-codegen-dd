package runtime

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeFormFields(t *testing.T) {
	type Coordinates struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	type Address struct {
		City        string `json:"city"`
		Country     string `json:"country"`
		Coordinates `json:"coordinates"`
	}

	type User struct {
		Name      string   `json:"name"`
		Age       int      `json:"age"`
		Address   Address  `json:"address"`
		Nicknames []string `json:"nicknames"`
	}

	payload := User{
		Name: "Jane Doe",
		Age:  30,
		Address: Address{
			City:    "Berlin",
			Country: "DE",
			Coordinates: Coordinates{
				Latitude:  52.5200,
				Longitude: 13.4050,
			},
		},
		Nicknames: []string{"JD", "Janie"},
	}

	t.Run("default", func(t *testing.T) {
		res, err := EncodeFormFields(payload, map[string]FieldEncoding{
			"address": {},
			"nicknames": {
				Style: "form",
			},
		})

		resDecoded, _ := url.QueryUnescape(res)
		require.NoError(t, err)
		expected := "address.city=Berlin&address.coordinates.latitude=52.52&address.coordinates.longitude=13.405&address.country=DE&age=30&name=Jane Doe&nicknames=JD&nicknames=Janie"
		assert.Equal(t, expected, resDecoded)
	})

	t.Run("deepObject", func(t *testing.T) {
		res, err := EncodeFormFields(payload, map[string]FieldEncoding{
			"address": {
				Style: "deepObject",
			},
			"nicknames": {
				Style: "deepObject",
			},
		})

		resDecoded, _ := url.QueryUnescape(res)
		require.NoError(t, err)
		expected := "address[city]=Berlin&address[coordinates][latitude]=52.52&address[coordinates][longitude]=13.405&address[country]=DE&age=30&name=Jane Doe&nicknames[0]=JD&nicknames[1]=Janie"
		assert.Equal(t, expected, resDecoded)
	})
}

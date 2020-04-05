package smartapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArguments(t *testing.T) {
	endpointParams := []EndpointParam{
		Header(""),
		JSONBody(""),
		StringBody(),
		ByteSliceBody(),
		BodyReader(),
		URLParam(""),
		Context(),
		ResponseStatus(404),
		ResponseHeaders(),
		ResponseCookies(),
		QueryParam(""),
		PostQueryParam(""),
		Cookie(""),
	}

	for _, p := range endpointParams {
		if p.options().has(flagArgument) {
			_, ok := p.(Argument)
			require.True(t, ok)
		}
		if p.options().has(flagResponseStatus) {
			_, ok := p.(responseStatusArgument)
			require.True(t, ok)
		}
	}
}

package smartapi

import (
	"net/http"
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
		Middleware(func(handler http.Handler) http.Handler { return nil }),
	}

	for _, p := range endpointParams {
		if p.options().has(flagArgument) {
			_, ok := p.(Argument)
			require.True(t, ok)
		}
		if p.options().has(flagMiddleware) {
			_, ok := p.(middleware)
			require.True(t, ok)
		}
		if p.options().has(flagResponseStatus) {
			_, ok := p.(responseStatusArgument)
			require.True(t, ok)
		}
	}
}

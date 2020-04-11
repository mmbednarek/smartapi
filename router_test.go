package smartapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_router_MustHandler(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		r := NewRouter()
		require.NotNil(t, r.MustHandler())
	})
	t.Run("Panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("not panicked")
			}
		}()
		r := NewRouter()
		r.Post("/test", nil)
		r.MustHandler()
	})
}
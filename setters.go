package smartapi

import (
	"net/http"
)

// Cookies interface allows to add response cookies
type Cookies interface {
	Add(c *http.Cookie)
}

type cookieSetter struct {
	w http.ResponseWriter
}

func (h cookieSetter) Add(c *http.Cookie) {
	cookies := h.w.Header().Get("Set-Cookie")
	if len(cookies) == 0 {
		h.w.Header().Set("Set-Cookie", c.String())
		return
	}
	h.w.Header().Set("Set-Cookie", cookies+"; "+c.String())
}

// Headers interface allows to add response header values
type Headers interface {
	Add(key, value string)
	Set(key, value string)
	Get(key string) string
}

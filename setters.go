package smartapi

import (
	"net/http"
)

type Cookies interface {
	Set(c *http.Cookie)
}

type cookieSetter struct {
	w http.ResponseWriter
}

func (h cookieSetter) Set(c *http.Cookie){
	http.SetCookie(h.w, c)
}

type Headers interface {
	Add(key, value string)
	Set(key, value string)
	Get(key string) string
}

package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func NewHandler(backendURL string) http.HandlerFunc {
	target, _ := url.Parse(backendURL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.FlushInterval = 100 * time.Millisecond
		proxy.ServeHTTP(w, r)
	})
}

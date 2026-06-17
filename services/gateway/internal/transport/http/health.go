package http

import "net/http"

// Livez implements [transport.Handler].
func (h handlers) Livez() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "OK"})
	}
}

// Readyz implements [transport.Handler].
func (h handlers) Readyz() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if h.ready() {
			writeJSON(w, http.StatusOK, map[string]string{"status": "Ready"})
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "NotReady"})
	}
}

// Startupz implements [transport.Handler].
func (h handlers) Startupz() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if h.ready() {
			writeJSON(w, http.StatusOK, map[string]string{"status": "OK"})
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "Starting"})
	}
}

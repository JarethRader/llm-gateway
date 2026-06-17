package transport

import "net/http"

type Handler interface {
	// Chat
	HandleChatCompletion() http.HandlerFunc

	// Health Check
	Livez() http.HandlerFunc
	Startupz() http.HandlerFunc
	Readyz() http.HandlerFunc
}

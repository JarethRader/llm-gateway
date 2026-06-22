package backend

import "net/http"

type Handler interface {
	CreateBackend() http.HandlerFunc // POST | /api/v1/backend
	UpdateBackend() http.HandlerFunc // PUT | /api/v1/backend/{backendID}
	DeleteBackend() http.HandlerFunc // DELETE | /api/v1/backend/{backendID}

	ListBackends() http.HandlerFunc // GET | /api/v1/backend
	GetBackend() http.HandlerFunc   // GET | /api/v1/backend/{backendID}
}

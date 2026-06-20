package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type MiddlewareHandlers interface {
	Recovery(next http.Handler) http.Handler
}

type middleware struct {
	lgr *slog.Logger
}

func NewMiddleware(lgr *slog.Logger) MiddlewareHandlers {
	return &middleware{
		lgr: lgr,
	}
}

func (m *middleware) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				m.lgr.Error("internal server error", slog.Any("error", err))

				json, _ := json.Marshal(map[string]string{
					"error": "There was an internal error",
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write(json)
				if err != nil {
					m.lgr.Error("internal server error", slog.Any("error", err))
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

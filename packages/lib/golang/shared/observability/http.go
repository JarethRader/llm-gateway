package observability

import (
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const HeaderRequestID = "X-Request-ID"

// NewHTTPClient returns an http.Client whose Transport injects W3C
// traceparent into every outbound request and records CLIENT span.
func NewHTTPClient(rt http.RoundTripper) *http.Client {
	transport := http.DefaultTransport
	if rt != nil {
		transport = rt
	}

	return &http.Client{
		Transport: otelhttp.NewTransport(transport),
	}
}

// HTTPMiddleware ensures every inbound HTTP request carries a requestId,
// a layer marker, and an authenticated userId (if present) on its context.
// It must be wired AFTER otelhttp.NewHandler so the OTel span context is
// already populated when the logger writes its first record.
func HTTPMiddleware(userIDExtractor func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			rid := r.Header.Get(HeaderRequestID)
			if rid == "" {
				rid = uuid.NewString()
			}
			w.Header().Set(HeaderRequestID, rid)

			ctx = WithRequestID(ctx, rid)
			ctx = WithLayer(ctx, LayerDelivery)
			if userIDExtractor != nil {
				if uid := userIDExtractor(r); uid != "" {
					ctx = WithUserID(ctx, uid)
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

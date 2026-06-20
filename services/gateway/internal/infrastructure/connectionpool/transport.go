package connectionpool

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"packages/lib/golang/shared/config"
	"packages/lib/golang/shared/observability"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
	"golang.org/x/net/http2"
)

func newClient(b model.Backend, cfg config.ConnectionPool) model.BackendConnection {
	base := &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			KeepAlive: cfg.TCPKeepAlive,
		}).DialContext,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		WriteBufferSize:       cfg.WriteBufferSize,
		ReadBufferSize:        cfg.ReadBufferSize,
		ForceAttemptHTTP2:     b.Protocol == model.ProtocolH2,
	}

	var roundTrip http.RoundTripper = base
	switch b.Protocol {
	case model.ProtocolH2:
		base.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		base.HTTP2 = &http.HTTP2Config{
			PingTimeout: cfg.H2PingTimeout,
		}
		_ = http2.ConfigureTransport(base)
	case model.ProtocolH2C:
		roundTrip = &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return (&net.Dialer{
					KeepAlive: cfg.TCPKeepAlive,
				}).DialContext(ctx, network, addr)
			},
			ReadIdleTimeout: cfg.H2ReadIdleTimeout,
			PingTimeout:     cfg.H2PingTimeout,
		}
	case model.ProtocolH1:
		// default: HTTP/1.1 with a large keep-alive pool
		base.ForceAttemptHTTP2 = false
	}

	return model.BackendConnection{
		Model:      b,
		Connection: observability.NewHTTPClient(roundTrip),
	}
}

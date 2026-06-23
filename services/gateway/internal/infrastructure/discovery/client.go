package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

type Client struct {
	httpClient *http.Client
	apiURL     string
	interval   time.Duration
	sink       func([]model.Backend)
	lgr        *slog.Logger
}

func NewClient(lgr *slog.Logger, url string, interval time.Duration, sink func([]model.Backend)) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		apiURL:     url,
		interval:   interval,
		sink:       sink,
		lgr:        lgr,
	}
}

func (c *Client) FetchBackends(ctx context.Context) ([]model.Backend, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/backend", c.apiURL), nil)
	if err != nil {
		return nil, fmt.Errorf("discovery: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discovery: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery: unexpected status %d", resp.StatusCode)
	}

	var envelope struct {
		Backends []BackendDTO `json:"backends"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("discovery: decode response: %w", err)
	}

	var result []model.Backend
	for _, b := range envelope.Backends {
		if !b.Enabled {
			continue
		}
		result = append(result, b.ToDomain())
	}
	return result, nil
}

func (c *Client) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.lgr.Info("discovery: starting poller", slog.String("interval", c.interval.String()))

	for {
		select {
		case <-ctx.Done():
			c.lgr.Info("discovery: stopping poller")
			return
		default:
		}

		backends, err := c.FetchBackends(ctx)
		if err != nil {
			c.lgr.Error("discovery: poll failed", slog.Any("error", err))
		} else {
			c.sink(backends)
			c.lgr.Debug("discovery: synced backends", slog.Int("count", len(backends)))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type BackendDTO struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Protocol      string   `json:"protocol"`
	BaseURL       string   `json:"base_url"`
	ModelsServed  []string `json:"models_served"`
	Weight        int      `json:"weight"`
	MaxConcurrent int      `json:"max_concurrent"`
	Enabled       bool     `json:"enabled"`
}

func (dto BackendDTO) ToDomain() model.Backend {
	var protocol model.Protocol
	switch dto.Protocol {
	case "h2c":
		protocol = model.ProtocolH2C
	case "h2":
		protocol = model.ProtocolH2
	default:
		protocol = model.ProtocolH1
	}

	models := make([]model.LargeLanguageModelID, len(dto.ModelsServed))
	for i, m := range dto.ModelsServed {
		models[i] = model.LargeLanguageModelID(m)
	}

	return model.Backend{
		ID:       model.BackendID(dto.Name),
		BaseURL:  dto.BaseURL,
		Protocol: protocol,
		Models:   models,
		Weight:   float64(dto.Weight),
	}
}

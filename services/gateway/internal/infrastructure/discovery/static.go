package discovery

import (
	"packages/lib/golang/shared/config"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

func LoadConfig(cfg config.BackendDiscovery) []model.Backend {
	backends := make([]model.Backend, len(cfg.Backends))
	for _, b := range cfg.Backends {
		var protocol model.Protocol
		switch b.Protocol {
		case "h2c":
			protocol = model.ProtocolH2C
		case "h2":
			protocol = model.ProtocolH2
		default:
			protocol = model.ProtocolH1
		}

		models := make([]model.LargeLanguageModelID, len(b.Models))
		for i, m := range b.Models {
			models[i] = model.LargeLanguageModelID(m)
		}

		backends = append(backends, model.Backend{
			ID:       model.BackendID(b.ID),
			BaseURL:  b.BaseURL,
			Protocol: protocol,
			Models:   models,
			Weight:   b.Weight,
		})
	}
	return backends
}

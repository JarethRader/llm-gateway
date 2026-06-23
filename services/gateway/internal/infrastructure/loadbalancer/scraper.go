package loadbalancer

import "github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"

func Scrape(_ model.Backend) {
	// TODO implement /metric scraper to collect LLM metrics per backend
	panic("unimplemented")
}

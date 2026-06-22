package dto

type Backend struct {
	ID                        *int64   `json:"id"`
	Name                      *string  `json:"name"`
	Protocol                  *string  `json:"protocol"`
	BaseURL                   *string  `json:"base_url"`
	Enabled                   *bool    `json:"enabled"`
	ModelsServed              []string `json:"models_served"`
	Weight                    *int     `json:"weight"`
	MaxConcurrent             *int     `json:"max_concurrent"`
	KVCacheAwareRouting       *bool    `json:"kv_cache_aware_routing"`
	MetricsURL                *string  `json:"metrics_url"`
	ScapeInterval             *int     `json:"scrape_interval"`
	MaxIdleConnectionsPerHost *int     `json:"max_idle_connections_per_host"`
	IdleConnectionTimeout     *int     `json:"idle_connection_timeout"`
	DialTimeout               *int     `json:"dial_timeout"`
	StreamStallTimeout        *int     `json:"stream_stall_timeout"`
	ResponseHeaderTimeout     *int     `json:"response_header_timeout"`
	FailureThreshold          *int     `json:"failure_threshold"`
	RollingWindow             *int     `json:"rolling_window"`
	OpenBase                  *int     `json:"open_base"`
	OpenMax                   *int     `json:"open_max"`
	BackoffFactor             *int     `json:"backoff_factor"`
	HalfOpenProbes            *int     `json:"half_open_probes"`
	HalfOpenSuccesses         *int     `json:"half_open_successes"`
	HealthCheckPath           *string  `json:"health_check_path"`
	HealthInterval            *int     `json:"health_interval"`
	VerifyTLSCert             *bool    `json:"verify_tls_cert"`
	Description               *string  `json:"description"`
	Labels                    []string `json:"labels"`
}

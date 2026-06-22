package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type Backend struct {
	ID                        int64         `json:"id" db:"id"`
	Name                      string        `json:"name" db:"name"`
	Protocol                  string        `json:"protocol" db:"protocol"`
	BaseURL                   string        `json:"base_url" db:"base_url"`
	Enabled                   bool          `json:"enabled" db:"enabled"`
	ModelsServed              StringSlice   `json:"models_served" db:"models_served"`
	Weight                    int           `json:"weight" db:"weight"`
	MaxConcurrent             int           `json:"max_concurrent" db:"max_concurrent"`
	KVCacheAwareRouting       bool          `json:"kv_cache_aware_routing" db:"kv_cache_aware_routing"`
	MetricsURL                *string       `json:"metrics_url" db:"metrics_url"`
	ScapeInterval             int           `json:"scrape_interval" db:"scrape_interval"`
	MaxIdleConnectionsPerHost int           `json:"max_idle_connections_per_host" db:"max_idle_connections_per_host"`
	IdleConnectionTimeout     time.Duration `json:"idle_connection_timeout" db:"idle_connection_timeout"`
	DialTimeout               time.Duration `json:"dial_timeout" db:"dial_timeout"`
	StreamStallTimeout        time.Duration `json:"stream_stall_timeout" db:"stream_stall_timeout"`
	ResponseHeaderTimeout     time.Duration `json:"response_header_timeout" db:"response_header_timeout"`
	FailureThreshold          int           `json:"failure_threshold" db:"failure_threshold"`
	RollingWindow             int           `json:"rolling_window" db:"rolling_window"`
	OpenBase                  int           `json:"open_base" db:"open_base"`
	OpenMax                   int           `json:"open_max" db:"open_max"`
	BackoffFactor             int           `json:"backoff_factor" db:"backoff_factor"`
	HalfOpenProbes            int           `json:"half_open_probes" db:"half_open_probes"`
	HalfOpenSuccesses         int           `json:"half_open_successes" db:"half_open_successes"`
	HealthCheckPath           string        `json:"health_check_path" db:"health_check_path"`
	HealthInterval            time.Duration `json:"health_interval" db:"health_interval"`
	VerifyTLSCert             bool          `json:"verify_tls_cert" db:"verify_tls_cert"`
	Description               *string       `json:"description" db:"description"`
	Labels                    StringSlice   `json:"labels" db:"labels"`
	CreatedAt                 Time          `json:"created_at" db:"created_at"`
	UpdatedAt                 Time          `json:"updated_at" db:"updated_at"`
}

type SparseBackend struct {
	ID            int64       `json:"id" db:"id"`
	Name          string      `json:"name" db:"name"`
	Protocol      string      `json:"protocol" db:"protocol"`
	BaseURL       string      `json:"base_url" db:"base_url"`
	Enabled       bool        `json:"enabled" db:"enabled"`
	ModelsServed  StringSlice `json:"models_served" db:"models_served"`
	Weight        int         `json:"weight" db:"weight"`
	MaxConcurrent int         `json:"max_concurrent" db:"max_concurrent"`
}

type StringSlice []string

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		b = []byte(value.(string))
	}
	return json.Unmarshal(b, s)
}

func (s StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

type Time time.Time

func (t *Time) Scan(value interface{}) error {
	if value == nil {
		*t = Time{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		b = []byte(value.(string))
	}
	parsed, err := time.Parse(time.RFC3339, string(b))
	if err != nil {
		return err
	}
	*t = Time(parsed)
	return nil
}

func (t Time) Value() (driver.Value, error) {
	return time.Time(t).Format(time.RFC3339), nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t))
}

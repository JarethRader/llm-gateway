package config

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator"
	"github.com/spf13/viper"
)

type (
	Config struct {
		CommitHash string
		Tag        string

		NS    string `mapstructure:"namespace"`
		Owner string `mapstructure:"owner"`

		App struct {
			Name        string `mapstructure:"name"`
			Version     string `mapstructure:"version"`
			Addr        string `mapstructure:"addr"`
			Port        string `mapstructure:"port"`
			LogLevel    string `mapstructure:"log_level"`
			Environment string `mapstructure:"environment"`
		}

		ShutdownWaitSec int `mapstructure:"shutdown_wait_sec"`

		Turso Turso `mapstructure:"turso"`

		Authentication   Auth             `mapstructure:"auth"`
		Telemetry        Telemetry        `mapstructure:"telemetry"`
		ConnectionPool   ConnectionPool   `mapstructure:"connection_pool"`
		SSEStreaming     SSEStreaming     `mapstructure:"sse_streaming"`
		LoadBalancer     LoadBalancer     `mapstructure:"load_balancer"`
		BackendDiscovery BackendDiscovery `mapstructure:"backend_discovery"`
		RateLimit        RateLimit        `mapstructure:"rate_limit"`
		Admission        Admission        `mapstructure:"admission"`
		Circuit          Circuit          `mapstructure:"circuit"`
	}

	Turso struct {
		URL   string `mapstructure:"url"`
		Token string `mapstructure:"token"`
	}

	Telemetry struct {
		Enabled     bool    `mapstructure:"enabled"`
		Endpoint    string  `mapstructure:"endpoint"`
		Insecure    bool    `mapstructure:"insecure" default:"false"`
		SampleRatio float64 `mapstructure:"sample_ratio" default:"1.0"`
	}

	Auth struct {
		Mode         string        `mapstructure:"mode"`
		KeyTablePath string        `mapstructure:"key_table_path"`
		ApiUrl       string        `mapstructure:"api_url"`
		Interval     time.Duration `mapstructure:"interval"`
	}

	ConnectionPool struct {
		TCPKeepAlive        time.Duration `mapstructure:"tcp_keep_alive"`
		MaxIdleConns        int           `mapstructure:"max_idle_conns"`
		MaxIdleConnsPerHost int           `mapstructure:"max_idle_conns_per_host"`
		MaxConnsPerHost     int           `mapstructure:"max_conns_per_host"`
		IdleConnTimeout     time.Duration `mapstructure:"idle_conn_timeout"`
		TLSHandshakeTimeout time.Duration `mapstructure:"tls_handshake_timeout"`
		WriteBufferSize     int           `mapstructure:"write_buffer_size"`
		ReadBufferSize      int           `mapstructure:"read_buffer_size"`
		H2ReadIdleTimeout   time.Duration `mapstructure:"h2_read_idle_timeout"`
		H2PingTimeout       time.Duration `mapstructure:"h2_ping_timeout"`
	}

	SSEStreaming struct {
		IdleTimeout       time.Duration `mapstructure:"idle_timeout"`       // max gap between upstream chunks
		HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"` // keepalive ping to client (0 = disable)
		FrameAware        bool          `mapstructure:"frame_aware"`        // parse SSE frame for token usage
		FlushEveryWrite   bool          `mapstructure:"flush_every_write"`
		MaxBodyBytes      int           `mapstructure:"max_body_bytes"`
	}

	LoadBalancer struct {
		Weights               RoutingWeights `mapstructure:"weights"`
		EwmaAlpha             float64        `mapstructure:"ewma_alpha"`
		MaxInFlightPerBackend int            `mapstructure:"max_in_flight_per_backend"`
		Scrape                MetricScrape   `mapstructure:"scrape"`
	}

	RoutingWeights struct {
		Latency   float64 `mapstructure:"latency"`
		InFlight  float64 `mapstructure:"in_flight"`
		Cache     float64 `mapstructure:"cache"`
		RefTTFTMS int     `mapstructure:"ref_ttft_ms"`
	}

	MetricScrape struct {
		Enabled  bool          `mapstructure:"enabled"`
		Interval time.Duration `mapstructure:"interval"`
		Timeout  time.Duration `mapstructure:"timeout"`
		CacheTTL time.Duration `mapstructure:"cache_ttl"`
	}

	BackendDiscovery struct {
		Mode     string        `mapstructure:"mode"`
		ApiUrl   string        `mapstructure:"api_url"`
		Interval time.Duration `mapstructure:"interval"`
		Backends []Backend     `mapstructure:"backends"`
	}

	Backend struct {
		ID       string   `mapstructure:"id"`
		BaseURL  string   `mapstructure:"base_url"`
		Protocol string   `mapstructure:"protocol"`
		Models   []string `mapstructure:"models"`
		Weight   float64  `mapstructure:"weight"`
	}

	RatePolicy struct {
		RatePerSec int `mapstructure:"rate_per_sec"`
		Burst      int `mapstructure:"burst"`
	}

	RateLimit struct {
		Enabled       bool                  `mapstructure:"enabled"`
		TokenWeighted bool                  `mapstructure:"token_weighted"`
		MaxRetryAfter time.Duration         `mapstructure:"max_retry_after"`
		SweepInterval time.Duration         `mapstructure:"sweep_interval"`
		IdleTTL       time.Duration         `mapstructure:"idle_ttl"`
		Global        RatePolicy            `mapstructure:"global"`
		DefaultModel  RatePolicy            `mapstructure:"default_model"`
		DefaultKey    RatePolicy            `mapstructure:"default_key"`
		PerModel      map[string]RatePolicy `mapstructure:"per_model"`
		PerKey        map[string]RatePolicy `mapstructure:"per_key"`
	}

	Admission struct {
		Enabled       bool          `mapstructure:"enabled"`
		MaxConcurrent int           `mapstructure:"max_concurrent"`
		MaxQueue      int           `mapstructure:"max_queue"`
		MaxRetryAfter time.Duration `mapstructure:"max_retry_after"`
		TokenWeighted bool          `mapstructure:"token_weighted"`
		EwmaAlpha     float64       `mapstructure:"ewma_alpha"`
	}

	Circuit struct {
		Enabled         bool          `mapstructure:"enabled"`
		FailureRatio    float64       `mapstructure:"failure_ratio"`
		MinRequests     int           `mapstructure:"min_requests"`
		Window          time.Duration `mapstructure:"window"`
		Buckets         int           `mapstructure:"buckets"`
		OpenBase        time.Duration `mapstructure:"open_base"`
		OpenMax         time.Duration `mapstructure:"open_max"`
		BackoffFactor   float64       `mapstructure:"backoff_factor"`
		HalfOpenMax     int32         `mapstructure:"half_open_max"`
		HalfOpenSuccess int32         `mapstructure:"half_open_success"`
		FailureStatuses []int         `mapstructure:"failure_statuses"`
	}
)

func Load(ctx context.Context, commitHash, tag string) (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AutomaticEnv()
	v.SetEnvPrefix("env")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %v", err)
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}

	c.CommitHash = commitHash
	c.Tag = tag

	validator := validator.New()
	if err := validator.Struct(c); err != nil {
		return nil, err
	}

	return &c, nil
}

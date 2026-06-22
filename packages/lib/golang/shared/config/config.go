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

		Telemetry      Telemetry      `mapstructure:"telemetry"`
		ConnectionPool ConnectionPool `mapstructure:"connection_pool"`
		SSEStreaming   SSEStreaming   `mapstructure:"sse_streaming"`
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

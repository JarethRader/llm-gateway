package config

import (
	"context"
	"log"
	"strings"

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

		Telemetry Telemetry `mapstructure:"telemetry"`
	}

	Telemetry struct {
		Enabled     bool    `mapstructure:"enabled"`
		Endpoint    string  `mapstructure:"endpoint"`
		Insecure    bool    `mapstructure:"insecure" default:"false"`
		SampleRatio float64 `mapstructure:"sample_ratio" default:"1.0"`
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

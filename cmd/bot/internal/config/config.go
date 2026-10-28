package config

import (
	"time"
)

type Config struct {
	LogLevel string  `yaml:"log_level" required:"true"`
	Tracing  Tracing `yaml:"tracing"   required:"true"`
	Bot      Bot     `yaml:"bot"  required:"true"`
	Storage  Storage `yaml:"storage"   required:"true"`
}

func (c *Config) Level() string {
	return c.LogLevel
}

func (c *Config) ServiceName() string {
	return c.Tracing.ServiceName
}

func (c *Config) TracingEndpoint() string {
	return c.Tracing.Endpoint
}

type Tracing struct {
	Endpoint    string `yaml:"endpoint"     required:"true"`
	ServiceName string `yaml:"service_name" required:"true"`
}

type Bot struct {
	Token string `env:"BOT_TELEGRAM_TOKEN" required:"true" yaml:"token"`
	// WebhookToken switches the bot to webhook mode (http server on :80), long polling is used when empty.
	WebhookToken string `yaml:"webhook_token"`
	// DownloadTimeout limits downloading of a single file from telegram.
	DownloadTimeout time.Duration `default:"60s" yaml:"download_timeout"`
	// ReadTimeout, WriteTimeout and ShutdownTimeout configure webhook http server.
	ReadTimeout     time.Duration `default:"10s" yaml:"read_timeout"`
	WriteTimeout    time.Duration `default:"10s" yaml:"write_timeout"`
	ShutdownTimeout time.Duration `default:"30s" yaml:"shutdown_timeout"`
}

type Storage struct {
	RootDir string `yaml:"root_dir" required:"true"`
}

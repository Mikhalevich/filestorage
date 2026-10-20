package config

import (
	"time"
)

type Config struct {
	LogLevel   string     `yaml:"log_level" required:"true"`
	Tracing    Tracing    `yaml:"tracing" required:"true"`
	HTTPServer HTTPServer `yaml:"http_server" required:"true"`
	Storage    Storage    `yaml:"storage" required:"true"`
	Limits     Limits     `yaml:"limits" required:"true"`
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
	Endpoint    string `yaml:"endpoint" required:"true"`
	ServiceName string `yaml:"service_name" required:"true"`
}

// HTTPServer has no write timeout: streamable http transport may keep SSE responses open.
type HTTPServer struct {
	Port            int           `yaml:"port" required:"true"`
	ReadTimeout     time.Duration `yaml:"read_timeout" required:"true"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" default:"60s"`
}

type Storage struct {
	RootDir string `yaml:"root_dir" required:"true"`
}

type Limits struct {
	// MaxReadBytes is the maximum size of a file returned by read_file tool.
	MaxReadBytes int64 `yaml:"max_read_bytes" required:"true"`
}

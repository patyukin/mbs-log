package config

import (
	"fmt"
	configLoader "github.com/patyukin/mbs-pkg/pkg/config"
)

type Config struct {
	MinLogLevel string `yaml:"min_log_level" validate:"required,oneof=debug info warn error"`
	HttpServer  struct {
		Port int `yaml:"port" validate:"required,numeric"`
	} `yaml:"http_server" validate:"required"`
	GRPCServer struct {
		Port              int `yaml:"port" validate:"required,numeric"`
		MaxConnectionIdle int `yaml:"max_connection_idle"`
		Timeout           int `yaml:"timeout"`
		MaxConnectionAge  int `yaml:"max_connection_age"`
	} `yaml:"grpc_server" validate:"required"`
	ClickhouseDsn string `yaml:"clickhouse_dsn" validate:"required"`
	RabbitMQURL   string `yaml:"rabbitmq_url" validate:"required"`
	Kafka         struct {
		Brokers       []string `yaml:"brokers" validate:"required"`
		ConsumerGroup string   `yaml:"consumer_group" validate:"required"`
		Topics        []string `yaml:"topics" validate:"required"`
	} `yaml:"kafka"`
	TracerHost string `yaml:"tracer_host" validate:"required"`
	S3         struct {
		Endpoint  string `yaml:"endpoint" validate:"required"`
		Bucket    string `yaml:"bucket" validate:"required"`
		AccessKey string `yaml:"access_key" validate:"required"`
		SecretKey string `yaml:"secret_key" validate:"required"`
	} `yaml:"s3"`
}

func LoadConfig() (*Config, error) {
	var config Config
	err := configLoader.LoadConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	return &config, nil
}

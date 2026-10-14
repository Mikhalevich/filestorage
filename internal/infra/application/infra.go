package application

import (
	"flag"
	"fmt"

	"github.com/jinzhu/configor"

	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

func LoadConfig(cfg any) error {
	configFile := flag.String("config", "config/config.yaml", "consumer worker config file")

	flag.Parse()

	if err := configor.Load(cfg, *configFile); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	return nil
}

func SetupLogger(lvl string) (logger.Logger, error) {
	log, err := logger.NewLogrusWithLevel(lvl)
	if err != nil {
		return nil, fmt.Errorf("creating new logger: %w", err)
	}

	logger.SetStdLogger(log)

	return log, nil
}

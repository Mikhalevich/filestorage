package application

import (
	"errors"
	"flag"
	"fmt"

	"github.com/jinzhu/configor"

	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

func LoadConfig(cfg any) error {
	configFile := flag.String("config", "", "path to the service config file (required)")

	flag.Parse()

	if *configFile == "" {
		return errors.New("-config flag is required")
	}

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

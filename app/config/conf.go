package config

import (
	"errors"
	sillyKits "github.com/irealing/silly-kits"
	"github.com/pelletier/go-toml/v2"
	"os"
)

const DefaultConfigFilename = "config.toml"

func LoadConfig(filename string) (*Config, error) {
	if filename == "" {
		filename = DefaultConfigFilename
	}
	return sillyKits.Apply(Default(), func(config *Config) (*Config, error) {
		return loadConfigFile(filename, config)
	}, func(config *Config) (*Config, error) {
		return writeDefaultConfig(filename, config)
	},
		initLogger, initTLSConfig,
	)
}
func loadConfigFile(filename string, config *Config) (*Config, error) {
	if _, err := os.Stat(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, nil
		}
		return config, err
	}
	if buf, err := os.ReadFile(filename); err != nil {
		return nil, err
	} else {
		return config, toml.Unmarshal(buf, config)
	}
}
func writeDefaultConfig(filename string, cfg *Config) (*Config, error) {
	if _, err := os.Stat(filename); err == nil || !os.IsNotExist(err) {
		return cfg, err
	}
	f, err := os.Create(filename)
	if err != nil {
		return cfg, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			cfg.logger.Warn("write default config error", err)
		}
	}()
	return cfg, toml.NewEncoder(f).Encode(cfg)
}
func initLogger(config *Config) (*Config, error) {
	config.logger = config.Log.makeLogger()
	return config, nil
}
func initTLSConfig(config *Config) (*Config, error) {
	cfg, err := config.TLS.makeTlsConfig()
	if err != nil {
		return nil, err
	}
	config.tlsConfig = cfg
	return config, nil
}

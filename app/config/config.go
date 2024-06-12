package config

import (
	"crypto/tls"
	"github.com/irealing/silly-ctrl"
	"github.com/irealing/silly-ctrl/internal/util"
	"gopkg.in/natefinch/lumberjack.v2"

	"io"
	"log/slog"
	"os"
)

type LogConf struct {
	Filename  string
	Level     slog.Level
	MaxSize   int
	MaxAge    int
	MaxBackup int
}

func (cfg *LogConf) makeLogger() *slog.Logger {
	var writer io.Writer
	if cfg.Filename == "" {
		writer = os.Stderr
	} else {
		writer = &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			MaxBackups: cfg.MaxBackup,
			LocalTime:  false,
			Compress:   false,
		}
	}
	h := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: cfg.Level})
	logger := slog.New(h)
	slog.SetDefault(logger)
	return logger
}

type Remote struct {
	App     util.App
	Address string
}
type Forward struct {
	Via           string
	App           string
	LocalAddress  string
	RemoteAddress string
}
type TLSConfig struct {
	PrivateKey string
	Cert       string
	SkipVerify bool
}

func (c TLSConfig) makeTlsConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(c.Cert, c.PrivateKey)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert}, InsecureSkipVerify: c.SkipVerify,
	}, nil
}

type Config struct {
	Remote    []Remote
	Apps      []util.App
	Ctrl      silly_ctrl.Config
	Log       LogConf
	TLS       TLSConfig
	Forward   []Forward
	logger    *slog.Logger
	tlsConfig *tls.Config
}

func (c *Config) TLSConfig() *tls.Config {
	return c.tlsConfig
}
func (c *Config) Logger() *slog.Logger {
	return c.logger
}

func Default() *Config {
	return &Config{
		Ctrl: *silly_ctrl.DefaultConfig(),
		Log: LogConf{
			Filename:  "",
			Level:     slog.LevelWarn,
			MaxSize:   3,
			MaxAge:    0,
			MaxBackup: 3,
		},
	}
}

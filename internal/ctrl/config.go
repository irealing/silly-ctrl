package ctrl

import (
	sillyKits "github.com/irealing/silly-kits"
	"time"
)

type Config struct {
	HeartbeatInterval    time.Duration `json:"heartbeat_interval"`     // 心跳时间间隔
	MaxHeartbeatInterval time.Duration `json:"max_heartbeat_interval"` // 最大心跳时间间隔
	LocalAddress         string        `json:"local_address"`
	ConnectionQueueSize  int           `json:"connection_queue_size"` // 连接队列的大小
	HandshakeTimeout     time.Duration `json:"handshake_timeout"`
}

func DefaultConfig() *Config {
	return &Config{
		HeartbeatInterval:    30,
		MaxHeartbeatInterval: 45,
		LocalAddress:         "127.0.0.1:0",
		ConnectionQueueSize:  10,
		HandshakeTimeout:     15,
	}
}
func (c *Config) Options(opt ...func(cfg *Config) (*Config, error)) (*Config, error) {
	return sillyKits.Apply(c, opt...)
}

package database

import (
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	Host                 string
	Port                 string
	User                 string
	Password             string
	Name                 string
	SSL                  string
	MaxOpenedConnections int
	MaxIdleConnections   int
	ConnMaxLifetime      time.Duration
}

const (
	defaultMaxRetries   = 10
	defaultInitialDelay = 1 * time.Second
	defaultMaxDelay     = 30 * time.Second
)

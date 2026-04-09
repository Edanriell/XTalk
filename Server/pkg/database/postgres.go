package database

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
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

func Connect(cfg Config, log *zap.Logger) (*sql.DB, error) {

}

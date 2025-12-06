package database

import (
	"database/sql"
	"fmt"
	"math"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Config struct {
	Host                  string
	Port                  string
	User                  string
	Password              string
	Name                  string
	SSL                   string
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
}

const (
	defaultMaxRetries   = 10
	defaultInitialDelay = 1 * time.Second
	defaultMaxDelay     = 30 * time.Second
)

func Connect(cfg Config, log *zap.Logger) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSL,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	maxOpen := cfg.MaxOpenConnections
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := cfg.MaxIdleConnections
	if maxIdle <= 0 {
		maxIdle = 5
	}
	maxLifetime := cfg.ConnectionMaxLifetime
	if maxLifetime <= 0 {
		maxLifetime = 5 * time.Minute
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)

	// Retry with exponential backoff
	for attempt := 0; attempt < defaultMaxRetries; attempt++ {
		if err = db.Ping(); err == nil {
			log.Info("connected to PostgreSQL", zap.String("host", cfg.Host), zap.String("db", cfg.Name))
			return db, nil
		}

		backoff := time.Duration(math.Min(
			float64(defaultInitialDelay)*math.Pow(2, float64(attempt)),
			float64(defaultMaxDelay),
		))
		log.Warn("database ping failed, retrying",
			zap.Int("attempt", attempt+1),
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)
		time.Sleep(backoff)
	}

	db.Close()
	return nil, fmt.Errorf("ping database after %d retries: %w", defaultMaxRetries, err)
}

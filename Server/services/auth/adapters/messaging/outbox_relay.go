package messaging

import (
	"XTalk/services/auth/application/interfaces"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// OutboxRelay publishes committed integration events with at-least-once
// delivery. Consumers must therefore be idempotent.
type OutboxRelay struct {
	db        *sql.DB
	publisher interfaces.EventPublisher
	interval  time.Duration
	log       *zap.Logger
}

func NewOutboxRelay(db *sql.DB, publisher interfaces.EventPublisher, log *zap.Logger) *OutboxRelay {
	return &OutboxRelay{db: db, publisher: publisher, interval: time.Second, log: log}
}

func (r *OutboxRelay) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			if err := r.publishBatch(ctx); err != nil && ctx.Err() == nil {
				r.log.Error("relay auth outbox", zap.Error(err))
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (r *OutboxRelay) publishBatch(ctx context.Context) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin outbox transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	rows, err := tx.QueryContext(ctx, `
		SELECT id, event_type, payload
		FROM auth_outbox
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT 50
		FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return fmt.Errorf("query auth outbox: %w", err)
	}

	type pendingEvent struct {
		id        int64
		eventType string
		payload   []byte
	}
	pending := make([]pendingEvent, 0, 50)
	for rows.Next() {
		var item pendingEvent
		if err = rows.Scan(&item.id, &item.eventType, &item.payload); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan auth outbox: %w", err)
		}
		pending = append(pending, item)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate auth outbox: %w", err)
	}
	if err = rows.Close(); err != nil {
		return fmt.Errorf("close auth outbox rows: %w", err)
	}

	for _, item := range pending {
		if item.eventType != "auth.user_registered" {
			return fmt.Errorf("unsupported auth outbox event %q", item.eventType)
		}
		var event interfaces.UserRegisteredEvent
		if err = json.Unmarshal(item.payload, &event); err != nil {
			return fmt.Errorf("decode auth outbox event %d: %w", item.id, err)
		}
		if err = r.publisher.PublishUserRegistered(ctx, event); err != nil {
			return fmt.Errorf("publish auth outbox event %d: %w", item.id, err)
		}
		if _, err = tx.ExecContext(ctx,
			`UPDATE auth_outbox SET published_at = CURRENT_TIMESTAMP WHERE id = $1`, item.id,
		); err != nil {
			return fmt.Errorf("mark auth outbox event %d published: %w", item.id, err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit auth outbox transaction: %w", err)
	}
	return nil
}

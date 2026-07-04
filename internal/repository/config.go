package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/amansharma/config-service/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("config not found")

type ConfigRepository struct {
	db *sqlx.DB
}

func NewConfigRepository(db *sqlx.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

func (r *ConfigRepository) GetByID(ctx context.Context, id string) (*model.Config, error) {
	var cfg model.Config
	err := r.db.GetContext(ctx, &cfg, "SELECT * FROM configs WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Upsert inserts or updates a config. Returns the resulting record and whether it was created (true) or updated (false).
func (r *ConfigRepository) Upsert(ctx context.Context, req model.UpsertConfigRequest) (*model.Config, bool, error) {
	query := `
		INSERT INTO configs (id, host, port, app_name, log_level)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			host = EXCLUDED.host,
			port = EXCLUDED.port,
			app_name = EXCLUDED.app_name,
			log_level = EXCLUDED.log_level,
			updated_at = NOW()
		RETURNING *, (xmax = 0) AS created`

	row := r.db.QueryRowxContext(ctx, query, req.ID, req.Host, req.Port, req.AppName, req.LogLevel)

	var cfg model.Config
	var created bool
	err := row.Scan(&cfg.ID, &cfg.Host, &cfg.Port, &cfg.AppName, &cfg.LogLevel, &cfg.CreatedAt, &cfg.UpdatedAt, &created)
	if err != nil {
		return nil, false, err
	}

	return &cfg, created, nil
}

func (r *ConfigRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

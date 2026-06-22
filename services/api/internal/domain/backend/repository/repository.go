package repository

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jarethrader/llm-gateway/api-service/internal/domain/backend"
	"github.com/jarethrader/llm-gateway/api-service/internal/models"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	lgr *slog.Logger
	db  *sqlx.DB
}

func NewRepository(db *sqlx.DB, lgr *slog.Logger) backend.Repository {
	return &repository{
		db:  db,
		lgr: lgr,
	}
}

// CreateBackend implements [backend.Repository].
func (r *repository) CreateBackend(ctx context.Context, backend models.Backend) (int64, error) {
	sql := `INSERT INTO backends
		(name, protocol, base_url, enabled, models_served, weight, max_concurrent, kv_cache_aware_routing, metrics_url, scrape_interval, max_idle_connections_per_host, idle_connection_timeout, dial_timeout, stream_stall_timeout, response_header_timeout, failure_threshold, rolling_window, open_base, open_max, backoff_factor, half_open_probes, half_open_successes, health_check_path, health_interval, verify_tls_cert, description, labels)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	modelsServedJSON, _ := json.Marshal(backend.ModelsServed)
	labelsJSON, _ := json.Marshal(backend.Labels)

	args := []any{
		backend.Name,
		backend.Protocol,
		backend.BaseURL,
		backend.Enabled,
		modelsServedJSON,
		backend.Weight,
		backend.MaxConcurrent,
		backend.KVCacheAwareRouting,
		backend.MetricsURL,
		backend.ScapeInterval,
		backend.MaxIdleConnectionsPerHost,
		backend.IdleConnectionTimeout,
		backend.DialTimeout,
		backend.StreamStallTimeout,
		backend.ResponseHeaderTimeout,
		backend.FailureThreshold,
		backend.RollingWindow,
		backend.OpenBase,
		backend.OpenMax,
		backend.BackoffFactor,
		backend.HalfOpenProbes,
		backend.HalfOpenSuccesses,
		backend.HealthCheckPath,
		backend.HealthInterval,
		backend.VerifyTLSCert,
		backend.Description,
		labelsJSON,
	}

	r.lgr.Debug("executing query", slog.String("sql", sql), slog.Any("args", args))
	res, err := r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		r.lgr.Error("failed to create backend config", slog.Any("error", err))
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		r.lgr.Error("failed to get last insert id", slog.Any("error", err))
		return 0, err
	}

	return id, nil
}

// DeleteBackend implements [backend.Repository].
func (r *repository) DeleteBackend(ctx context.Context, backendID int64) error {
	sql := `DELETE FROM backends WHERE id = ?;`

	r.lgr.Debug("executing query", slog.String("sql", sql), slog.Int64("backendID", backendID))
	if _, err := r.db.ExecContext(ctx, sql, backendID); err != nil {
		r.lgr.Error("failed to delete backend config", slog.Any("error", err))
		return err
	}

	return nil
}

// GetBackendByID implements [backend.Repository].
func (r *repository) GetBackendByID(ctx context.Context, backendID int64) (*models.Backend, error) {
	sql := `SELECT * FROM backends WHERE id = ?;`

	var backend models.Backend
	r.lgr.Debug("executing query", slog.String("sql", sql), slog.Int64("backendID", backendID))
	if err := r.db.GetContext(ctx, &backend, sql, backendID); err != nil {
		r.lgr.Error("failed to get backend by id", slog.Any("error", err))
		return nil, err
	}

	return &backend, nil
}

// SparseListBackends implements [backend.Repository].
func (r *repository) SparseListBackends(ctx context.Context) ([]models.SparseBackend, error) {
	sql := `SELECT id, name, protocol, base_url, enabled, models_served, weight, max_concurrent FROM backends;`

	backends := make([]models.SparseBackend, 0)
	r.lgr.Debug("executing query", slog.String("sql", sql))
	if err := r.db.SelectContext(ctx, &backends, sql); err != nil {
		r.lgr.Error("failed to get sparse details for backends", slog.Any("error", err))
		return nil, err
	}

	return backends, nil
}

// UpdateBackend implements [backend.Repository].
func (r *repository) UpdateBackend(ctx context.Context, backendID int64, backend models.Backend) error {
	sql := `UPDATE backends SET
			name = ?,
			protocol = ?,
			base_url = ?,
			enabled = ?,
			models_served = ?,
			weight = ?,
			max_concurrent = ?,
			kv_cache_aware_routing = ?,
			metrics_url = ?,
			scrape_interval = ?,
			max_idle_connections_per_host = ?,
			idle_connection_timeout = ?,
			dial_timeout = ?,
			stream_stall_timeout = ?,
			response_header_timeout = ?,
			failure_threshold = ?,
			rolling_window = ?,
			open_base = ?,
			open_max = ?,
			backoff_factor = ?,
			half_open_probes = ?,
			half_open_successes = ?,
			health_check_path = ?,
			health_interval = ?,
			verify_tls_cert = ?,
			description = ?,
			labels = ?,
			updated_at = (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		WHERE id = ?`

	args := []any{
		backend.Name,
		backend.Protocol,
		backend.BaseURL,
		backend.Enabled,
		backend.ModelsServed,
		backend.Weight,
		backend.MaxConcurrent,
		backend.KVCacheAwareRouting,
		backend.MetricsURL,
		backend.ScapeInterval,
		backend.MaxIdleConnectionsPerHost,
		backend.IdleConnectionTimeout,
		backend.DialTimeout,
		backend.StreamStallTimeout,
		backend.ResponseHeaderTimeout,
		backend.FailureThreshold,
		backend.RollingWindow,
		backend.OpenBase,
		backend.OpenMax,
		backend.BackoffFactor,
		backend.HalfOpenProbes,
		backend.HalfOpenSuccesses,
		backend.HealthCheckPath,
		backend.HealthInterval,
		backend.VerifyTLSCert,
		backend.Description,
		backend.Labels,
		backendID,
	}

	r.lgr.Debug("executing query", slog.String("sql", sql), slog.Int64("backendID", backendID))
	if _, err := r.db.ExecContext(ctx, sql, args...); err != nil {
		r.lgr.Error("failed to update backend config", slog.Any("error", err))
		return err
	}

	return nil
}

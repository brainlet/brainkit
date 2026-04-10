package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/internal/store/sqlgen/postgres"
	"github.com/brainlet/brainkit/internal/types"
	_ "github.com/lib/pq"
)

// PostgresKitStore implements types.KitStore using sqlc-generated Postgres queries.
type PostgresKitStore struct {
	db      *sql.DB
	queries *pggen.Queries
}

// NewPostgresKitStore creates a new Postgres-backed KitStore.
func NewPostgresKitStore(connStr string) (*PostgresKitStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("kitstore-pg: open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("kitstore-pg: ping: %w", err)
	}

	schemaSQL, _ := postgresSchema()
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("kitstore-pg: create tables: %w", err)
	}

	return &PostgresKitStore{db: db, queries: pggen.New(db)}, nil
}

func (s *PostgresKitStore) Close() error { return s.db.Close() }

// --- Deployments ---

func (s *PostgresKitStore) SaveDeployment(d types.PersistedDeployment) error {
	return s.queries.SaveDeployment(ctx(), pggen.SaveDeploymentParams{
		Source: d.Source, Code: d.Code, DeployOrder: int32(d.Order),
		DeployedAt: d.DeployedAt, PackageName: d.PackageName,
	})
}

func (s *PostgresKitStore) LoadDeployments() ([]types.PersistedDeployment, error) {
	rows, err := s.queries.LoadDeployments(ctx())
	if err != nil {
		return nil, err
	}
	result := make([]types.PersistedDeployment, len(rows))
	for i, r := range rows {
		result[i] = types.PersistedDeployment{
			Source: r.Source, Code: r.Code, Order: int(r.DeployOrder),
			DeployedAt: r.DeployedAt, PackageName: r.PackageName,
		}
	}
	return result, nil
}

func (s *PostgresKitStore) LoadDeployment(source string) (types.PersistedDeployment, error) {
	r, err := s.queries.LoadDeployment(ctx(), source)
	if err != nil {
		return types.PersistedDeployment{}, err
	}
	return types.PersistedDeployment{
		Source: r.Source, Code: r.Code, Order: int(r.DeployOrder),
		DeployedAt: r.DeployedAt, PackageName: r.PackageName,
	}, nil
}

func (s *PostgresKitStore) DeleteDeployment(source string) error {
	return s.queries.DeleteDeployment(ctx(), source)
}

// --- Schedules ---

func (s *PostgresKitStore) SaveSchedule(sc types.PersistedSchedule) error {
	return s.queries.SaveSchedule(ctx(), pggen.SaveScheduleParams{
		ID: sc.ID, Expression: sc.Expression, DurationNs: int64(sc.Duration),
		Topic: sc.Topic, Payload: string(sc.Payload), Source: sc.Source,
		CreatedAt: sc.CreatedAt, NextFire: sc.NextFire, OneTime: sc.OneTime,
	})
}

func (s *PostgresKitStore) LoadSchedules() ([]types.PersistedSchedule, error) {
	rows, err := s.queries.LoadSchedules(ctx())
	if err != nil {
		return nil, err
	}
	result := make([]types.PersistedSchedule, len(rows))
	for i, r := range rows {
		result[i] = types.PersistedSchedule{
			ID: r.ID, Expression: r.Expression, Duration: time.Duration(r.DurationNs),
			Topic: r.Topic, Payload: json.RawMessage(r.Payload), Source: r.Source,
			CreatedAt: r.CreatedAt, NextFire: r.NextFire, OneTime: r.OneTime,
		}
	}
	return result, nil
}

func (s *PostgresKitStore) DeleteSchedule(id string) error {
	return s.queries.DeleteSchedule(ctx(), id)
}

// --- Schedule Fires ---

func (s *PostgresKitStore) ClaimScheduleFire(scheduleID string, fireTime time.Time) (bool, error) {
	truncated := fireTime.Truncate(100 * time.Millisecond).Format(time.RFC3339Nano)
	err := s.queries.ClaimScheduleFire(ctx(), pggen.ClaimScheduleFireParams{
		ScheduleID: scheduleID, FireTime: truncated, ClaimedAt: time.Now(),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// --- Installed Plugins ---

func (s *PostgresKitStore) SaveInstalledPlugin(p types.InstalledPlugin) error {
	return s.queries.SaveInstalledPlugin(ctx(), pggen.SaveInstalledPluginParams{
		Name: p.Name, Owner: p.Owner, Version: p.Version,
		BinaryPath: p.BinaryPath, Manifest: p.Manifest, InstalledAt: p.InstalledAt,
	})
}

func (s *PostgresKitStore) LoadInstalledPlugins() ([]types.InstalledPlugin, error) {
	rows, err := s.queries.LoadInstalledPlugins(ctx())
	if err != nil {
		return nil, err
	}
	result := make([]types.InstalledPlugin, len(rows))
	for i, r := range rows {
		result[i] = types.InstalledPlugin{
			Name: r.Name, Owner: r.Owner, Version: r.Version,
			BinaryPath: r.BinaryPath, Manifest: r.Manifest, InstalledAt: r.InstalledAt,
		}
	}
	return result, nil
}

func (s *PostgresKitStore) DeleteInstalledPlugin(name string) error {
	return s.queries.DeleteInstalledPlugin(ctx(), name)
}

// --- Running Plugins ---

func (s *PostgresKitStore) SaveRunningPlugin(p types.RunningPluginRecord) error {
	envJSON, _ := json.Marshal(p.Env)
	configStr := "{}"
	if p.Config != nil {
		configStr = string(p.Config)
	}
	return s.queries.SaveRunningPlugin(ctx(), pggen.SaveRunningPluginParams{
		Name: p.Name, Owner: p.Owner, Version: p.Version,
		BinaryPath: p.BinaryPath, Env: string(envJSON), Config: configStr,
		StartOrder: int32(p.StartOrder), StartedAt: p.StartedAt,
	})
}

func (s *PostgresKitStore) LoadRunningPlugins() ([]types.RunningPluginRecord, error) {
	rows, err := s.queries.LoadRunningPlugins(ctx())
	if err != nil {
		return nil, err
	}
	result := make([]types.RunningPluginRecord, len(rows))
	for i, r := range rows {
		var env map[string]string
		json.Unmarshal([]byte(r.Env), &env)
		result[i] = types.RunningPluginRecord{
			Name: r.Name, Owner: r.Owner, Version: r.Version,
			BinaryPath: r.BinaryPath, Env: env, Config: json.RawMessage(r.Config),
			StartOrder: int(r.StartOrder), StartedAt: r.StartedAt,
		}
	}
	return result, nil
}

func (s *PostgresKitStore) DeleteRunningPlugin(name string) error {
	return s.queries.DeleteRunningPlugin(ctx(), name)
}

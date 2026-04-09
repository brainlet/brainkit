// Package store provides configurable persistence backends for KitStore and AuditStore.
// Supports SQLite (default, zero-config) and PostgreSQL (production scale).
// PostgreSQL uses sqlc-generated queries. SQLite uses database/sql directly
// (sqlc's SQLite engine has a codegen bug with tables having 10+ columns).
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	_ "modernc.org/sqlite"
)

// SQLiteKitStore implements types.KitStore using database/sql with embedded schema.
type SQLiteKitStore struct {
	db *sql.DB
}

func NewSQLiteKitStore(path string) (*SQLiteKitStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("kitstore: create directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("kitstore: open: %w", err)
	}
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA busy_timeout=5000")

	if _, err := db.Exec(sqliteSchemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("kitstore: create tables: %w", err)
	}

	return &SQLiteKitStore{db: db}, nil
}

func (s *SQLiteKitStore) Close() error { return s.db.Close() }
func (s *SQLiteKitStore) DB() *sql.DB  { return s.db }

// --- Deployments ---

func (s *SQLiteKitStore) SaveDeployment(d types.PersistedDeployment) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO deployments (source, code, deploy_order, deployed_at, package_name, role) VALUES (?, ?, ?, ?, ?, ?)`,
		d.Source, d.Code, d.Order, d.DeployedAt.Format(time.RFC3339), d.PackageName, d.Role,
	)
	return err
}

func (s *SQLiteKitStore) LoadDeployments() ([]types.PersistedDeployment, error) {
	rows, err := s.db.Query("SELECT source, code, deploy_order, deployed_at, package_name, role FROM deployments ORDER BY deploy_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []types.PersistedDeployment
	for rows.Next() {
		var d types.PersistedDeployment
		var ts string
		if err := rows.Scan(&d.Source, &d.Code, &d.Order, &ts, &d.PackageName, &d.Role); err != nil {
			return nil, err
		}
		d.DeployedAt, _ = time.Parse(time.RFC3339, ts)
		result = append(result, d)
	}
	if result == nil {
		result = []types.PersistedDeployment{}
	}
	return result, rows.Err()
}

func (s *SQLiteKitStore) LoadDeployment(source string) (types.PersistedDeployment, error) {
	var d types.PersistedDeployment
	var ts string
	err := s.db.QueryRow("SELECT source, code, deploy_order, deployed_at, package_name, role FROM deployments WHERE source = ?", source).
		Scan(&d.Source, &d.Code, &d.Order, &ts, &d.PackageName, &d.Role)
	if err != nil {
		return d, err
	}
	d.DeployedAt, _ = time.Parse(time.RFC3339, ts)
	return d, nil
}

func (s *SQLiteKitStore) DeleteDeployment(source string) error {
	_, err := s.db.Exec("DELETE FROM deployments WHERE source = ?", source)
	return err
}

// --- Schedules ---

func (s *SQLiteKitStore) SaveSchedule(sc types.PersistedSchedule) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO schedules (id, expression, duration_ns, topic, payload, source, created_at, next_fire, one_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sc.ID, sc.Expression, int64(sc.Duration), sc.Topic, string(sc.Payload),
		sc.Source, sc.CreatedAt.Format(time.RFC3339), sc.NextFire.Format(time.RFC3339), boolToInt(sc.OneTime),
	)
	return err
}

func (s *SQLiteKitStore) LoadSchedules() ([]types.PersistedSchedule, error) {
	rows, err := s.db.Query("SELECT id, expression, duration_ns, topic, payload, source, created_at, next_fire, one_time FROM schedules")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []types.PersistedSchedule
	for rows.Next() {
		var sc types.PersistedSchedule
		var durNs int64
		var ca, nf, payloadStr string
		var oneTime int64
		if err := rows.Scan(&sc.ID, &sc.Expression, &durNs, &sc.Topic, &payloadStr, &sc.Source, &ca, &nf, &oneTime); err != nil {
			return nil, err
		}
		sc.Duration = time.Duration(durNs)
		sc.Payload = json.RawMessage(payloadStr)
		sc.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		sc.NextFire, _ = time.Parse(time.RFC3339, nf)
		sc.OneTime = oneTime != 0
		result = append(result, sc)
	}
	if result == nil {
		result = []types.PersistedSchedule{}
	}
	return result, rows.Err()
}

func (s *SQLiteKitStore) DeleteSchedule(id string) error {
	_, err := s.db.Exec("DELETE FROM schedules WHERE id = ?", id)
	return err
}

// --- Schedule Fires ---

func (s *SQLiteKitStore) ClaimScheduleFire(scheduleID string, fireTime time.Time) (bool, error) {
	truncated := fireTime.Truncate(100 * time.Millisecond).Format(time.RFC3339Nano)
	result, err := s.db.Exec("INSERT OR IGNORE INTO schedule_fires (schedule_id, fire_time, claimed_at) VALUES (?, ?, ?)",
		scheduleID, truncated, time.Now().Format(time.RFC3339))
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

// --- Installed Plugins ---

func (s *SQLiteKitStore) SaveInstalledPlugin(p types.InstalledPlugin) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO installed_plugins (name, owner, version, binary_path, manifest, installed_at) VALUES (?, ?, ?, ?, ?, ?)`,
		p.Name, p.Owner, p.Version, p.BinaryPath, p.Manifest, p.InstalledAt.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteKitStore) LoadInstalledPlugins() ([]types.InstalledPlugin, error) {
	rows, err := s.db.Query("SELECT name, owner, version, binary_path, manifest, installed_at FROM installed_plugins")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []types.InstalledPlugin
	for rows.Next() {
		var p types.InstalledPlugin
		var ts string
		if err := rows.Scan(&p.Name, &p.Owner, &p.Version, &p.BinaryPath, &p.Manifest, &ts); err != nil {
			return nil, err
		}
		p.InstalledAt, _ = time.Parse(time.RFC3339, ts)
		result = append(result, p)
	}
	if result == nil {
		result = []types.InstalledPlugin{}
	}
	return result, rows.Err()
}

func (s *SQLiteKitStore) DeleteInstalledPlugin(name string) error {
	_, err := s.db.Exec("DELETE FROM installed_plugins WHERE name = ?", name)
	return err
}

// --- Running Plugins ---

func (s *SQLiteKitStore) SaveRunningPlugin(p types.RunningPluginRecord) error {
	envJSON, _ := json.Marshal(p.Env)
	configStr := "{}"
	if p.Config != nil {
		configStr = string(p.Config)
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO running_plugins (name, owner, version, binary_path, env, config, start_order, started_at, role) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Owner, p.Version, p.BinaryPath, string(envJSON), configStr, p.StartOrder, p.StartedAt.Format(time.RFC3339), p.Role,
	)
	return err
}

func (s *SQLiteKitStore) LoadRunningPlugins() ([]types.RunningPluginRecord, error) {
	rows, err := s.db.Query("SELECT name, owner, version, binary_path, env, config, start_order, started_at, role FROM running_plugins ORDER BY start_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []types.RunningPluginRecord
	for rows.Next() {
		var r types.RunningPluginRecord
		var envStr, configStr, ts string
		if err := rows.Scan(&r.Name, &r.Owner, &r.Version, &r.BinaryPath, &envStr, &configStr, &r.StartOrder, &ts, &r.Role); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(envStr), &r.Env)
		r.Config = json.RawMessage(configStr)
		r.StartedAt, _ = time.Parse(time.RFC3339, ts)
		result = append(result, r)
	}
	if result == nil {
		result = []types.RunningPluginRecord{}
	}
	return result, rows.Err()
}

func (s *SQLiteKitStore) DeleteRunningPlugin(name string) error {
	_, err := s.db.Exec("DELETE FROM running_plugins WHERE name = ?", name)
	return err
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

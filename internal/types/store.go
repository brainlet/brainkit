package types

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// KitStore provides optional persistence for deployments, schedules, and plugins.
// When configured on a Kit, data survives Kit restarts.
type KitStore interface {
	// Deployments
	SaveDeployment(d PersistedDeployment) error
	LoadDeployments() ([]PersistedDeployment, error)
	LoadDeployment(source string) (PersistedDeployment, error)
	DeleteDeployment(source string) error

	// Schedules
	SaveSchedule(s PersistedSchedule) error
	LoadSchedules() ([]PersistedSchedule, error)
	DeleteSchedule(id string) error

	// Schedule deduplication (for multi-replica)
	// ClaimScheduleFire atomically claims a schedule fire.
	// Returns true if this replica claimed it, false if another already did.
	ClaimScheduleFire(scheduleID string, fireTime time.Time) (bool, error)

	// Installed plugins
	SaveInstalledPlugin(p InstalledPlugin) error
	LoadInstalledPlugins() ([]InstalledPlugin, error)
	DeleteInstalledPlugin(name string) error

	// Running plugins (for restart recovery)
	SaveRunningPlugin(p RunningPluginRecord) error
	LoadRunningPlugins() ([]RunningPluginRecord, error)
	DeleteRunningPlugin(name string) error

	// Plugin KV state (per-plugin key-value storage)
	SavePluginState(pluginID, key, value string) error
	LoadPluginState(pluginID, key string) (string, error)
	DeletePluginState(pluginID string) error

	// Lifecycle
	Close() error
}

// PersistedDeployment is the on-disk format for a .ts deployment.
type PersistedDeployment struct {
	Source      string    `json:"source"`
	Code        string    `json:"code"`
	Order       int       `json:"order"`
	DeployedAt  time.Time `json:"deployedAt"`
	Role        string    `json:"role,omitempty"`
	PackageName string    `json:"packageName,omitempty"`
}

// PersistedSchedule is the on-disk format for a scheduled bus message.
type PersistedSchedule struct {
	ID         string          `json:"id"`
	Expression string          `json:"expression"`
	Duration   time.Duration   `json:"duration"`
	Topic      string          `json:"topic"`
	Payload    json.RawMessage `json:"payload"`
	Source     string          `json:"source"`
	CreatedAt  time.Time       `json:"createdAt"`
	NextFire   time.Time       `json:"nextFire"`
	OneTime    bool            `json:"oneTime"`
}

// ---------------------------------------------------------------------------
// SQLiteStore — pure Go SQLite-backed KitStore
// ---------------------------------------------------------------------------

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS deployments (
    source TEXT PRIMARY KEY,
    code TEXT NOT NULL,
    deploy_order INTEGER NOT NULL DEFAULT 0,
    deployed_at TEXT NOT NULL,
    package_name TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'service'
);

CREATE TABLE IF NOT EXISTS schedules (
    id TEXT PRIMARY KEY,
    expression TEXT NOT NULL,
    duration_ns INTEGER NOT NULL,
    topic TEXT NOT NULL,
    payload TEXT NOT NULL,
    source TEXT NOT NULL,
    created_at TEXT NOT NULL,
    next_fire TEXT NOT NULL,
    one_time INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS installed_plugins (
    name TEXT NOT NULL,
    owner TEXT NOT NULL,
    version TEXT NOT NULL,
    binary_path TEXT NOT NULL,
    manifest TEXT NOT NULL,
    installed_at TEXT NOT NULL,
    PRIMARY KEY (owner, name)
);

CREATE TABLE IF NOT EXISTS running_plugins (
    name TEXT PRIMARY KEY,
    owner TEXT NOT NULL,
    version TEXT NOT NULL,
    binary_path TEXT NOT NULL,
    env TEXT NOT NULL,
    config TEXT NOT NULL DEFAULT '{}',
    start_order INTEGER NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'service'
);

CREATE TABLE IF NOT EXISTS plugin_state (
    plugin_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (plugin_id, key)
);
CREATE TABLE IF NOT EXISTS schedule_fires (
    schedule_id TEXT NOT NULL,
    fire_time   TEXT NOT NULL,
    claimed_at  TEXT NOT NULL,
    PRIMARY KEY (schedule_id, fire_time)
);
`

// SQLiteStore implements KitStore using pure Go SQLite (modernc.org/sqlite).
type SQLiteStore struct {
	DB *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed store at the given file path.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("kitstore: create directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("kitstore: open: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("kitstore: set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("kitstore: set synchronous: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("kitstore: set busy timeout: %w", err)
	}

	if _, err := db.Exec(sqliteSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("kitstore: create tables: %w", err)
	}

	return &SQLiteStore{DB: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.DB.Close()
}

// --- Deployments ---

func (s *SQLiteStore) SaveDeployment(d PersistedDeployment) error {
	_, err := s.DB.Exec(
		`INSERT OR REPLACE INTO deployments (source, code, deploy_order, deployed_at, package_name, role)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		d.Source, d.Code, d.Order, d.DeployedAt.Format(time.RFC3339), d.PackageName, d.Role,
	)
	return err
}

func (s *SQLiteStore) LoadDeployments() ([]PersistedDeployment, error) {
	rows, err := s.DB.Query("SELECT source, code, deploy_order, deployed_at, package_name, role FROM deployments ORDER BY deploy_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []PersistedDeployment
	for rows.Next() {
		var d PersistedDeployment
		var deployedAtStr string
		if err := rows.Scan(&d.Source, &d.Code, &d.Order, &deployedAtStr, &d.PackageName, &d.Role); err != nil {
			return nil, err
		}
		d.DeployedAt, _ = time.Parse(time.RFC3339, deployedAtStr)
		deployments = append(deployments, d)
	}
	return deployments, rows.Err()
}

func (s *SQLiteStore) DeleteDeployment(source string) error {
	_, err := s.DB.Exec("DELETE FROM deployments WHERE source = ?", source)
	return err
}

func (s *SQLiteStore) LoadDeployment(source string) (PersistedDeployment, error) {
	row := s.DB.QueryRow("SELECT source, code, deploy_order, deployed_at, package_name, role FROM deployments WHERE source = ?", source)
	var d PersistedDeployment
	var deployedAtStr string
	if err := row.Scan(&d.Source, &d.Code, &d.Order, &deployedAtStr, &d.PackageName, &d.Role); err != nil {
		return d, err
	}
	d.DeployedAt, _ = time.Parse(time.RFC3339, deployedAtStr)
	return d, nil
}

// --- Schedule deduplication ---

func (s *SQLiteStore) ClaimScheduleFire(scheduleID string, fireTime time.Time) (bool, error) {
	// Truncate to 100ms — fine enough for sub-second schedules, coarse enough for replica dedup.
	truncated := fireTime.Truncate(100 * time.Millisecond).Format(time.RFC3339Nano)
	result, err := s.DB.Exec(
		"INSERT OR IGNORE INTO schedule_fires (schedule_id, fire_time, claimed_at) VALUES (?, ?, ?)",
		scheduleID, truncated, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

// --- Schedules ---

func (s *SQLiteStore) SaveSchedule(sc PersistedSchedule) error {
	_, err := s.DB.Exec(
		`INSERT OR REPLACE INTO schedules (id, expression, duration_ns, topic, payload, source, created_at, next_fire, one_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sc.ID, sc.Expression, int64(sc.Duration), sc.Topic, string(sc.Payload),
		sc.Source, sc.CreatedAt.Format(time.RFC3339), sc.NextFire.Format(time.RFC3339),
		boolToInt(sc.OneTime),
	)
	return err
}

func (s *SQLiteStore) LoadSchedules() ([]PersistedSchedule, error) {
	rows, err := s.DB.Query("SELECT id, expression, duration_ns, topic, payload, source, created_at, next_fire, one_time FROM schedules")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []PersistedSchedule
	for rows.Next() {
		var sc PersistedSchedule
		var durationNs int64
		var payloadStr, createdAtStr, nextFireStr string
		var oneTimeInt int
		if err := rows.Scan(&sc.ID, &sc.Expression, &durationNs, &sc.Topic, &payloadStr,
			&sc.Source, &createdAtStr, &nextFireStr, &oneTimeInt); err != nil {
			return nil, err
		}
		sc.Duration = time.Duration(durationNs)
		sc.Payload = json.RawMessage(payloadStr)
		sc.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		sc.NextFire, _ = time.Parse(time.RFC3339, nextFireStr)
		sc.OneTime = oneTimeInt != 0
		schedules = append(schedules, sc)
	}
	return schedules, rows.Err()
}

func (s *SQLiteStore) DeleteSchedule(id string) error {
	_, err := s.DB.Exec("DELETE FROM schedules WHERE id = ?", id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- Installed Plugins ---

func (s *SQLiteStore) SaveInstalledPlugin(p InstalledPlugin) error {
	_, err := s.DB.Exec(
		`INSERT OR REPLACE INTO installed_plugins (name, owner, version, binary_path, manifest, installed_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.Name, p.Owner, p.Version, p.BinaryPath, p.Manifest, p.InstalledAt.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) LoadInstalledPlugins() ([]InstalledPlugin, error) {
	rows, err := s.DB.Query("SELECT name, owner, version, binary_path, manifest, installed_at FROM installed_plugins")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var plugins []InstalledPlugin
	for rows.Next() {
		var p InstalledPlugin
		var installedAtStr string
		if err := rows.Scan(&p.Name, &p.Owner, &p.Version, &p.BinaryPath, &p.Manifest, &installedAtStr); err != nil {
			return nil, err
		}
		p.InstalledAt, _ = time.Parse(time.RFC3339, installedAtStr)
		plugins = append(plugins, p)
	}
	return plugins, rows.Err()
}

func (s *SQLiteStore) DeleteInstalledPlugin(name string) error {
	_, err := s.DB.Exec("DELETE FROM installed_plugins WHERE name = ?", name)
	return err
}

// --- Running Plugins ---

func (s *SQLiteStore) SaveRunningPlugin(p RunningPluginRecord) error {
	envJSON, _ := json.Marshal(p.Env)
	configStr := "{}"
	if len(p.Config) > 0 {
		configStr = string(p.Config)
	}
	role := p.Role
	if role == "" {
		role = "service"
	}
	_, err := s.DB.Exec(
		`INSERT OR REPLACE INTO running_plugins (name, owner, version, binary_path, env, config, start_order, started_at, role)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Owner, p.Version, p.BinaryPath, string(envJSON), configStr,
		p.StartOrder, p.StartedAt.Format(time.RFC3339), role,
	)
	return err
}

func (s *SQLiteStore) LoadRunningPlugins() ([]RunningPluginRecord, error) {
	rows, err := s.DB.Query("SELECT name, owner, version, binary_path, env, config, start_order, started_at, role FROM running_plugins ORDER BY start_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var plugins []RunningPluginRecord
	for rows.Next() {
		var p RunningPluginRecord
		var envStr, configStr, startedAtStr string
		if err := rows.Scan(&p.Name, &p.Owner, &p.Version, &p.BinaryPath, &envStr, &configStr, &p.StartOrder, &startedAtStr, &p.Role); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(envStr), &p.Env)
		p.Config = json.RawMessage(configStr)
		p.StartedAt, _ = time.Parse(time.RFC3339, startedAtStr)
		plugins = append(plugins, p)
	}
	return plugins, rows.Err()
}

func (s *SQLiteStore) DeleteRunningPlugin(name string) error {
	_, err := s.DB.Exec("DELETE FROM running_plugins WHERE name = ?", name)
	return err
}

// --- Plugin KV State ---

func (s *SQLiteStore) SavePluginState(pluginID, key, value string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.DB.Exec(
		`INSERT OR REPLACE INTO plugin_state (plugin_id, key, value, updated_at)
		 VALUES (?, ?, ?, ?)`,
		pluginID, key, value, now,
	)
	return err
}

func (s *SQLiteStore) LoadPluginState(pluginID, key string) (string, error) {
	var value string
	err := s.DB.QueryRow(
		"SELECT value FROM plugin_state WHERE plugin_id = ? AND key = ?",
		pluginID, key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *SQLiteStore) DeletePluginState(pluginID string) error {
	_, err := s.DB.Exec("DELETE FROM plugin_state WHERE plugin_id = ?", pluginID)
	return err
}

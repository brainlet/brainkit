package kit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// KitStore provides optional persistence for WASM modules, shard descriptors, and shard state.
// When configured on a Kit, data survives Kit restarts.
type KitStore interface {
	// Modules
	SaveModule(name string, binary []byte, info WASMModuleInfo) error
	LoadModules() (map[string]*WASMModule, error)
	DeleteModule(name string) error

	// Shards
	SaveShard(name string, desc ShardDescriptor) error
	LoadShards() (map[string]ShardDescriptor, error)
	DeleteShard(name string) error

	// Shard State
	SaveState(shardName, key string, state map[string]string) error
	LoadState(shardName, key string) (map[string]string, error)
	DeleteState(shardName string) error // delete all state for a shard

	// Deployments
	SaveDeployment(d PersistedDeployment) error
	LoadDeployments() ([]PersistedDeployment, error)
	DeleteDeployment(source string) error

	// Lifecycle
	Close() error
}

// PersistedDeployment is the on-disk format for a .ts deployment.
type PersistedDeployment struct {
	Source     string    `json:"source"`
	Code       string    `json:"code"`
	Order      int       `json:"order"`
	DeployedAt time.Time `json:"deployedAt"`
}

// ---------------------------------------------------------------------------
// SQLiteStore — pure Go SQLite-backed KitStore
// ---------------------------------------------------------------------------

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS wasm_modules (
    name TEXT PRIMARY KEY,
    binary BLOB NOT NULL,
    source_hash TEXT NOT NULL,
    exports TEXT NOT NULL,
    size INTEGER NOT NULL,
    compiled_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS wasm_shards (
    name TEXT PRIMARY KEY,
    mode TEXT NOT NULL,
    state_key TEXT NOT NULL DEFAULT '',
    handlers TEXT NOT NULL,
    deployed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS shard_state (
    shard_name TEXT NOT NULL,
    state_key TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (shard_name, state_key)
);

CREATE TABLE IF NOT EXISTS deployments (
    source TEXT PRIMARY KEY,
    code TEXT NOT NULL,
    deploy_order INTEGER NOT NULL DEFAULT 0,
    deployed_at TEXT NOT NULL
);
`

// SQLiteStore implements KitStore using pure Go SQLite (modernc.org/sqlite).
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed store at the given file path.
// Creates the directory and file if they don't exist.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("kitstore: create directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("kitstore: open: %w", err)
	}

	// Configure SQLite for concurrent access
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

	// Create tables
	if _, err := db.Exec(sqliteSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("kitstore: create tables: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// ---------------------------------------------------------------------------
// Modules
// ---------------------------------------------------------------------------

func (s *SQLiteStore) SaveModule(name string, binary []byte, info WASMModuleInfo) error {
	exportsJSON, _ := json.Marshal(info.Exports)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO wasm_modules (name, binary, source_hash, exports, size, compiled_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		name, binary, info.SourceHash, string(exportsJSON), info.Size, info.CompiledAt,
	)
	return err
}

func (s *SQLiteStore) LoadModules() (map[string]*WASMModule, error) {
	rows, err := s.db.Query("SELECT name, binary, source_hash, exports, size, compiled_at FROM wasm_modules")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	modules := make(map[string]*WASMModule)
	for rows.Next() {
		var name, sourceHash, exportsStr, compiledAtStr string
		var binary []byte
		var size int
		if err := rows.Scan(&name, &binary, &sourceHash, &exportsStr, &size, &compiledAtStr); err != nil {
			return nil, err
		}

		var exports []string
		json.Unmarshal([]byte(exportsStr), &exports)

		compiledAt, _ := time.Parse(time.RFC3339, compiledAtStr)

		modules[name] = &WASMModule{
			Name:       name,
			Binary:     binary,
			SourceHash: sourceHash,
			Exports:    exports,
			Size:       size,
			CompiledAt: compiledAt,
		}
	}
	return modules, rows.Err()
}

func (s *SQLiteStore) DeleteModule(name string) error {
	_, err := s.db.Exec("DELETE FROM wasm_modules WHERE name = ?", name)
	return err
}

// ---------------------------------------------------------------------------
// Shards
// ---------------------------------------------------------------------------

func (s *SQLiteStore) SaveShard(name string, desc ShardDescriptor) error {
	handlersJSON, _ := json.Marshal(desc.Handlers)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO wasm_shards (name, mode, state_key, handlers, deployed_at)
		 VALUES (?, ?, ?, ?, ?)`,
		name, desc.Mode, "", string(handlersJSON), desc.DeployedAt.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) LoadShards() (map[string]ShardDescriptor, error) {
	rows, err := s.db.Query("SELECT name, mode, handlers, deployed_at FROM wasm_shards")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shards := make(map[string]ShardDescriptor)
	for rows.Next() {
		var name, mode, handlersStr, deployedAtStr string
		if err := rows.Scan(&name, &mode, &handlersStr, &deployedAtStr); err != nil {
			return nil, err
		}

		var handlers map[string]string
		json.Unmarshal([]byte(handlersStr), &handlers)

		deployedAt, _ := time.Parse(time.RFC3339, deployedAtStr)

		shards[name] = ShardDescriptor{
			Module:     name,
			Mode:       mode,
			Handlers:   handlers,
			DeployedAt: deployedAt,
		}
	}
	return shards, rows.Err()
}

func (s *SQLiteStore) DeleteShard(name string) error {
	_, err := s.db.Exec("DELETE FROM wasm_shards WHERE name = ?", name)
	return err
}

// ---------------------------------------------------------------------------
// Shard State
// ---------------------------------------------------------------------------

func (s *SQLiteStore) SaveState(shardName, key string, state map[string]string) error {
	stateJSON, _ := json.Marshal(state)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO shard_state (shard_name, state_key, state, updated_at)
		 VALUES (?, ?, ?, ?)`,
		shardName, key, string(stateJSON), time.Now().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) LoadState(shardName, key string) (map[string]string, error) {
	var stateStr string
	err := s.db.QueryRow(
		"SELECT state FROM shard_state WHERE shard_name = ? AND state_key = ?",
		shardName, key,
	).Scan(&stateStr)
	if err == sql.ErrNoRows {
		return nil, nil // no state saved
	}
	if err != nil {
		return nil, err
	}

	var state map[string]string
	json.Unmarshal([]byte(stateStr), &state)
	return state, nil
}

func (s *SQLiteStore) DeleteState(shardName string) error {
	_, err := s.db.Exec("DELETE FROM shard_state WHERE shard_name = ?", shardName)
	return err
}

// ---------------------------------------------------------------------------
// Deployments
// ---------------------------------------------------------------------------

func (s *SQLiteStore) SaveDeployment(d PersistedDeployment) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO deployments (source, code, deploy_order, deployed_at)
		 VALUES (?, ?, ?, ?)`,
		d.Source, d.Code, d.Order, d.DeployedAt.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) LoadDeployments() ([]PersistedDeployment, error) {
	rows, err := s.db.Query("SELECT source, code, deploy_order, deployed_at FROM deployments ORDER BY deploy_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []PersistedDeployment
	for rows.Next() {
		var d PersistedDeployment
		var deployedAtStr string
		if err := rows.Scan(&d.Source, &d.Code, &d.Order, &deployedAtStr); err != nil {
			return nil, err
		}
		d.DeployedAt, _ = time.Parse(time.RFC3339, deployedAtStr)
		deployments = append(deployments, d)
	}
	return deployments, rows.Err()
}

func (s *SQLiteStore) DeleteDeployment(source string) error {
	_, err := s.db.Exec("DELETE FROM deployments WHERE source = ?", source)
	return err
}

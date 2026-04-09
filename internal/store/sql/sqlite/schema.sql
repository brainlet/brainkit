-- KitStore tables

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
    owner TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    binary_path TEXT NOT NULL,
    env TEXT NOT NULL DEFAULT '{}',
    config TEXT NOT NULL DEFAULT '{}',
    start_order INTEGER NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'service'
);

CREATE TABLE IF NOT EXISTS schedule_fires (
    schedule_id TEXT NOT NULL,
    fire_time   TEXT NOT NULL,
    claimed_at  TEXT NOT NULL,
    PRIMARY KEY (schedule_id, fire_time)
);

-- AuditStore tables

CREATE TABLE IF NOT EXISTS audit_events (
    id         TEXT PRIMARY KEY,
    timestamp  TEXT NOT NULL,
    category   TEXT NOT NULL,
    event_type TEXT NOT NULL,
    source     TEXT NOT NULL DEFAULT '',
    runtime_id TEXT NOT NULL DEFAULT '',
    namespace  TEXT NOT NULL DEFAULT '',
    data       TEXT NOT NULL DEFAULT '{}',
    duration   INTEGER NOT NULL DEFAULT 0,
    error_msg  TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_category ON audit_events(category);
CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_source ON audit_events(source);

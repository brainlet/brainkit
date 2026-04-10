-- ── Deployments ──

-- name: SaveDeployment :exec
INSERT INTO deployments (source, code, deploy_order, deployed_at, package_name)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (source) DO UPDATE SET
    code = EXCLUDED.code,
    deploy_order = EXCLUDED.deploy_order,
    deployed_at = EXCLUDED.deployed_at,
    package_name = EXCLUDED.package_name;

-- name: LoadDeployments :many
SELECT source, code, deploy_order, deployed_at, package_name
FROM deployments ORDER BY deploy_order;

-- name: LoadDeployment :one
SELECT source, code, deploy_order, deployed_at, package_name
FROM deployments WHERE source = $1;

-- name: DeleteDeployment :exec
DELETE FROM deployments WHERE source = $1;

-- ── Schedules ──

-- name: SaveSchedule :exec
INSERT INTO schedules (id, expression, duration_ns, topic, payload, source, created_at, next_fire, one_time)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
    expression = EXCLUDED.expression,
    duration_ns = EXCLUDED.duration_ns,
    topic = EXCLUDED.topic,
    payload = EXCLUDED.payload,
    source = EXCLUDED.source,
    created_at = EXCLUDED.created_at,
    next_fire = EXCLUDED.next_fire,
    one_time = EXCLUDED.one_time;

-- name: LoadSchedules :many
SELECT id, expression, duration_ns, topic, payload, source, created_at, next_fire, one_time
FROM schedules;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = $1;

-- ── Schedule Fires (deduplication) ──

-- name: ClaimScheduleFire :exec
INSERT INTO schedule_fires (schedule_id, fire_time, claimed_at)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- ── Installed Plugins ──

-- name: SaveInstalledPlugin :exec
INSERT INTO installed_plugins (name, owner, version, binary_path, manifest, installed_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (owner, name) DO UPDATE SET
    version = EXCLUDED.version,
    binary_path = EXCLUDED.binary_path,
    manifest = EXCLUDED.manifest,
    installed_at = EXCLUDED.installed_at;

-- name: LoadInstalledPlugins :many
SELECT name, owner, version, binary_path, manifest, installed_at
FROM installed_plugins;

-- name: DeleteInstalledPlugin :exec
DELETE FROM installed_plugins WHERE name = $1;

-- ── Running Plugins ──

-- name: SaveRunningPlugin :exec
INSERT INTO running_plugins (name, owner, version, binary_path, env, config, start_order, started_at, role)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (name) DO UPDATE SET
    owner = EXCLUDED.owner,
    version = EXCLUDED.version,
    binary_path = EXCLUDED.binary_path,
    env = EXCLUDED.env,
    config = EXCLUDED.config,
    start_order = EXCLUDED.start_order,
    started_at = EXCLUDED.started_at,
    role = EXCLUDED.role;

-- name: LoadRunningPlugins :many
SELECT name, owner, version, binary_path, env, config, start_order, started_at, role
FROM running_plugins ORDER BY start_order;

-- name: DeleteRunningPlugin :exec
DELETE FROM running_plugins WHERE name = $1;

-- ── Audit Events ──

-- name: RecordAuditEvent :exec
INSERT INTO audit_events (id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: QueryAuditEventsByCategory :many
SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE category = $1 ORDER BY created_at DESC LIMIT $2;

-- name: QueryAuditEventsByType :many
SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE event_type = $1 ORDER BY created_at DESC LIMIT $2;

-- name: QueryAuditEventsBySource :many
SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE source = $1 ORDER BY created_at DESC LIMIT $2;

-- name: QueryAuditEventsAll :many
SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events ORDER BY created_at DESC LIMIT $1;

-- name: QueryAuditEventsSince :many
SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE created_at >= $1 ORDER BY created_at DESC LIMIT $2;

-- name: CountAuditEvents :one
SELECT COUNT(*) FROM audit_events;

-- name: CountAuditEventsByCategory :many
SELECT category, COUNT(*) AS cnt FROM audit_events GROUP BY category;

-- name: PruneAuditEvents :exec
DELETE FROM audit_events WHERE created_at < $1;

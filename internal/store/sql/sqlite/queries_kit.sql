-- ── Deployments ──

-- name: SaveDeployment :exec
INSERT OR REPLACE INTO deployments (source, code, deploy_order, deployed_at, package_name)
VALUES (?, ?, ?, ?, ?);

-- name: LoadDeployments :many
SELECT source, code, deploy_order, deployed_at, package_name
FROM deployments ORDER BY deploy_order;

-- name: LoadDeployment :one
SELECT source, code, deploy_order, deployed_at, package_name
FROM deployments WHERE source = ?;

-- name: DeleteDeployment :exec
DELETE FROM deployments WHERE source = ?;

-- ── Schedules ──

-- name: SaveSchedule :exec
INSERT OR REPLACE INTO schedules (id, expression, duration_ns, topic, payload, source, created_at, next_fire, one_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: LoadSchedules :many
SELECT id, expression, duration_ns, topic, payload, source, created_at, next_fire, one_time
FROM schedules;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = ?;

-- ── Schedule Fires (deduplication) ──

-- name: ClaimScheduleFire :exec
INSERT OR IGNORE INTO schedule_fires (schedule_id, fire_time, claimed_at)
VALUES (?, ?, ?);

-- ── Installed Plugins ──

-- name: SaveInstalledPlugin :exec
INSERT OR REPLACE INTO installed_plugins (name, owner, version, binary_path, manifest, installed_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: LoadInstalledPlugins :many
SELECT name, owner, version, binary_path, manifest, installed_at
FROM installed_plugins;

-- name: DeleteInstalledPlugin :exec
DELETE FROM installed_plugins WHERE name = ?;

-- ── Running Plugins ──

-- name: SaveRunningPlugin :exec
INSERT OR REPLACE INTO running_plugins (name, owner, version, binary_path, env, config, start_order, started_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: LoadRunningPlugins :many
SELECT name, owner, version, binary_path, env, config, start_order, started_at
FROM running_plugins ORDER BY start_order;

-- name: DeleteRunningPlugin :exec
DELETE FROM running_plugins WHERE name = ?;

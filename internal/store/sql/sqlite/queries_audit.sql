-- ── Audit Events ──

-- name: RecordAuditEvent :exec
INSERT INTO audit_events (id, timestamp, category, event_type, source, runtime_id, namespace, data, duration, error_msg)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: QueryAuditEventsByCategory :many
SELECT id, timestamp, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE category = ? ORDER BY timestamp DESC LIMIT ?;

-- name: QueryAuditEventsByType :many
SELECT id, timestamp, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE event_type = ? ORDER BY timestamp DESC LIMIT ?;

-- name: QueryAuditEventsBySource :many
SELECT id, timestamp, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE source = ? ORDER BY timestamp DESC LIMIT ?;

-- name: QueryAuditEventsAll :many
SELECT id, timestamp, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events ORDER BY timestamp DESC LIMIT ?;

-- name: QueryAuditEventsSince :many
SELECT id, timestamp, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE timestamp >= ? ORDER BY timestamp DESC LIMIT ?;

-- name: CountAuditEvents :one
SELECT COUNT(*) FROM audit_events;

-- name: CountAuditEventsByCategory :many
SELECT category, COUNT(*) AS cnt FROM audit_events GROUP BY category;

-- name: PruneAuditEvents :exec
DELETE FROM audit_events WHERE timestamp < ?;

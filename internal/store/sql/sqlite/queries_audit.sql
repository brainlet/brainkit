-- ── Audit Events ──

-- name: RecordAuditEvent :exec
INSERT INTO audit_events (id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: QueryAuditEventsByCategory :many
SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE category = ? ORDER BY created_at DESC LIMIT ?;

-- name: QueryAuditEventsByType :many
SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE event_type = ? ORDER BY created_at DESC LIMIT ?;

-- name: QueryAuditEventsBySource :many
SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE source = ? ORDER BY created_at DESC LIMIT ?;

-- name: QueryAuditEventsAll :many
SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events ORDER BY created_at DESC LIMIT ?;

-- name: QueryAuditEventsSince :many
SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg
FROM audit_events WHERE created_at >= ? ORDER BY created_at DESC LIMIT ?;


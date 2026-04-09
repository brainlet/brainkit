-- name: CountAuditEvents :one
SELECT COUNT(*) FROM audit_events;

-- name: CountAuditEventsByCategory :many
SELECT category, COUNT(*) AS cnt FROM audit_events GROUP BY category;

-- name: PruneAuditEvents :exec
DELETE FROM audit_events WHERE created_at < ?;

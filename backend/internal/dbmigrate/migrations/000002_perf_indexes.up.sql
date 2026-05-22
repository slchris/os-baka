-- Performance indexes for high-frequency query paths.
--
-- nodes(status)              : Dashboard summary counts (4 status filters per
--                              refresh) and the nodes list status filter.
-- nodes(last_heartbeat)      : StartStaleNodeChecker scans every 5 minutes
--                              with WHERE last_heartbeat < threshold.
-- audit_logs(created_at DESC): All audit log listing is ORDER BY created_at
--                              DESC LIMIT N. Without this index every page
--                              load is a full table scan + sort.
-- audit_logs(user_id, resource): "What did user X do to resource Y" — a
--                              composite makes the common combined filter
--                              fast without two separate indexes.
--
-- IF NOT EXISTS is intentional: an operator may have hand-added one of these
-- in production already.

BEGIN;

CREATE INDEX IF NOT EXISTS idx_nodes_status
    ON nodes (status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_nodes_last_heartbeat
    ON nodes (last_heartbeat)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at
    ON audit_logs (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id_resource
    ON audit_logs (user_id, resource);

COMMIT;

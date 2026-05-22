BEGIN;

DROP INDEX IF EXISTS idx_audit_logs_user_id_resource;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_nodes_last_heartbeat;
DROP INDEX IF EXISTS idx_nodes_status;

COMMIT;

BEGIN;

DROP INDEX IF EXISTS idx_nodes_installing_started_at;
ALTER TABLE nodes DROP COLUMN IF EXISTS installing_started_at;

COMMIT;

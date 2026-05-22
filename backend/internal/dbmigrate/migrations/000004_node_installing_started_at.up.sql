-- Track when a node entered the provisioning state. Required for the
-- install-timeout watcher: nodes that have been "installing" for longer
-- than the configured threshold get flipped to "error" automatically so
-- they don't sit forever invisible to operators.

BEGIN;

ALTER TABLE nodes ADD COLUMN installing_started_at timestamptz;

-- Backfill: any node currently in installing/pending gets stamped NOW so
-- a fresh deploy doesn't immediately flag every in-flight node as stale.
UPDATE nodes
SET installing_started_at = now()
WHERE deleted_at IS NULL
  AND status IN ('installing', 'pending')
  AND installing_started_at IS NULL;

-- Partial index: only relevant for nodes currently provisioning. Keeps
-- the index tiny since most nodes will be active/offline.
CREATE INDEX IF NOT EXISTS idx_nodes_installing_started_at
    ON nodes (installing_started_at)
    WHERE installing_started_at IS NOT NULL AND deleted_at IS NULL;

COMMIT;

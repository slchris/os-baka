-- Target disk for OS installation. NULL = let preseed early_command
-- auto-detect (the safe default for mixed hardware fleets). Set
-- explicitly (e.g. /dev/nvme0n1) when auto-detect can't disambiguate.
--
-- No partial index — most rows will be NULL, and we never filter on
-- target_disk in the query layer; it's only consumed by the preseed
-- generator at PXE-boot time.

BEGIN;

ALTER TABLE nodes ADD COLUMN target_disk text;

COMMIT;

BEGIN;

ALTER TABLE dhcp_configs DROP COLUMN IF EXISTS security_mirror_url;
ALTER TABLE nodes DROP COLUMN IF EXISTS security_mirror_url;

COMMIT;

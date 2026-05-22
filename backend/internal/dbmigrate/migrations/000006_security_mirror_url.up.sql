-- Add SecurityMirrorURL columns. Debian (and Ubuntu) ship security
-- updates from a separate archive — for Debian that's
-- security.debian.org, not the deb.debian.org main archive. The previous
-- preseed code hardcoded `<main_mirror>-security` as the path which is
-- wrong: it sent installers to e.g. http://deb.debian.org/debian-security
-- which 404s.
--
-- Both columns are NULLable — empty/unset means "fall back to env or the
-- built-in default for this OS family". Operators on an intranet who
-- mirror security into a single host typically set DHCPConfig's value
-- (applies fleet-wide) and rarely need the per-node override.

BEGIN;

ALTER TABLE nodes ADD COLUMN security_mirror_url text;
ALTER TABLE dhcp_configs ADD COLUMN security_mirror_url text;

COMMIT;

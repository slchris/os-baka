-- Rename the GORM-mangled IPMI columns on the nodes table to their
-- intended names. GORM's snake_case converter split the "IPMI" prefix
-- mid-acronym (e.g. IPMIPassword → ip_m_ipassword), which leaks an
-- internal quirk into the schema. After this migration the model uses
-- explicit `gorm:"column:..."` tags so GORM stops doing the conversion.
--
-- Safe to run on a freshly-baselined DB or one that already had data —
-- RENAME COLUMN preserves values, types, defaults, and any indexes on
-- the column. There are no indexes on these four columns today.

BEGIN;

ALTER TABLE nodes RENAME COLUMN ip_mi_address         TO ipmi_address;
ALTER TABLE nodes RENAME COLUMN ip_mi_username        TO ipmi_username;
ALTER TABLE nodes RENAME COLUMN ip_m_ipassword        TO ipmi_password;
ALTER TABLE nodes RENAME COLUMN ip_mi_allow_untrusted TO ipmi_allow_untrusted;

COMMIT;

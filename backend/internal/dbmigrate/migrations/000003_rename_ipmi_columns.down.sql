BEGIN;

ALTER TABLE nodes RENAME COLUMN ipmi_address         TO ip_mi_address;
ALTER TABLE nodes RENAME COLUMN ipmi_username        TO ip_mi_username;
ALTER TABLE nodes RENAME COLUMN ipmi_password        TO ip_m_ipassword;
ALTER TABLE nodes RENAME COLUMN ipmi_allow_untrusted TO ip_mi_allow_untrusted;

COMMIT;

BEGIN;

DROP INDEX IF EXISTS users_mattermost_id_unique;
ALTER TABLE users DROP COLUMN IF EXISTS mattermost_id;

-- NOTE: PostgreSQL does not support removing enum values.
-- The 'mattermost' value added to audit_record_source_type cannot be reverted.

COMMIT;

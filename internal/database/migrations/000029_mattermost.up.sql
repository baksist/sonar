BEGIN;

ALTER TABLE users ADD COLUMN mattermost_id TEXT;
CREATE UNIQUE INDEX users_mattermost_id_unique ON users (mattermost_id) WHERE mattermost_id IS NOT NULL;

ALTER TYPE audit_record_source_type ADD VALUE 'mattermost';

COMMIT;

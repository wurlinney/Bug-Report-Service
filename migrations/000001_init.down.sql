BEGIN;

DROP TRIGGER IF EXISTS set_bug_reports_updated_at ON bug_reports;
DROP TRIGGER IF EXISTS set_moderators_updated_at ON moderators;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS internal_notes;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS bug_reports;
DROP TABLE IF EXISTS moderators;

DROP FUNCTION IF EXISTS set_updated_at();

COMMIT;

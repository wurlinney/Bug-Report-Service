BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS moderators (
  id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
  name TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bug_reports (
  id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
  reporter_name TEXT NOT NULL,
  description TEXT NULL,
  status TEXT NOT NULL DEFAULT 'new',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT bug_reports_status_check CHECK (status IN ('new', 'in_review', 'resolved', 'rejected'))
);

CREATE TABLE IF NOT EXISTS attachments (
  id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
  bug_report_id TEXT NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
  file_name TEXT NOT NULL,
  content_type TEXT NOT NULL,
  file_size BIGINT NOT NULL,
  storage_key TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  idempotency_key TEXT NULL
);

CREATE TABLE IF NOT EXISTS internal_notes (
  id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
  bug_report_id TEXT NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
  author_moderator_id TEXT NOT NULL REFERENCES moderators(id) ON DELETE RESTRICT,
  text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
  moderator_id TEXT NOT NULL REFERENCES moderators(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at TIMESTAMPTZ NULL,
  replaced_by TEXT NULL
);

CREATE INDEX IF NOT EXISTS idx_bug_reports_created
  ON bug_reports(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bug_reports_status_updated
  ON bug_reports(status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_bug_reports_reporter_name
  ON bug_reports(reporter_name);
CREATE INDEX IF NOT EXISTS idx_internal_notes_report_created
  ON internal_notes(bug_report_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_moderator_created
  ON refresh_tokens(moderator_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS ux_attachments_report_idem
  ON attachments(bug_report_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;

DROP TRIGGER IF EXISTS set_moderators_updated_at ON moderators;
CREATE TRIGGER set_moderators_updated_at
BEFORE UPDATE ON moderators
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_bug_reports_updated_at ON bug_reports;
CREATE TRIGGER set_bug_reports_updated_at
BEFORE UPDATE ON bug_reports
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;

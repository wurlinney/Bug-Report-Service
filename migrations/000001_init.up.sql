BEGIN;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS moderators (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bug_reports (
  id BIGSERIAL PRIMARY KEY,
  reporter_name TEXT NOT NULL,
  description TEXT NULL,
  status TEXT NOT NULL DEFAULT 'new',
  influence TEXT NOT NULL DEFAULT 'Не задано',
  priority TEXT NOT NULL DEFAULT 'Не задан',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT bug_reports_status_check CHECK (status IN ('new', 'in_review', 'resolved', 'rejected')),
  CONSTRAINT bug_reports_influence_check CHECK (influence IN ('Крит/блокер', 'Крит/Блокер', 'Высокий', 'Средний', 'Низкий', 'Не баг а фича', 'Не задано')),
  CONSTRAINT bug_reports_priority_check CHECK (priority IN ('Высокий', 'Средний', 'Низкий', 'Не задан'))
);

CREATE TABLE IF NOT EXISTS upload_sessions (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS attachments (
  id BIGSERIAL PRIMARY KEY,
  bug_report_id BIGINT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
  upload_session_id BIGINT NULL REFERENCES upload_sessions(id) ON DELETE CASCADE,
  file_name TEXT NOT NULL,
  content_type TEXT NOT NULL,
  file_size BIGINT NOT NULL,
  storage_key TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  idempotency_key TEXT NULL,
  CONSTRAINT attachments_owner_check CHECK (
    (bug_report_id IS NOT NULL AND upload_session_id IS NULL)
    OR (bug_report_id IS NULL AND upload_session_id IS NOT NULL)
  )
);

CREATE TABLE IF NOT EXISTS internal_notes (
  id BIGSERIAL PRIMARY KEY,
  bug_report_id BIGINT NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
  author_moderator_id BIGINT NOT NULL REFERENCES moderators(id) ON DELETE RESTRICT,
  text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id BIGSERIAL PRIMARY KEY,
  moderator_id BIGINT NOT NULL REFERENCES moderators(id) ON DELETE CASCADE,
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
CREATE INDEX IF NOT EXISTS idx_attachments_upload_session_created
  ON attachments(upload_session_id, created_at ASC);
CREATE UNIQUE INDEX IF NOT EXISTS ux_attachments_report_idem
  ON attachments(bug_report_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_attachments_session_idem
  ON attachments(upload_session_id, idempotency_key)
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

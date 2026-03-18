BEGIN;

-- 1) Moderators (replace users)
CREATE TABLE IF NOT EXISTS moderators (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

-- Best-effort migration of existing staff accounts (if any legacy users table exists).
DO $$
BEGIN
  IF to_regclass('public.users') IS NOT NULL THEN
    EXECUTE $SQL$
      INSERT INTO moderators (id, email, password_hash, role, created_at, updated_at)
      SELECT id, email, password_hash, role, created_at, updated_at
      FROM users
      WHERE role IN ('moderator', 'admin')
      ON CONFLICT (id) DO NOTHING
    $SQL$;
  END IF;
END $$;

-- 2) Bug reports: remove ownership, add reporter_name, remove title, make description nullable
ALTER TABLE bug_reports
  ADD COLUMN IF NOT EXISTS reporter_name TEXT NOT NULL DEFAULT 'unknown';

ALTER TABLE bug_reports
  ALTER COLUMN description DROP NOT NULL;

-- Drop FKs/columns only if they exist (handles repeated runs in dev)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='bug_reports' AND column_name='user_id'
  ) THEN
    EXECUTE 'ALTER TABLE bug_reports DROP CONSTRAINT IF EXISTS bug_reports_user_id_fkey';
    EXECUTE 'ALTER TABLE bug_reports DROP COLUMN user_id';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='bug_reports' AND column_name='title'
  ) THEN
    EXECUTE 'ALTER TABLE bug_reports DROP COLUMN title';
  END IF;
END $$;

-- Replace index targeting user_id
DROP INDEX IF EXISTS idx_bug_reports_user_created;
CREATE INDEX IF NOT EXISTS idx_bug_reports_created ON bug_reports(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bug_reports_reporter_name ON bug_reports(reporter_name);
-- keep existing idx_bug_reports_status_updated if present

-- 3) Attachments: remove uploader fields, keep idempotency key uniqueness
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='attachments' AND column_name='uploaded_by_id'
  ) THEN
    EXECUTE 'ALTER TABLE attachments DROP COLUMN uploaded_by_id';
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='attachments' AND column_name='uploaded_by_role'
  ) THEN
    EXECUTE 'ALTER TABLE attachments DROP COLUMN uploaded_by_role';
  END IF;
END $$;

-- 4) Internal notes (replace messages/chat)
CREATE TABLE IF NOT EXISTS internal_notes (
  id TEXT PRIMARY KEY,
  bug_report_id TEXT NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
  author_moderator_id TEXT NOT NULL REFERENCES moderators(id) ON DELETE RESTRICT,
  text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_internal_notes_report_created
  ON internal_notes(bug_report_id, created_at ASC);

-- 5) Refresh tokens: point to moderators
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='refresh_tokens' AND column_name='user_id'
  ) THEN
    EXECUTE 'ALTER TABLE refresh_tokens RENAME COLUMN user_id TO moderator_id';
  END IF;
  EXECUTE 'ALTER TABLE refresh_tokens DROP CONSTRAINT IF EXISTS refresh_tokens_user_id_fkey';
END $$;

-- Legacy refresh tokens for deleted/non-moderator users are no longer valid.
DELETE FROM refresh_tokens
WHERE moderator_id NOT IN (SELECT id FROM moderators);

ALTER TABLE refresh_tokens
  ADD CONSTRAINT refresh_tokens_moderator_id_fkey
    FOREIGN KEY (moderator_id) REFERENCES moderators(id) ON DELETE CASCADE;

DROP INDEX IF EXISTS idx_refresh_tokens_user_created;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_moderator_created
  ON refresh_tokens(moderator_id, created_at DESC);

-- 6) Remove user accounts and chat table
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS users;

COMMIT;


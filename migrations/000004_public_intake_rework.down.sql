BEGIN;

-- Recreate users table (legacy) and move moderators back.
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  name TEXT NOT NULL DEFAULT ''
);

INSERT INTO users (id, email, password_hash, role, created_at, updated_at, name)
SELECT id, email, password_hash, role, created_at, updated_at, ''
FROM moderators
ON CONFLICT (id) DO NOTHING;

-- Restore refresh_tokens to reference users(id)
DO $$
BEGIN
  EXECUTE 'ALTER TABLE refresh_tokens DROP CONSTRAINT IF EXISTS refresh_tokens_moderator_id_fkey';
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='refresh_tokens' AND column_name='moderator_id'
  ) THEN
    EXECUTE 'ALTER TABLE refresh_tokens RENAME COLUMN moderator_id TO user_id';
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='refresh_tokens' AND column_name='user_id'
  ) THEN
    EXECUTE 'ALTER TABLE refresh_tokens
      ADD CONSTRAINT refresh_tokens_user_id_fkey
      FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE';
  END IF;
END $$;

DROP INDEX IF EXISTS idx_refresh_tokens_moderator_created;
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='refresh_tokens' AND column_name='user_id'
  ) THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_created ON refresh_tokens(user_id, created_at DESC)';
  END IF;
END $$;

-- Restore bug_reports legacy columns (best-effort; data may be lost)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='bug_reports' AND column_name='user_id'
  ) THEN
    EXECUTE 'ALTER TABLE bug_reports ADD COLUMN user_id TEXT NULL';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='bug_reports' AND column_name='title'
  ) THEN
    EXECUTE $SQL$ALTER TABLE bug_reports ADD COLUMN title TEXT NOT NULL DEFAULT ''$SQL$;
  END IF;
END $$;

ALTER TABLE bug_reports
  ALTER COLUMN description SET NOT NULL;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='bug_reports' AND column_name='user_id'
  ) THEN
    EXECUTE 'ALTER TABLE bug_reports
      ADD CONSTRAINT bug_reports_user_id_fkey
      FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE';
  END IF;
END $$;

-- Remove reporter_name
ALTER TABLE bug_reports
  DROP COLUMN IF EXISTS reporter_name;

DROP INDEX IF EXISTS idx_bug_reports_created;
DROP INDEX IF EXISTS idx_bug_reports_reporter_name;
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='bug_reports' AND column_name='user_id'
  ) THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_bug_reports_user_created ON bug_reports(user_id, created_at DESC)';
  END IF;
END $$;

-- Restore attachments legacy columns
ALTER TABLE attachments
  ADD COLUMN IF NOT EXISTS uploaded_by_id TEXT NOT NULL DEFAULT '';
ALTER TABLE attachments
  ADD COLUMN IF NOT EXISTS uploaded_by_role TEXT NOT NULL DEFAULT '';

-- Restore chat/messages table (empty)
CREATE TABLE IF NOT EXISTS messages (
  id TEXT PRIMARY KEY,
  bug_report_id TEXT NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
  sender_id TEXT NOT NULL,
  sender_role TEXT NOT NULL,
  text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_report_created ON messages(bug_report_id, created_at ASC);

-- Drop internal notes and moderators
DROP TABLE IF EXISTS internal_notes;
DROP TABLE IF EXISTS moderators;

COMMIT;


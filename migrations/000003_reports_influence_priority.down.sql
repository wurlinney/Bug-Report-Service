BEGIN;

-- Drop constraints if exist.
ALTER TABLE bug_reports
  DROP CONSTRAINT IF EXISTS bug_reports_influence_check;

ALTER TABLE bug_reports
  DROP CONSTRAINT IF EXISTS bug_reports_priority_check;

ALTER TABLE bug_reports
  DROP COLUMN IF EXISTS influence;

ALTER TABLE bug_reports
  DROP COLUMN IF EXISTS priority;

COMMIT;


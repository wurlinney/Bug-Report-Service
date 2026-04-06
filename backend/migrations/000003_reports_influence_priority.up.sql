BEGIN;

ALTER TABLE bug_reports
  ADD COLUMN IF NOT EXISTS influence TEXT NOT NULL DEFAULT 'Не задано';

ALTER TABLE bug_reports
  ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'Не задан';

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'bug_reports_influence_check') THEN
    ALTER TABLE bug_reports
      ADD CONSTRAINT bug_reports_influence_check
      CHECK (influence IN ('Крит/блокер', 'Крит/Блокер', 'Высокий', 'Средний', 'Низкий', 'Не баг а фича', 'Не задано'));
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'bug_reports_priority_check') THEN
    ALTER TABLE bug_reports
      ADD CONSTRAINT bug_reports_priority_check
      CHECK (priority IN ('Высокий', 'Средний', 'Низкий', 'Не задан'));
  END IF;
END$$;

COMMIT;


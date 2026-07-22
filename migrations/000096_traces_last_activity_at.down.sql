DROP INDEX IF EXISTS idx_traces_running_activity;
ALTER TABLE traces DROP COLUMN IF EXISTS last_activity_at;

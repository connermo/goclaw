-- Heartbeat column for stale-trace recovery.
--
-- A trace is orphaned when the process that owned it dies mid-run (crash,
-- container restart, SIGKILL): nothing ever writes its terminal status, so it
-- stays 'running' forever. Recovery cannot key off start_time — a legitimate
-- run that takes an hour looks identical to an orphan that started an hour ago.
-- Nor off the newest span: a single long LLM call emits no spans while it runs.
--
-- last_activity_at is written by the owning process on every flush cycle, so it
-- advances while that process lives and freezes the moment it dies. Recovery
-- gates on it going stale, which distinguishes the two cases.
--
-- NULL means "never heartbeated" (rows predating this migration, or a trace
-- created in the window before its first flush); readers fall back to
-- start_time, which keeps the old behavior for those rows.
ALTER TABLE traces ADD COLUMN IF NOT EXISTS last_activity_at TIMESTAMPTZ;

-- Partial index: the recovery sweep only ever scans running traces, which are a
-- tiny fraction of the table.
CREATE INDEX IF NOT EXISTS idx_traces_running_activity
  ON traces (last_activity_at)
  WHERE status = 'running';

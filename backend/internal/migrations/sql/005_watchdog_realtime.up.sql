-- Add real-time tracking columns to watchdog.
-- WatchedSeconds accumulates actual playback time (excluding paused periods).
-- LastTickAt records when the session was last processed by SessionTick.
ALTER TABLE jf_activity_watchdog
  ADD COLUMN IF NOT EXISTS "WatchedSeconds" bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "LastTickAt"     timestamptz;

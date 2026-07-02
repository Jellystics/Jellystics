ALTER TABLE jf_activity_watchdog
  DROP COLUMN IF EXISTS "WatchedSeconds",
  DROP COLUMN IF EXISTS "LastTickAt";

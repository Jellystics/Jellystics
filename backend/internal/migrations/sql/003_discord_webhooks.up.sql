ALTER TABLE webhooks
  ADD COLUMN IF NOT EXISTS bot_username   text NOT NULL DEFAULT 'jellystics_bot',
  ADD COLUMN IF NOT EXISTS bot_avatar_url text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS discord_events jsonb NOT NULL DEFAULT '[]';

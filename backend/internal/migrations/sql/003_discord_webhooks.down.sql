ALTER TABLE webhooks
  DROP COLUMN IF EXISTS bot_username,
  DROP COLUMN IF EXISTS bot_avatar_url,
  DROP COLUMN IF EXISTS discord_events;

package webhook

import (
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"gorm.io/datatypes"
)

func sp(s string) *string { return &s }

// TestIsSubscribed_DiscordEvents verifies the modern discord_events jsonb array
// is the source of truth: a matching event subscribes, a non-matching one does
// not.
func TestIsSubscribed_DiscordEvents(t *testing.T) {
	hook := &models.Webhook{
		DiscordEvents: datatypes.JSON([]byte(`["task_start","backup_complete"]`)),
	}
	if !isSubscribed(hook, "task_start") {
		t.Error("expected subscription to task_start via discord_events")
	}
	if !isSubscribed(hook, "backup_complete") {
		t.Error("expected subscription to backup_complete via discord_events")
	}
	if isSubscribed(hook, "task_failed") {
		t.Error("task_failed not in discord_events, should not subscribe")
	}
}

// TestIsSubscribed_DiscordEventsOverridesLegacy verifies that when discord_events
// is present, the legacy event_type is ignored entirely.
func TestIsSubscribed_DiscordEventsOverridesLegacy(t *testing.T) {
	hook := &models.Webhook{
		DiscordEvents: datatypes.JSON([]byte(`["task_start"]`)),
		TriggerType:   "event",
		EventType:     sp("task_failed"),
	}
	if isSubscribed(hook, "task_failed") {
		t.Error("legacy event_type must be ignored when discord_events is set")
	}
	if !isSubscribed(hook, "task_start") {
		t.Error("should subscribe via discord_events")
	}
}

// TestIsSubscribed_LegacyFallback verifies the legacy path: with no (or empty)
// discord_events, TriggerType=="event" and a matching EventType subscribe.
func TestIsSubscribed_LegacyFallback(t *testing.T) {
	hook := &models.Webhook{
		DiscordEvents: datatypes.JSON([]byte(`[]`)),
		TriggerType:   "event",
		EventType:     sp("task_complete"),
	}
	if !isSubscribed(hook, "task_complete") {
		t.Error("legacy fallback should subscribe to matching event_type")
	}
	if isSubscribed(hook, "task_start") {
		t.Error("legacy fallback should not subscribe to a different event")
	}
}

// TestIsSubscribed_LegacyWrongTrigger verifies a non-"event" trigger type never
// subscribes through the legacy path.
func TestIsSubscribed_LegacyWrongTrigger(t *testing.T) {
	hook := &models.Webhook{
		DiscordEvents: datatypes.JSON([]byte(`[]`)),
		TriggerType:   "schedule",
		EventType:     sp("task_complete"),
	}
	if isSubscribed(hook, "task_complete") {
		t.Error("schedule trigger must not subscribe via the event path")
	}
}

// TestBuildEmbed_KnownEvent verifies a known event produces the branded embed
// with title, description, color, and the expected fields.
func TestBuildEmbed_KnownEvent(t *testing.T) {
	embed := buildEmbed("task_complete", map[string]any{
		"name":     "Full Sync",
		"duration": "42s",
	}, "http://app/logo.png")

	if embed["title"] != "Task Completed" {
		t.Errorf("title = %v, want Task Completed", embed["title"])
	}
	if embed["color"] != 5763719 {
		t.Errorf("color = %v, want 5763719", embed["color"])
	}
	fields, ok := embed["fields"].([]any)
	if !ok || len(fields) != 2 {
		t.Fatalf("fields = %v, want 2 (Task, Duration)", embed["fields"])
	}
	f0 := fields[0].(map[string]any)
	if f0["name"] != "Task" || f0["value"] != "Full Sync" {
		t.Errorf("field[0] = %v, want {Task Full Sync}", f0)
	}
	// icon_url should be set on author when avatarURL is provided.
	author := embed["author"].(map[string]any)
	if author["icon_url"] != "http://app/logo.png" {
		t.Errorf("author.icon_url = %v, want the avatar URL", author["icon_url"])
	}
}

// TestBuildEmbed_NoFieldsWhenDataMissing verifies known events omit the fields
// key entirely when data has no usable values.
func TestBuildEmbed_NoFieldsWhenDataMissing(t *testing.T) {
	embed := buildEmbed("task_start", map[string]any{}, "")
	if _, ok := embed["fields"]; ok {
		t.Errorf("fields should be absent when no data provided, got %v", embed["fields"])
	}
	// No avatar → author has no icon_url.
	author := embed["author"].(map[string]any)
	if _, ok := author["icon_url"]; ok {
		t.Error("author.icon_url should be absent without an avatar URL")
	}
}

// TestBuildEmbed_UnknownEvent verifies an unknown event falls back to a generic
// embed titled with the raw event type and the fallback color.
func TestBuildEmbed_UnknownEvent(t *testing.T) {
	embed := buildEmbed("something_custom", nil, "")
	if embed["title"] != "something_custom" {
		t.Errorf("title = %v, want raw event type", embed["title"])
	}
	if embed["color"] != 7506394 {
		t.Errorf("color = %v, want fallback 7506394", embed["color"])
	}
	if _, ok := embed["fields"]; ok {
		t.Error("unknown event should have no fields")
	}
}

// TestBuildEmbed_BackupFields verifies backup_complete maps file (non-inline)
// and size (inline).
func TestBuildEmbed_BackupFields(t *testing.T) {
	embed := buildEmbed("backup_complete", map[string]any{
		"file": "backup-2026.tar.gz",
		"size": "12MB",
	}, "")
	fields := embed["fields"].([]any)
	if len(fields) != 2 {
		t.Fatalf("fields = %d, want 2", len(fields))
	}
	file := fields[0].(map[string]any)
	if file["name"] != "File" || file["inline"] != false {
		t.Errorf("file field = %v, want File non-inline", file)
	}
	size := fields[1].(map[string]any)
	if size["name"] != "Size" || size["inline"] != true {
		t.Errorf("size field = %v, want Size inline", size)
	}
}

// TestEmbedField verifies the field builder shape.
func TestEmbedField(t *testing.T) {
	f := embedField("Key Name", "prod", true)
	if f["name"] != "Key Name" || f["value"] != "prod" || f["inline"] != true {
		t.Errorf("embedField = %v", f)
	}
}

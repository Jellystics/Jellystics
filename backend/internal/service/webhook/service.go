package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
)

type Service struct {
	repos *repository.Container
}

func New(repos *repository.Container) *Service {
	return &Service{repos: repos}
}

func (s *Service) List(ctx context.Context) ([]models.Webhook, error) {
	return s.repos.Webhook.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id int) (*models.Webhook, error) {
	return s.repos.Webhook.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, wh *models.Webhook) error {
	if wh.BotUsername == "" {
		wh.BotUsername = "jellystics_bot"
	}
	if wh.Method == "" {
		wh.Method = "POST"
	}
	return s.repos.Webhook.Create(ctx, wh)
}

func (s *Service) Update(ctx context.Context, wh *models.Webhook) error {
	if wh.BotUsername == "" {
		wh.BotUsername = "jellystics_bot"
	}
	return s.repos.Webhook.Update(ctx, wh)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	return s.repos.Webhook.Delete(ctx, id)
}

// ─── Event definitions ────────────────────────────────────────────────────────

type eventMeta struct {
	Title string
	Emoji string
	Color int
}

// AllEventTypes is the authoritative list of supported Discord notification events.
var AllEventTypes = []string{
	"task_start",
	"task_complete",
	"task_failed",
	"backup_complete",
	"api_key_created",
	"api_key_deleted",
}

var eventDefs = map[string]eventMeta{
	"task_start":      {"Task Started", "▶️", 3447003},
	"task_complete":   {"Task Completed", "✅", 5763719},
	"task_failed":     {"Task Failed", "❌", 15158332},
	"backup_complete": {"Backup Completed", "💾", 1752220},
	"api_key_created": {"API Key Created", "🔑", 16705372},
	"api_key_deleted": {"API Key Deleted", "🗑️", 15105570},
}

// ─── Fire ─────────────────────────────────────────────────────────────────────

// Fire sends Discord embeds to all enabled webhooks subscribed to eventType.
func (s *Service) Fire(ctx context.Context, eventType string, data map[string]any) []error {
	hooks, err := s.repos.Webhook.List(ctx)
	if err != nil {
		return []error{err}
	}

	var errs []error
	for i := range hooks {
		hook := &hooks[i]
		if !hook.Enabled {
			continue
		}
		if !isSubscribed(hook, eventType) {
			continue
		}
		payload := s.buildDiscordPayload(ctx, hook, eventType, data)
		if err := fireHTTP(ctx, hook.Url, payload); err != nil {
			errs = append(errs, fmt.Errorf("webhook %d: %w", hook.Id, err))
		} else {
			now := time.Now()
			hook.LastTriggered = &now
			_ = s.repos.Webhook.Update(ctx, hook)
		}
	}
	return errs
}

// isSubscribed checks if the webhook is subscribed to the given event type.
// Prefers discord_events jsonb array; falls back to legacy event_type.
func isSubscribed(hook *models.Webhook, eventType string) bool {
	if len(hook.DiscordEvents) > 2 {
		var events []string
		if err := json.Unmarshal(hook.DiscordEvents, &events); err == nil {
			for _, e := range events {
				if e == eventType {
					return true
				}
			}
			return false
		}
	}
	return hook.TriggerType == "event" && hook.EventType != nil && *hook.EventType == eventType
}

// ─── Discord payload builder ──────────────────────────────────────────────────

func (s *Service) resolveAvatarUrl(ctx context.Context, hook *models.Webhook) string {
	if hook.BotAvatarUrl != "" {
		return hook.BotAvatarUrl
	}
	cfg, err := s.repos.Config.Get(ctx)
	if err != nil || cfg.AppUrl == "" {
		return ""
	}
	return strings.TrimRight(cfg.AppUrl, "/") + "/logo.png"
}

func (s *Service) buildDiscordPayload(ctx context.Context, hook *models.Webhook, eventType string, data map[string]any) map[string]any {
	username := hook.BotUsername
	if username == "" {
		username = "jellystics_bot"
	}
	avatarUrl := s.resolveAvatarUrl(ctx, hook)
	payload := map[string]any{
		"username": username,
		"embeds":   []any{buildEmbed(eventType, data, avatarUrl)},
	}
	if avatarUrl != "" {
		payload["avatar_url"] = avatarUrl
	}
	return payload
}

// BuildTestPayload builds a Discord test embed for the given webhook.
// Always uses task_complete so the test shows a representative embed with all fields.
func (s *Service) BuildTestPayload(ctx context.Context, hook *models.Webhook) map[string]any {
	return s.buildDiscordPayload(ctx, hook, "task_complete", map[string]any{
		"name":     "Test Task",
		"duration": "123ms",
	})
}

// eventDescriptions provides a human-readable description for each event type.
var eventDescriptions = map[string]string{
	"task_start":      "A scheduled task has started running.",
	"task_complete":   "A task finished successfully.",
	"task_failed":     "A task encountered an error and failed.",
	"backup_complete": "A backup has been created successfully.",
	"api_key_created": "A new API key has been generated.",
	"api_key_deleted": "An API key has been revoked.",
}

func buildEmbed(eventType string, data map[string]any, avatarURL string) map[string]any {
	meta, ok := eventDefs[eventType]
	if !ok {
		return map[string]any{
			"title":     eventType,
			"color":     7506394,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"footer":    map[string]any{"text": "Jellystics"},
		}
	}

	// Author block — Jellystics branding; icon_url only if a public URL is available.
	author := map[string]any{"name": "Jellystics"}
	if avatarURL != "" {
		author["icon_url"] = avatarURL
	}

	// Footer — reuse avatarURL as icon_url (requires text to be present per Discord spec).
	footer := map[string]any{"text": "Jellystics"}
	if avatarURL != "" {
		footer["icon_url"] = avatarURL
	}

	embed := map[string]any{
		"author":      author,
		"title":       meta.Title,
		"description": eventDescriptions[eventType],
		"color":       meta.Color,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"footer":      footer,
	}

	var fields []any
	switch eventType {
	case "task_start", "task_complete", "task_failed":
		if name, _ := data["name"].(string); name != "" {
			fields = append(fields, embedField("Task", name, true))
		}
		if dur, _ := data["duration"].(string); dur != "" {
			fields = append(fields, embedField("Duration", dur, true))
		}
	case "backup_complete":
		if file, _ := data["file"].(string); file != "" {
			fields = append(fields, embedField("File", file, false))
		}
		if size, _ := data["size"].(string); size != "" {
			fields = append(fields, embedField("Size", size, true))
		}
	case "api_key_created", "api_key_deleted":
		if name, _ := data["name"].(string); name != "" {
			fields = append(fields, embedField("Key Name", name, true))
		}
	}

	if len(fields) > 0 {
		embed["fields"] = fields
	}
	return embed
}

func embedField(name, value string, inline bool) map[string]any {
	return map[string]any{"name": name, "value": value, "inline": inline}
}

// ─── HTTP ─────────────────────────────────────────────────────────────────────

func fireHTTP(ctx context.Context, url string, payload any) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d from Discord", resp.StatusCode)
	}
	return nil
}

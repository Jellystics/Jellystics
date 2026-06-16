package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/service"
	"github.com/gin-gonic/gin"
)

type WebhookHandler struct{ svcs *service.Container }

func NewWebhookHandler(svcs *service.Container) *WebhookHandler { return &WebhookHandler{svcs} }

// GET /webhooks/
func (h *WebhookHandler) List(c *gin.Context) {
	hooks, err := h.svcs.Webhook.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, hooks)
}

// GET /webhooks/:id
func (h *WebhookHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	wh, err := h.svcs.Webhook.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, wh)
}

// POST /webhooks/
func (h *WebhookHandler) Create(c *gin.Context) {
	var wh models.Webhook
	if err := c.ShouldBindJSON(&wh); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svcs.Webhook.Create(c.Request.Context(), &wh); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, wh)
}

// PUT /webhooks/:id
func (h *WebhookHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var wh models.Webhook
	if err := c.ShouldBindJSON(&wh); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wh.Id = id
	if err := h.svcs.Webhook.Update(c.Request.Context(), &wh); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, wh)
}

// DELETE /webhooks/:id
func (h *WebhookHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.svcs.Webhook.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// POST /webhooks/:id/test
// Fires the webhook with a test payload (matches old Node.js behaviour).
func (h *WebhookHandler) Test(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	wh, err := h.svcs.Webhook.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}

	var testPayload map[string]any
	_ = c.ShouldBindJSON(&testPayload)
	if testPayload == nil {
		testPayload = map[string]any{}
	}

	// Build trigger-specific test data (mirrors old JS webhook test logic).
	var payload any
	isDiscord := isDiscordURL(wh.Url)

	if isDiscord {
		payload = map[string]any{
			"content": "Test webhook from Jellystics",
			"embeds": []any{map[string]any{
				"title":       "Discord test notification",
				"description": "This is a test notification of the Jellystics Discord webhook",
				"color":       3447003,
				"fields": []any{
					map[string]any{"name": "Webhook type", "value": wh.TriggerType, "inline": true},
					map[string]any{"name": "ID", "value": strconv.Itoa(wh.Id), "inline": true},
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}},
		}
	} else if wh.TriggerType == "event" && wh.EventType != nil {
		switch *wh.EventType {
		case "playback_started":
			payload = map[string]any{
				"event":       "playback_started",
				"triggeredAt": time.Now().UTC().Format(time.RFC3339),
				"sessionInfo": map[string]any{
					"userId": "test-user-id", "deviceName": "Test Device",
					"clientName": "Test Client", "mediaType": "Movie", "mediaName": "Test Movie",
					"startTime": time.Now().UTC().Format(time.RFC3339),
				},
			}
		case "playback_ended":
			payload = map[string]any{
				"event":       "playback_ended",
				"triggeredAt": time.Now().UTC().Format(time.RFC3339),
				"sessionInfo": map[string]any{
					"userId": "test-user-id", "deviceName": "Test Device",
					"mediaName": "Test Movie", "playbackDuration": 3600,
				},
			}
		case "media_recently_added":
			payload = map[string]any{
				"event":       "media_recently_added",
				"triggeredAt": time.Now().UTC().Format(time.RFC3339),
				"mediaItem": map[string]any{
					"id": "test-item-id", "name": "Test Media", "type": "Movie",
					"addedDate": time.Now().UTC().Format(time.RFC3339),
				},
			}
		default:
			payload = map[string]any{"event": *wh.EventType, "triggeredAt": time.Now().UTC().Format(time.RFC3339)}
		}
	} else {
		payload = map[string]any{
			"event":       "test",
			"triggeredAt": time.Now().UTC().Format(time.RFC3339),
		}
	}

	if err := fireWebhookHTTP(c.Request.Context(), wh, payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while executing webhook: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Webhook executed successfully"})
}

// POST /webhooks/:id/trigger-monthly
// Sends the monthly summary Discord embed for this webhook (Discord-specific).
func (h *WebhookHandler) TriggerMonthly(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	wh, err := h.svcs.Webhook.GetByID(c.Request.Context(), id)
	if err != nil || !wh.Enabled {
		c.JSON(http.StatusNotFound, gin.H{"message": "Webhook not found or disabled"})
		return
	}

	// Build a simple monthly summary payload (Discord embed format).
	now := time.Now()
	prevMonth := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	monthName := prevMonth.Format("January 2006")

	payload := map[string]any{
		"content": "📊 **Monthly Report - " + monthName + "**",
		"embeds": []any{map[string]any{
			"title":       "📈 General Statistics",
			"color":       5763719,
			"description": "Monthly report generated by Jellystics",
			"footer": map[string]any{
				"text": "Period: " + prevMonth.Format("01/02/2006") + " to " +
					time.Date(prevMonth.Year(), prevMonth.Month()+1, 0, 0, 0, 0, 0, time.UTC).Format("01/02/2006"),
			},
		}},
	}

	if err := fireWebhookHTTP(c.Request.Context(), wh, payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send monthly report"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Monthly report sent successfully"})
}

// GET /webhooks/event-status
// Returns whether event webhooks exist and are enabled for each event type.
func (h *WebhookHandler) EventStatus(c *gin.Context) {
	ctx := c.Request.Context()
	hooks, err := h.svcs.Webhook.List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	eventTypes := []string{"playback_started", "playback_ended", "media_recently_added"}
	result := map[string]any{}

	for _, et := range eventTypes {
		var matching []map[string]any
		anyEnabled := false
		for _, hook := range hooks {
			if hook.TriggerType == "event" && hook.EventType != nil && *hook.EventType == et {
				matching = append(matching, map[string]any{
					"id":      hook.Id,
					"name":    hook.Name,
					"enabled": hook.Enabled,
				})
				if hook.Enabled {
					anyEnabled = true
				}
			}
		}
		if matching == nil {
			matching = []map[string]any{}
		}
		result[et] = map[string]any{
			"exists":   len(matching) > 0,
			"enabled":  anyEnabled,
			"webhooks": matching,
		}
	}
	c.JSON(http.StatusOK, result)
}

// POST /webhooks/toggle-event/:eventType
// Enables or disables all webhooks of a specific event type.
func (h *WebhookHandler) ToggleEvent(c *gin.Context) {
	eventType := c.Param("eventType")
	validTypes := map[string]bool{
		"playback_started":     true,
		"playback_ended":       true,
		"media_recently_added": true,
	}
	if !validTypes[eventType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event type"})
		return
	}

	var body struct {
		Enabled bool   `json:"enabled"`
		URL     string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enabled parameter is required"})
		return
	}

	ctx := c.Request.Context()
	hooks, err := h.svcs.Webhook.List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	affected := 0
	for _, hook := range hooks {
		if hook.TriggerType == "event" && hook.EventType != nil && *hook.EventType == eventType {
			hook.Enabled = body.Enabled
			_ = h.svcs.Webhook.Update(ctx, &hook)
			affected++
		}
	}

	// If no webhook existed and we want to enable, create a default one if URL provided.
	if affected == 0 && body.Enabled {
		if body.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":    "URL parameter is required when creating a new webhook",
				"needsUrl": true,
			})
			return
		}
		et := eventType
		newHook := &models.Webhook{
			Name:        "Webhook pour " + eventType,
			Url:         body.URL,
			Method:      "POST",
			TriggerType: "event",
			EventType:   &et,
			Enabled:     true,
		}
		_ = h.svcs.Webhook.Create(ctx, newHook)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Webhooks for " + eventType + " updated",
		"affectedCount": affected,
	})
}

// fireWebhookHTTP sends a single HTTP request to the given webhook URL with the given payload.
func fireWebhookHTTP(ctx context.Context, wh *models.Webhook, payload any) error {
	body, _ := json.Marshal(payload)
	method := wh.Method
	if method == "" {
		method = "POST"
	}
	req, err := http.NewRequestWithContext(ctx, method, wh.Url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Apply custom headers from the webhook.
	var headers map[string]string
	if err := json.Unmarshal(wh.Headers, &headers); err == nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// isDiscordURL returns true if the URL looks like a Discord webhook.
func isDiscordURL(url string) bool {
	return strings.Contains(url, "discord.com/api/webhooks")
}

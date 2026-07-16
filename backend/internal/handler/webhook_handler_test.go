package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service"
	webhooksvc "github.com/Jellystics/Jellystics/internal/service/webhook"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
)

func itoa(n int) string { return strconv.Itoa(n) }

// putJSON issues a PUT with a JSON body.
func putJSON(t *testing.T, r *gin.Engine, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, target, jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// doDelete issues a DELETE with no body.
func doDelete(t *testing.T, r *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, target, nil))
	return w
}

func webhookHandler(t *testing.T) (*handler.WebhookHandler, *repository.Container, *gin.Engine) {
	t.Helper()
	db := testutil.NewDB(t)
	repos := repository.New(db)
	svcs := &service.Container{Webhook: webhooksvc.New(repos)}
	gin.SetMode(gin.TestMode)
	return handler.NewWebhookHandler(svcs), repos, gin.New()
}

func TestWebhookCRUD(t *testing.T) {
	h, _, r := webhookHandler(t)
	r.GET("/webhooks/", h.List)
	r.GET("/webhooks/:id", h.Get)
	r.POST("/webhooks/", h.Create)
	r.PUT("/webhooks/:id", h.Update)
	r.DELETE("/webhooks/:id", h.Delete)

	// Empty list.
	w := do(t, r, "/webhooks/")
	var hooks []models.Webhook
	_ = json.Unmarshal(w.Body.Bytes(), &hooks)
	if len(hooks) != 0 {
		t.Fatalf("initial hooks = %d, want 0", len(hooks))
	}

	// Create.
	cw := postJSON(t, r, "/webhooks/", `{"name":"Discord","url":"http://example.com/hook","trigger_type":"event","event_type":"task_complete","enabled":true}`)
	if cw.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body %s", cw.Code, cw.Body.String())
	}
	var created models.Webhook
	_ = json.Unmarshal(cw.Body.Bytes(), &created)
	if created.Id == 0 {
		t.Fatalf("created webhook has no id: %v", created)
	}

	// Get by id.
	gw := do(t, r, "/webhooks/"+itoa(created.Id))
	if gw.Code != http.StatusOK {
		t.Fatalf("get status = %d", gw.Code)
	}
	var got models.Webhook
	_ = json.Unmarshal(gw.Body.Bytes(), &got)
	if got.Name != "Discord" {
		t.Errorf("name = %q, want Discord", got.Name)
	}

	// Update: disable it. A realistic client round-trips the full object
	// (including discord_events/headers/payload), so include them here.
	uw := putJSON(t, r, "/webhooks/"+itoa(created.Id), `{"name":"Discord2","url":"http://example.com/hook","method":"POST","trigger_type":"event","event_type":"task_complete","enabled":false,"headers":{},"payload":{},"discord_events":[]}`)
	if uw.Code != http.StatusOK {
		t.Fatalf("update status = %d, body %s", uw.Code, uw.Body.String())
	}
	gw2 := do(t, r, "/webhooks/"+itoa(created.Id))
	_ = json.Unmarshal(gw2.Body.Bytes(), &got)
	if got.Name != "Discord2" || got.Enabled {
		t.Errorf("after update = %+v, want Discord2 disabled", got)
	}

	// Delete.
	dw := doDelete(t, r, "/webhooks/"+itoa(created.Id))
	if dw.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", dw.Code)
	}
	// Now not found.
	if nf := do(t, r, "/webhooks/"+itoa(created.Id)); nf.Code != http.StatusNotFound {
		t.Errorf("get after delete status = %d, want 404", nf.Code)
	}
}

func TestWebhookGet_NotFound(t *testing.T) {
	h, _, r := webhookHandler(t)
	r.GET("/webhooks/:id", h.Get)
	if w := do(t, r, "/webhooks/999"); w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestWebhookEventStatus(t *testing.T) {
	h, repos, r := webhookHandler(t)
	// Create an enabled task_complete event webhook.
	et := "task_complete"
	_ = repos.Webhook.Create(t.Context(), &models.Webhook{
		Name: "H", Url: "http://x", TriggerType: "event", EventType: &et, Enabled: true,
	})
	r.GET("/webhooks/event-status", h.EventStatus)

	w := do(t, r, "/webhooks/event-status")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var status map[string]struct {
		Exists  bool `json:"exists"`
		Enabled bool `json:"enabled"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &status)
	if !status["task_complete"].Exists || !status["task_complete"].Enabled {
		t.Errorf("task_complete = %+v, want exists+enabled", status["task_complete"])
	}
	if status["task_failed"].Exists {
		t.Error("task_failed should not exist")
	}
}

func TestWebhookToggleEvent_Invalid(t *testing.T) {
	h, _, r := webhookHandler(t)
	r.POST("/webhooks/toggle-event/:eventType", h.ToggleEvent)
	if w := postJSON(t, r, "/webhooks/toggle-event/bogus", `{"enabled":true}`); w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for invalid event type", w.Code)
	}
}

func TestWebhookToggleEvent_NeedsUrl(t *testing.T) {
	h, _, r := webhookHandler(t)
	r.POST("/webhooks/toggle-event/:eventType", h.ToggleEvent)
	// Enabling with no existing hook and no URL → 400 needsUrl.
	w := postJSON(t, r, "/webhooks/toggle-event/task_start", `{"enabled":true}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["needsUrl"] != true {
		t.Errorf("needsUrl = %v, want true", got["needsUrl"])
	}
}

func TestWebhookToggleEvent_CreatesDefault(t *testing.T) {
	h, repos, r := webhookHandler(t)
	r.POST("/webhooks/toggle-event/:eventType", h.ToggleEvent)
	w := postJSON(t, r, "/webhooks/toggle-event/task_start", `{"enabled":true,"url":"http://new"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	hooks, _ := repos.Webhook.List(t.Context())
	if len(hooks) != 1 || hooks[0].EventType == nil || *hooks[0].EventType != "task_start" {
		t.Errorf("hooks = %+v, want one task_start hook created", hooks)
	}
}

// TestWebhookTest_FiresHTTP stands up a mock endpoint and verifies the test
// button actually POSTs to the webhook URL.
func TestWebhookTest_FiresHTTP(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	h, repos, r := webhookHandler(t)
	et := "task_complete"
	wh := &models.Webhook{Name: "H", Url: srv.URL, TriggerType: "event", EventType: &et, Enabled: true}
	_ = repos.Webhook.Create(t.Context(), wh)
	r.POST("/webhooks/:id/test", h.Test)

	w := postJSON(t, r, "/webhooks/"+itoa(wh.Id)+"/test", `{}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("mock endpoint hits = %d, want 1", hits)
	}
}

func TestWebhookTest_NotFound(t *testing.T) {
	h, _, r := webhookHandler(t)
	r.POST("/webhooks/:id/test", h.Test)
	if w := postJSON(t, r, "/webhooks/999/test", `{}`); w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

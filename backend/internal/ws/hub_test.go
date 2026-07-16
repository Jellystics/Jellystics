package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestNewHub verifies NewHub returns a usable, initialized hub.
func TestNewHub(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("NewHub() returned nil")
	}
	if h.clients == nil {
		t.Error("clients map not initialized")
	}
	if h.sessions == nil {
		t.Error("sessions map not initialized")
	}
}

// TestPollingHandshake verifies the EIO4 open packet: a GET with
// transport=polling and no sid returns a "0{...}" packet containing sid,
// upgrades, and ping settings.
func TestPollingHandshake(t *testing.T) {
	h := NewHub()
	req := httptest.NewRequest(http.MethodGet, "/socket.io/?transport=polling", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, "0{") {
		t.Fatalf("expected open packet starting with %q, got %q", "0{", body)
	}

	var open struct {
		SID          string   `json:"sid"`
		Upgrades     []string `json:"upgrades"`
		PingInterval int      `json:"pingInterval"`
		PingTimeout  int      `json:"pingTimeout"`
		MaxPayload   int      `json:"maxPayload"`
	}
	if err := json.Unmarshal([]byte(body[1:]), &open); err != nil {
		t.Fatalf("failed to parse handshake JSON: %v (body=%q)", err, body)
	}
	if open.SID == "" {
		t.Error("handshake missing sid")
	}
	if len(open.Upgrades) != 1 || open.Upgrades[0] != "websocket" {
		t.Errorf("expected upgrades [websocket], got %v", open.Upgrades)
	}
	if open.PingInterval <= 0 {
		t.Errorf("expected positive pingInterval, got %d", open.PingInterval)
	}
	if open.PingTimeout <= 0 {
		t.Errorf("expected positive pingTimeout, got %d", open.PingTimeout)
	}

	// Handshake must register the session server-side.
	if h.getSession(open.SID) == nil {
		t.Error("handshake did not create a server-side session")
	}
}

// pollHandshake performs the initial polling handshake and returns the sid.
func pollHandshake(t *testing.T, h *Hub) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/socket.io/?transport=polling", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.HasPrefix(body, "0{") {
		t.Fatalf("bad handshake: %q", body)
	}
	var open struct {
		SID string `json:"sid"`
	}
	if err := json.Unmarshal([]byte(body[1:]), &open); err != nil {
		t.Fatalf("parse handshake: %v", err)
	}
	return open.SID
}

// TestPollingPostQueuesNamespaceAck verifies POST with a valid sid queues a
// "40" namespace ack (returning "ok"), and a subsequent GET drains it.
func TestPollingPostQueuesNamespaceAck(t *testing.T) {
	h := NewHub()
	sid := pollHandshake(t, h)

	// POST the namespace connect packet.
	postReq := httptest.NewRequest(http.MethodPost, "/socket.io/?transport=polling&sid="+sid, strings.NewReader("40"))
	postRec := httptest.NewRecorder()
	h.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("POST: expected 200, got %d", postRec.Code)
	}
	if got := postRec.Body.String(); got != "ok" {
		t.Fatalf("POST: expected %q, got %q", "ok", got)
	}

	// GET should now drain the pending "40" ack.
	getReq := httptest.NewRequest(http.MethodGet, "/socket.io/?transport=polling&sid="+sid, nil)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	if got := getRec.Body.String(); got != "40" {
		t.Fatalf("GET after POST: expected pending %q, got %q", "40", got)
	}
}

// TestPollingGetUnknownSidReturnsClose verifies GET with an unknown sid
// returns "1" (EIO close).
func TestPollingGetUnknownSidReturnsClose(t *testing.T) {
	h := NewHub()
	req := httptest.NewRequest(http.MethodGet, "/socket.io/?transport=polling&sid=does-not-exist", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Body.String(); got != "1" {
		t.Fatalf("GET unknown sid: expected %q (EIO close), got %q", "1", got)
	}
}

// TestPollingGetNoPendingReturnsNoop verifies GET on a valid sid with nothing
// queued returns "6" (EIO noop).
func TestPollingGetNoPendingReturnsNoop(t *testing.T) {
	h := NewHub()
	sid := pollHandshake(t, h)

	req := httptest.NewRequest(http.MethodGet, "/socket.io/?transport=polling&sid="+sid, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Body.String(); got != "6" {
		t.Fatalf("GET no pending: expected %q (EIO noop), got %q", "6", got)
	}
}

// pollGet drains pending packets for sid using a fresh recorder.
func pollGet(t *testing.T, h *Hub, sid string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/socket.io/?transport=polling&sid="+sid, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Body.String()
}

// TestPollingMultiplePacketsJoinedByRecordSeparator verifies multiple pending
// packets are joined by the \x1e record separator on drain.
func TestPollingMultiplePacketsJoinedByRecordSeparator(t *testing.T) {
	h := NewHub()
	sid := pollHandshake(t, h)

	// Two POSTs queue two "40" acks.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/socket.io/?transport=polling&sid="+sid, strings.NewReader("40"))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}

	got := pollGet(t, h, sid)
	want := "40\x1e40"
	if got != want {
		t.Fatalf("expected packets joined by record separator %q, got %q", want, got)
	}

	// After draining, a further GET yields noop.
	if got := pollGet(t, h, sid); got != "6" {
		t.Fatalf("expected %q after drain, got %q", "6", got)
	}
}

// TestMissingTransportReturns400 verifies a request without a transport param
// returns HTTP 400.
func TestMissingTransportReturns400(t *testing.T) {
	h := NewHub()
	req := httptest.NewRequest(http.MethodGet, "/socket.io/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing transport: expected 400, got %d", rec.Code)
	}
}

// TestInvalidTransportReturns400 verifies an unknown transport returns 400.
func TestInvalidTransportReturns400(t *testing.T) {
	h := NewHub()
	req := httptest.NewRequest(http.MethodGet, "/socket.io/?transport=carrierpigeon", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid transport: expected 400, got %d", rec.Code)
	}
}

// TestEmitWithZeroClientsIsNoop verifies Emit with no connected clients does
// not panic and is a safe no-op.
func TestEmitWithZeroClientsIsNoop(t *testing.T) {
	h := NewHub()
	// Must not panic.
	h.Emit("TestEvent", map[string]string{"hello": "world"})
}

// TestWebSocketUpgradeAndEmit connects a real WS client, performs the EIO4
// upgrade handshake, and verifies Emit delivers a 42[...] event frame.
func TestWebSocketUpgradeAndEmit(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(http.HandlerFunc(h.ServeHTTP))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/?transport=websocket"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test done")

	// EIO4 upgrade handshake: client sends "2probe".
	if err := conn.Write(ctx, websocket.MessageText, []byte("2probe")); err != nil {
		t.Fatalf("write 2probe: %v", err)
	}
	// Server replies "3probe".
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read after 2probe: %v", err)
	}
	if string(data) != "3probe" {
		t.Fatalf("expected %q, got %q", "3probe", string(data))
	}
	// Client sends "5" (upgrade complete).
	if err := conn.Write(ctx, websocket.MessageText, []byte("5")); err != nil {
		t.Fatalf("write 5: %v", err)
	}
	// Server sends namespace ack "40{"sid":...}".
	_, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf("read namespace ack: %v", err)
	}
	if !strings.HasPrefix(string(data), "40{") {
		t.Fatalf("expected namespace ack starting with %q, got %q", "40{", string(data))
	}
	var ack struct {
		SID string `json:"sid"`
	}
	if err := json.Unmarshal(data[2:], &ack); err != nil {
		t.Fatalf("parse namespace ack: %v (data=%q)", err, string(data))
	}
	if ack.SID == "" {
		t.Error("namespace ack missing sid")
	}

	// The client should now be registered. Poll briefly since registration
	// happens after the ack write returns.
	if !waitForClients(h, 1, 2*time.Second) {
		t.Fatal("client was not registered after handshake")
	}

	// Emit an event; the client should receive a 42["TestEvent",<json>] frame.
	payload := map[string]any{"count": 42, "name": "jelly"}
	h.Emit("TestEvent", payload)

	_, evt, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read emitted event: %v", err)
	}
	got := string(evt)
	const prefix = `42["TestEvent",`
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("expected event frame starting with %q, got %q", prefix, got)
	}
	// The trailing part before the closing "]" must be valid JSON matching payload.
	inner := strings.TrimSuffix(strings.TrimPrefix(got, `42["TestEvent",`), "]")
	var decoded map[string]any
	if err := json.Unmarshal([]byte(inner), &decoded); err != nil {
		t.Fatalf("emitted payload not valid JSON: %v (inner=%q)", err, inner)
	}
	if decoded["name"] != "jelly" {
		t.Errorf("expected name=jelly, got %v", decoded["name"])
	}
	if decoded["count"].(float64) != 42 {
		t.Errorf("expected count=42, got %v", decoded["count"])
	}
}

// waitForClients polls until the hub has exactly n clients or the timeout hits.
func waitForClients(h *Hub, n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		h.mu.RLock()
		count := len(h.clients)
		h.mu.RUnlock()
		if count == n {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

const pingInterval = 25 * time.Second

// Hub is a minimal Socket.IO v4 / Engine.IO v4 server.
// It supports the EIO4 polling handshake + WebSocket upgrade and
// lets callers broadcast Socket.IO events to all connected WS clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}

	sessionMu sync.Mutex
	sessions  map[string]*pollSession
}

type wsClient struct {
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

// pollSession holds state for a client during the polling phase.
type pollSession struct {
	createdAt time.Time
	mu        sync.Mutex
	pending   []string
}

func NewHub() *Hub {
	h := &Hub{
		clients:  make(map[*wsClient]struct{}),
		sessions: make(map[string]*pollSession),
	}
	go h.cleanupLoop()
	return h
}

// Emit sends a Socket.IO event to all connected WebSocket clients.
func (h *Hub) Emit(event string, data any) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return
	}
	// Socket.IO v4 event packet: 42["EventName",data]
	packet := fmt.Sprintf(`42[%q,%s]`, event, encoded)
	raw := []byte(packet)

	h.mu.RLock()
	clients := make([]*wsClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	log.Printf("[ws] Emit %s → %d client(s)", event, len(clients))
	for _, c := range clients {
		if err := c.conn.Write(c.ctx, websocket.MessageText, raw); err != nil {
			log.Printf("[ws] Emit %s write error (removing client): %v", event, err)
			h.removeClient(c)
		}
	}
}

// ServeHTTP handles all /socket.io/ requests (polling + WS upgrade).
// Mount this with: r.Any("/socket.io/*path", gin.WrapH(hub))
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	transport := r.URL.Query().Get("transport")
	sid := r.URL.Query().Get("sid")

	switch transport {
	case "polling":
		h.handlePolling(w, r, sid)
	case "websocket":
		h.handleWebSocket(w, r, sid)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

// handlePolling serves the EIO4 HTTP long-polling transport.
func (h *Hub) handlePolling(w http.ResponseWriter, r *http.Request, sid string) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")

	if r.Method == http.MethodGet && sid == "" {
		// Initial handshake - create session
		newSid := uuid.New().String()
		h.sessionMu.Lock()
		h.sessions[newSid] = &pollSession{
			createdAt: time.Now(),
			pending:   []string{},
		}
		h.sessionMu.Unlock()

		handshake := fmt.Sprintf(
			`0{"sid":%q,"upgrades":["websocket"],"pingInterval":25000,"pingTimeout":5000,"maxPayload":1000000}`,
			newSid,
		)
		fmt.Fprint(w, handshake)
		return
	}

	sess := h.getSession(sid)

	switch r.Method {
	case http.MethodPost:
		// Client sends namespace connect packet (e.g. "40" or "40{"token":"..."}").
		// Queue the namespace connect ack back.
		if sess != nil {
			sess.mu.Lock()
			sess.pending = append(sess.pending, "40")
			sess.mu.Unlock()
		}
		fmt.Fprint(w, "ok")

	case http.MethodGet:
		if sess == nil {
			// Unknown or expired session
			fmt.Fprint(w, "1") // EIO close
			return
		}
		sess.mu.Lock()
		pending := sess.pending
		sess.pending = nil
		sess.mu.Unlock()

		if len(pending) > 0 {
			// EIO4 multiple packets are separated by \x1e (record separator)
			fmt.Fprint(w, strings.Join(pending, "\x1e"))
		} else {
			fmt.Fprint(w, "6") // EIO noop
		}
	}
}

// handleWebSocket handles the EIO4 WebSocket upgrade.
func (h *Hub) handleWebSocket(w http.ResponseWriter, r *http.Request, sid string) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}

	// IMPORTANT: use Background, NOT r.Context().
	// r.Context() is cancelled when the gin handler returns, which would
	// immediately kill the WS connection.
	ctx, cancel := context.WithCancel(context.Background())

	// EIO4 upgrade handshake:
	//   client → "2probe"
	//   server → "3probe"
	//   client → "5" (upgrade complete)
	_, data, err := conn.Read(ctx)
	if err != nil || string(data) != "2probe" {
		cancel()
		conn.CloseNow()
		return
	}
	if err := conn.Write(ctx, websocket.MessageText, []byte("3probe")); err != nil {
		cancel()
		conn.CloseNow()
		return
	}
	_, data, err = conn.Read(ctx)
	if err != nil || string(data) != "5" {
		cancel()
		conn.CloseNow()
		return
	}

	// Remove the polling session now that WS took over
	if sid != "" {
		h.sessionMu.Lock()
		delete(h.sessions, sid)
		h.sessionMu.Unlock()
	}

	// Socket.IO v4 namespace ack: must include "sid" for the client to fire
	// its "connect" event and register event listeners. Bare "40" is insufficient.
	newSID := uuid.New().String()
	nsAck := fmt.Sprintf(`40{"sid":%q}`, newSID)
	if err := conn.Write(ctx, websocket.MessageText, []byte(nsAck)); err != nil {
		cancel()
		conn.CloseNow()
		return
	}

	c := &wsClient{conn: conn, ctx: ctx, cancel: cancel}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	total := len(h.clients)
	h.mu.Unlock()
	log.Printf("[ws] client connected (total=%d)", total)

	// Read loop – keeps connection alive and detects disconnects
	go func() {
		defer h.removeClient(c)
		for {
			_, msg, err := conn.Read(ctx)
			if err != nil {
				return
			}
			s := string(msg)
			if s != "3" { // ignore pong packets
				log.Printf("[ws] recv: %q", s)
			}
		}
	}()

	// Ping loop – send EIO ping every 25 s
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := conn.Write(ctx, websocket.MessageText, []byte("2")); err != nil {
					return
				}
			}
		}
	}()
}

func (h *Hub) removeClient(c *wsClient) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		c.cancel()
		c.conn.CloseNow()
		log.Printf("[ws] client disconnected (total=%d)", len(h.clients))
	}
	h.mu.Unlock()
}

func (h *Hub) getSession(sid string) *pollSession {
	h.sessionMu.Lock()
	defer h.sessionMu.Unlock()
	return h.sessions[sid]
}

// cleanupLoop removes stale polling sessions periodically.
func (h *Hub) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-10 * time.Minute)
		h.sessionMu.Lock()
		for sid, sess := range h.sessions {
			if sess.createdAt.Before(cutoff) {
				delete(h.sessions, sid)
			}
		}
		h.sessionMu.Unlock()
	}
}

package handler

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
)

type ProxyApiHandler struct {
	repos      *repository.Container
	httpClient *http.Client
}

func NewProxyApiHandler(repos *repository.Container) *ProxyApiHandler {
	return &ProxyApiHandler{
		repos:      repos,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// GET /proxy/*path
func (h *ProxyApiHandler) Proxy(c *gin.Context) {
	rawPath := c.Param("path") // e.g., "/Items/Images/Primary/"

	// Dispatch non-image API routes that share the /proxy/* wildcard.
	switch rawPath {
	case "/getSessions":
		h.GetSessions(c)
		return
	case "/getAdminUsers":
		h.GetAdminUsers(c)
		return
	case "/getRecentlyAdded":
		h.GetRecentlyAdded(c)
		return
	}

	// Get id from query
	id := c.Query("id")

	// Build Jellyfin path by inserting id
	// Input: "/Items/Images/Primary/" + id="XYZ" → "/Items/XYZ/Images/Primary"
	// Input: "/Users/Images/Primary/" + id="XYZ" → "/Users/XYZ/Images/Primary"
	cleanPath := strings.Trim(rawPath, "/")
	parts := strings.SplitN(cleanPath, "/", 2)

	var jfPath string
	if len(parts) == 2 {
		suffix := strings.TrimSuffix(parts[1], "/")
		if id != "" {
			jfPath = "/" + parts[0] + "/" + id + "/" + suffix
		} else {
			jfPath = "/" + cleanPath
		}
	} else {
		jfPath = "/" + cleanPath
	}

	// Get JF config from DB
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	// Build query params (everything except 'id')
	q := c.Request.URL.Query()
	q.Del("id")

	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}
	if apiKey != "" {
		q.Set("api_key", apiKey)
	}

	jfURL := strings.TrimRight(*cfg.JFHost, "/") + jfPath
	if len(q) > 0 {
		jfURL += "?" + q.Encode()
	}

	// Make request to Jellyfin
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, jfURL, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// GET /proxy/getSessions — returns live Jellyfin sessions.
func (h *ProxyApiHandler) GetSessions(c *gin.Context) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jellyfin not configured"})
		return
	}
	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}
	jf := jellyfin.NewClient(*cfg.JFHost, apiKey)
	sessions, err := jf.GetSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

// GET /proxy/getAdminUsers — returns Jellyfin users with IsAdministrator = true.
func (h *ProxyApiHandler) GetAdminUsers(c *gin.Context) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jellyfin not configured"})
		return
	}
	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}
	jf := jellyfin.NewClient(*cfg.JFHost, apiKey)
	users, err := jf.GetUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	admins := make([]jellyfin.User, 0)
	for _, u := range users {
		if u.Policy.IsAdministrator {
			admins = append(admins, u)
		}
	}
	c.JSON(http.StatusOK, admins)
}

// GET /proxy/getRecentlyAdded?libraryid=<id> — returns recently added items for a library.
func (h *ProxyApiHandler) GetRecentlyAdded(c *gin.Context) {
	libraryId := c.Query("libraryid")

	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jellyfin not configured"})
		return
	}
	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}
	jf := jellyfin.NewClient(*cfg.JFHost, apiKey)

	// Find the first admin user to use for the /Users/{id}/Items/Latest endpoint.
	users, err := jf.GetUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	var adminId string
	for _, u := range users {
		if u.Policy.IsAdministrator {
			adminId = u.Id
			break
		}
	}
	if adminId == "" && len(users) > 0 {
		adminId = users[0].Id
	}
	if adminId == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no users found"})
		return
	}

	items, err := jf.GetRecentlyAdded(c.Request.Context(), adminId, libraryId, 20, jellyfin.StandardFields)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

// POST /proxy/validateSettings — validates a Jellyfin URL + API key.
// Body: {"url": "...", "apikey": "..."}
func (h *ProxyApiHandler) ValidateSettings(c *gin.Context) {
	var body struct {
		URL    string `json:"url"`
		APIKey string `json:"apikey"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.URL == "" || body.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url and apikey are required"})
		return
	}

	// Normalise the URL (strip trailing slash, add scheme if missing).
	rawURL := body.URL
	rawURL = strings.TrimSuffix(rawURL, "/")
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	info, _, err := jellyfin.GetSystemInfo(c.Request.Context(), rawURL, body.APIKey)
	if err != nil {
		statusCode := http.StatusBadGateway
		c.JSON(http.StatusOK, gin.H{
			"isValid":      false,
			"status":       statusCode,
			"errorMessage": err.Error(),
		})
		return
	}

	serverName, _ := info["ServerName"].(string)
	serverId, _ := info["Id"].(string)
	c.JSON(http.StatusOK, gin.H{
		"isValid":    true,
		"serverName": serverName,
		"serverId":   serverId,
	})
}

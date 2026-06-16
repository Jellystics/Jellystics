package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type FrontendAuthHandler struct {
	repos *repository.Container
	cfg   *config.Config
}

func NewFrontendAuthHandler(repos *repository.Container, cfg *config.Config) *FrontendAuthHandler {
	return &FrontendAuthHandler{repos: repos, cfg: cfg}
}

// GET /auth/isConfigured
func (h *FrontendAuthHandler) IsConfigured(c *gin.Context) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.JSON(http.StatusOK, gin.H{"state": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": 1, "jellyfinUrl": *cfg.JFHost})
}

// POST /auth/configSetup
func (h *FrontendAuthHandler) ConfigSetup(c *gin.Context) {
	var body struct {
		JFHost   string `json:"JF_HOST"`
		JFApiKey string `json:"JF_API_KEY"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errorMessage": err.Error()})
		return
	}
	if body.JFHost == "" {
		c.JSON(http.StatusBadRequest, gin.H{"errorMessage": "JF_HOST is required"})
		return
	}

	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errorMessage": "Failed to load config"})
		return
	}

	host := strings.TrimRight(body.JFHost, "/")
	cfg.JFHost = &host
	if body.JFApiKey != "" {
		cfg.JFApiKey = &body.JFApiKey
	}

	if err := h.repos.Config.Save(c.Request.Context(), cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errorMessage": "Failed to save config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /auth/check-server
func (h *FrontendAuthHandler) CheckServer(c *gin.Context) {
	var body struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}

	// Ping Jellyfin /System/Info/Public
	reqURL := strings.TrimRight(body.URL, "/") + "/System/Info/Public"
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(reqURL)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cannot reach Jellyfin server"})
		return
	}
	defer resp.Body.Close()

	var info struct {
		ServerName string `json:"ServerName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil || info.ServerName == "" {
		info.ServerName = body.URL
	}

	c.JSON(http.StatusOK, gin.H{"serverName": info.ServerName, "url": body.URL})
}

// POST /auth/jellyfin-login
func (h *FrontendAuthHandler) JellyfinLogin(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Get JF config from DB
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Jellyfin not configured"})
		return
	}

	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}

	jfClient := jellyfin.NewClient(*cfg.JFHost, apiKey)
	auth, err := jfClient.AuthenticateUser(c.Request.Context(), body.Username, body.Password)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") || strings.Contains(err.Error(), "401") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Jellyfin unreachable"})
		}
		return
	}

	if !auth.User.Policy.IsAdministrator {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	type Claims struct {
		UserId   string `json:"userId"`
		UserName string `json:"userName"`
		IsAdmin  bool   `json:"isAdmin"`
		jwt.RegisteredClaims
	}

	claims := Claims{
		UserId:   auth.User.Id,
		UserName: auth.User.Name,
		IsAdmin:  auth.User.Policy.IsAdministrator,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": signed})
}

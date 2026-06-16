package handler

import (
	"net/http"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
)

type SettingsHandler struct{ repos *repository.Container }

func NewSettingsHandler(repos *repository.Container) *SettingsHandler {
	return &SettingsHandler{repos}
}

// GET /api/settings
func (h *SettingsHandler) Get(c *gin.Context) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Never expose sensitive fields
	c.JSON(http.StatusOK, gin.H{
		"requireLogin": cfg.RequireLogin,
		"hasHost":      cfg.JFHost != nil && *cfg.JFHost != "",
		"hasApiKey":    cfg.JFApiKey != nil && *cfg.JFApiKey != "",
		"settings":     cfg.Settings,
	})
}

// PUT /api/settings
func (h *SettingsHandler) Update(c *gin.Context) {
	var body struct {
		JFHost       *string `json:"jfHost"`
		JFApiKey     *string `json:"jfApiKey"`
		RequireLogin *bool   `json:"requireLogin"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if body.JFHost != nil {
		cfg.JFHost = body.JFHost
	}
	if body.JFApiKey != nil {
		cfg.JFApiKey = body.JFApiKey
	}
	if body.RequireLogin != nil {
		cfg.RequireLogin = *body.RequireLogin
	}
	if err := h.repos.Config.Save(c.Request.Context(), cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

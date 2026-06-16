package handler

import (
	"net/http"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/middleware"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
)

// SecurityHandler handles security-related endpoints (password change).
type SecurityHandler struct {
	repos *repository.Container
}

func NewSecurityHandler(repos *repository.Container) *SecurityHandler {
	return &SecurityHandler{repos: repos}
}

// POST /api/updatePassword
func (h *SecurityHandler) UpdatePassword(c *gin.Context) {
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.CurrentPassword == "" || body.NewPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current_password and new_password are required"})
		return
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

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

	if err := jfClient.UpdatePassword(c.Request.Context(), claims.UserName, body.CurrentPassword, body.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

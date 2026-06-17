package handler

import (
	"net/http"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
)

type SessionsApiHandler struct {
	repos *repository.Container
}

func NewSessionsApiHandler(repos *repository.Container) *SessionsApiHandler {
	return &SessionsApiHandler{repos: repos}
}

// GET /sessions/current
func (h *SessionsApiHandler) Current(c *gin.Context) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}

	jf := jellyfin.NewClient(*cfg.JFHost, apiKey)
	sessions, err := jf.GetSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

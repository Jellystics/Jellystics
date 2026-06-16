package handler

import (
	"net/http"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
)

// LogsHandler serves task/sync log history from jf_logging.
type LogsHandler struct {
	repos *repository.Container
}

func NewLogsHandler(repos *repository.Container) *LogsHandler {
	return &LogsHandler{repos: repos}
}

// GET /logs/getLogs
// Returns the 50 most recent log entries, ordered by TimeRun desc (matching old Node.js behaviour).
func (h *LogsHandler) GetLogs(c *gin.Context) {
	logs, err := h.repos.Log.List(c.Request.Context(), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if logs == nil {
		logs = []models.JFLogging{}
	}
	c.JSON(http.StatusOK, logs)
}

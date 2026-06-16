package handler

import (
	"context"
	"net/http"

	"github.com/Jellystics/Jellystics/internal/service"
	"github.com/gin-gonic/gin"
)

type SyncHandler struct{ svcs *service.Container }

func NewSyncHandler(svcs *service.Container) *SyncHandler { return &SyncHandler{svcs} }

// POST /api/sync/full
func (h *SyncHandler) FullSync(c *gin.Context) {
	go func() {
		_ = h.svcs.Task.Run(context.Background(), "full-sync", func(ctx context.Context) error {
			return h.svcs.Sync.FullSync(ctx)
		})
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "sync started"})
}

// POST /api/sync/libraries
func (h *SyncHandler) SyncLibraries(c *gin.Context) {
	go func() {
		_ = h.svcs.Task.Run(context.Background(), "sync-libraries", func(ctx context.Context) error {
			_, err := h.svcs.Sync.SyncLibraries(ctx)
			return err
		})
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "libraries sync started"})
}

// POST /api/sync/users
func (h *SyncHandler) SyncUsers(c *gin.Context) {
	if err := h.svcs.Sync.SyncUsers(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "users synced"})
}

// GET /api/sync/status
func (h *SyncHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": h.svcs.Task.Status()})
}

// POST /sync/fetchItem — syncs a single item from Jellyfin into the DB.
// Body: {"Id": "<jellyfin-item-id>"}
func (h *SyncHandler) FetchItem(c *gin.Context) {
	var body struct {
		Id string `json:"Id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Id is required"})
		return
	}

	n, err := h.svcs.Sync.FetchItem(c.Request.Context(), body.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": n})
}

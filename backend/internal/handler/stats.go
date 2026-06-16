package handler

import (
	"net/http"
	"strconv"

	"github.com/Jellystics/Jellystics/internal/service"
	"github.com/gin-gonic/gin"
)

type StatsHandler struct{ svcs *service.Container }

func NewStatsHandler(svcs *service.Container) *StatsHandler { return &StatsHandler{svcs} }

// GET /api/stats/global
func (h *StatsHandler) GlobalStats(c *gin.Context) {
	data, err := h.svcs.Stats.GlobalStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/stats/libraries/most-viewed
func (h *StatsHandler) MostViewedLibraries(c *gin.Context) {
	limit := queryInt(c, "limit", 10)
	data, err := h.svcs.Stats.MostViewedLibraries(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/stats/libraries/:id
func (h *StatsHandler) LibraryStats(c *gin.Context) {
	id := c.Param("id")
	data, err := h.svcs.Stats.LibraryStats(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/stats/activity?days=90
func (h *StatsHandler) ActivityOverTime(c *gin.Context) {
	days := queryInt(c, "days", 90)
	data, err := h.svcs.Stats.ActivityOverTime(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/stats/users/top
func (h *StatsHandler) TopUsers(c *gin.Context) {
	limit := queryInt(c, "limit", 10)
	data, err := h.svcs.Stats.TopUsers(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/stats/libraries/:id/items
func (h *StatsHandler) MostPlayedItems(c *gin.Context) {
	id := c.Param("id")
	limit := queryInt(c, "limit", 10)
	data, err := h.svcs.Stats.MostPlayedItems(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/stats/libraries/:id/artists
func (h *StatsHandler) MostPlayedArtists(c *gin.Context) {
	id := c.Param("id")
	limit := queryInt(c, "limit", 10)
	data, err := h.svcs.Stats.MostPlayedArtists(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/stats/libraries/:id/albums?artistId=...
func (h *StatsHandler) MostPlayedAlbums(c *gin.Context) {
	id := c.Param("id")
	artistId := c.Query("artistId")
	limit := queryInt(c, "limit", 10)
	data, err := h.svcs.Stats.MostPlayedAlbums(c.Request.Context(), id, artistId, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/stats/libraries/:id/tracks?albumId=...
func (h *StatsHandler) MostPlayedTracks(c *gin.Context) {
	id := c.Param("id")
	albumId := c.Query("albumId")
	limit := queryInt(c, "limit", 10)
	data, err := h.svcs.Stats.MostPlayedTracks(c.Request.Context(), id, albumId, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/stats/users/:id/history
func (h *StatsHandler) UserHistory(c *gin.Context) {
	userId := c.Param("id")
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 20)
	entries, total, err := h.svcs.Stats.UserHistory(c.Request.Context(), userId, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": entries, "total": total, "page": page, "pageSize": pageSize})
}

func queryInt(c *gin.Context, key string, def int) int {
	v, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return def
	}
	return v
}

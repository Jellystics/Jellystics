package handler

import (
	"net/http"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LibraryHandler struct {
	svcs  *service.Container
	repos *repository.Container
	db    *gorm.DB
}

func NewLibraryHandler(svcs *service.Container, repos *repository.Container, db *gorm.DB) *LibraryHandler {
	return &LibraryHandler{svcs: svcs, repos: repos, db: db}
}

// GET /api/libraries
func (h *LibraryHandler) List(c *gin.Context) {
	libs, err := h.repos.Library.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, libs)
}

// GET /api/libraries/:id
func (h *LibraryHandler) Get(c *gin.Context) {
	id := c.Param("id")
	lib, err := h.repos.Library.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, lib)
}

// GET /api/libraries/:id/items
func (h *LibraryHandler) Items(c *gin.Context) {
	id := c.Param("id")
	items, err := h.repos.Item.ListByParent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

// GET /api/libraries/:id/episodes
func (h *LibraryHandler) Episodes(c *gin.Context) {
	id := c.Param("id")
	// id here is a series id
	episodes, err := h.repos.Episode.ListBySeries(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, episodes)
}

// GET /api/libraries/:id/seasons
func (h *LibraryHandler) Seasons(c *gin.Context) {
	id := c.Param("id")
	seasons, err := h.repos.Season.ListBySeries(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, seasons)
}

type artistRow struct {
	Id               string  `json:"Id"`
	Name             *string `json:"Name"`
	ImageTagsPrimary *string `json:"ImageTagsPrimary"`
	AlbumCount       int     `json:"AlbumCount"`
	TrackCount       int     `json:"TrackCount"`
	PlayCount        int     `json:"PlayCount"`
}

// GET /api/libraries/:id/artists
func (h *LibraryHandler) Artists(c *gin.Context) {
	id := c.Param("id")
	var rows []artistRow
	err := h.db.WithContext(c.Request.Context()).Raw(`
		SELECT
			ar."Id",
			ar."Name",
			ar."ImageTagsPrimary",
			COUNT(DISTINCT li."Id")::int AS "AlbumCount",
			COUNT(DISTINCT t."Id")::int  AS "TrackCount",
			0                            AS "PlayCount"
		FROM jf_music_artists ar
		LEFT JOIN jf_library_items li
			ON li."ArtistId" = ar."Id" AND li."Type" = 'MusicAlbum' AND li.archived = false
		LEFT JOIN jf_music_tracks t
			ON t."ArtistId" = ar."Id" AND t.archived = false
		WHERE ar."LibraryId" = ? AND ar.archived = false
		GROUP BY ar."Id", ar."Name", ar."ImageTagsPrimary"
		ORDER BY ar."Name" ASC
	`, id).Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []artistRow{}
	}
	c.JSON(http.StatusOK, rows)
}

// GET /api/libraries/:id/tracks
func (h *LibraryHandler) Tracks(c *gin.Context) {
	id := c.Param("id")
	tracks, err := h.repos.MusicTrack.ListByLibrary(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tracks)
}

// GET /api/libraries/:id/albums/:albumId/tracks
func (h *LibraryHandler) AlbumTracks(c *gin.Context) {
	albumId := c.Param("albumId")
	tracks, err := h.repos.MusicTrack.ListByAlbum(c.Request.Context(), albumId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tracks)
}

// GET /api/items/:id
func (h *LibraryHandler) GetItem(c *gin.Context) {
	id := c.Param("id")
	item, err := h.repos.Item.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

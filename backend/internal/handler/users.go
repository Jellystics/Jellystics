package handler

import (
	"net/http"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
)

type UserHandler struct{ repos *repository.Container }

func NewUserHandler(repos *repository.Container) *UserHandler { return &UserHandler{repos} }

// GET /api/users
func (h *UserHandler) List(c *gin.Context) {
	users, err := h.repos.User.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

// GET /api/users/:id
func (h *UserHandler) Get(c *gin.Context) {
	id := c.Param("id")
	user, err := h.repos.User.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/Jellystics/Jellystics/internal/service/auth"
	"github.com/gin-gonic/gin"
)

const claimsKey = "claims"

// Auth validates the Bearer JWT and attaches claims to the context.
func Auth(svc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := svc.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
}

// AdminOnly rejects non-admin users.
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.Get(claimsKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		if !claims.(*auth.Claims).IsAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin only"})
			return
		}
		c.Next()
	}
}

// GetClaims extracts the claims from a gin context (set by Auth middleware).
func GetClaims(c *gin.Context) *auth.Claims {
	v, _ := c.Get(claimsKey)
	if v == nil {
		return nil
	}
	return v.(*auth.Claims)
}

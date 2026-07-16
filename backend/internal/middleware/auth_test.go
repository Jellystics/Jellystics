package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/middleware"
	"github.com/Jellystics/Jellystics/internal/service/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const secret = "mw-secret"

func svc() *auth.Service {
	return auth.New(nil, nil, &config.Config{JWTSecret: secret})
}

func token(t *testing.T, isAdmin bool) string {
	t.Helper()
	claims := auth.Claims{
		UserId:  "u1",
		UserName: "Alice",
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

// TestAuth_MissingHeader verifies requests without a Bearer header are rejected
// with 401 and the handler is never reached.
func TestAuth_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reached := false
	r := gin.New()
	r.GET("/x", middleware.Auth(svc()), func(c *gin.Context) { reached = true; c.Status(200) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if reached {
		t.Error("handler should not run without a token")
	}
}

// TestAuth_InvalidToken verifies a bad token yields 401.
func TestAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", middleware.Auth(svc()), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer garbage.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// TestAuth_ValidToken verifies a valid token passes through and claims are
// attached to the context.
func TestAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotClaims *auth.Claims
	r := gin.New()
	r.GET("/x", middleware.Auth(svc()), func(c *gin.Context) {
		gotClaims = middleware.GetClaims(c)
		c.Status(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token(t, false))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if gotClaims == nil || gotClaims.UserId != "u1" {
		t.Errorf("claims = %+v, want UserId u1 attached", gotClaims)
	}
}

// TestAdminOnly_NoClaims verifies AdminOnly rejects with 401 when no claims are
// present (Auth did not run).
func TestAdminOnly_NoClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", middleware.AdminOnly(), func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// TestAdminOnly_NonAdmin verifies a non-admin user gets 403.
func TestAdminOnly_NonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", middleware.Auth(svc()), middleware.AdminOnly(), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token(t, false))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

// TestAdminOnly_Admin verifies an admin user passes through.
func TestAdminOnly_Admin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reached := false
	r := gin.New()
	r.GET("/x", middleware.Auth(svc()), middleware.AdminOnly(), func(c *gin.Context) { reached = true; c.Status(200) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token(t, true))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !reached {
		t.Errorf("status = %d reached = %v, want 200 & handler reached", w.Code, reached)
	}
}

// TestGetClaims_Nil verifies GetClaims returns nil when nothing is set.
func TestGetClaims_Nil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := middleware.GetClaims(c); got != nil {
		t.Errorf("GetClaims = %+v, want nil", got)
	}
}

package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	repos  *repository.Container
	jf     *jellyfin.Client
	cfg    *config.Config
}

func New(repos *repository.Container, jf *jellyfin.Client, cfg *config.Config) *Service {
	return &Service{repos: repos, jf: jf, cfg: cfg}
}

type Claims struct {
	UserId   string `json:"userId"`
	UserName string `json:"userName"`
	IsAdmin  bool   `json:"isAdmin"`
	jwt.RegisteredClaims
}

// Login authenticates against Jellyfin, then issues a JWT.
func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	// First check if there's a stored JF host; use it
	cfg, err := s.repos.Config.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	if cfg.JFHost == nil || *cfg.JFHost == "" {
		return "", fmt.Errorf("jellyfin host not configured")
	}

	jfClient := jellyfin.NewClient(*cfg.JFHost, "")
	auth, err := jfClient.AuthenticateUser(ctx, username, password)
	if err != nil {
		return "", fmt.Errorf("jellyfin auth: %w", err)
	}

	claims := Claims{
		UserId:   auth.User.Id,
		UserName: auth.User.Name,
		IsAdmin:  auth.User.Policy.IsAdministrator,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT, returning its claims.
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// GetOrCreateAPIKey generates a new API key and stores it in app_config.
func (s *Service) GetConfig(ctx context.Context) (map[string]any, error) {
	cfg, err := s.repos.Config.Get(ctx)
	if err != nil {
		return nil, err
	}
	requireLogin := cfg.RequireLogin
	hasHost := cfg.JFHost != nil && *cfg.JFHost != ""
	return map[string]any{
		"requireLogin": requireLogin,
		"hasHost":      hasHost,
	}, nil
}

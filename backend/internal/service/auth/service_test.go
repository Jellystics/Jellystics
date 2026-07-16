package auth_test

import (
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key"

// sign builds a JWT for the given claims signed with method + secret.
func sign(t *testing.T, method jwt.SigningMethod, key any, claims auth.Claims) string {
	t.Helper()
	tok := jwt.NewWithClaims(method, claims)
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func newService(secret string) *auth.Service {
	return auth.New(nil, nil, &config.Config{JWTSecret: secret})
}

// TestValidateToken_RoundTrip verifies a token signed with the configured secret
// validates and its claims are returned intact.
func TestValidateToken_RoundTrip(t *testing.T) {
	svc := newService(testSecret)
	claims := auth.Claims{
		UserId:   "u1",
		UserName: "Alice",
		IsAdmin:  true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := sign(t, jwt.SigningMethodHS256, []byte(testSecret), claims)

	got, err := svc.ValidateToken(tok)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if got.UserId != "u1" || got.UserName != "Alice" || !got.IsAdmin {
		t.Errorf("claims = %+v, want {u1 Alice true}", got)
	}
}

// TestValidateToken_WrongSecret verifies a token signed with a different secret
// is rejected.
func TestValidateToken_WrongSecret(t *testing.T) {
	svc := newService(testSecret)
	tok := sign(t, jwt.SigningMethodHS256, []byte("other-secret"), auth.Claims{
		UserId: "u1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	if _, err := svc.ValidateToken(tok); err == nil {
		t.Fatal("expected error for token signed with wrong secret")
	}
}

// TestValidateToken_Expired verifies an expired token is rejected.
func TestValidateToken_Expired(t *testing.T) {
	svc := newService(testSecret)
	tok := sign(t, jwt.SigningMethodHS256, []byte(testSecret), auth.Claims{
		UserId: "u1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})
	if _, err := svc.ValidateToken(tok); err == nil {
		t.Fatal("expected error for expired token")
	}
}

// TestValidateToken_NonHMAC verifies tokens using a non-HMAC signing method are
// rejected (guards against the alg-confusion attack).
func TestValidateToken_NonHMAC(t *testing.T) {
	svc := newService(testSecret)
	// A token claiming "none" algorithm must not validate.
	tok := sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, auth.Claims{
		UserId: "u1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	if _, err := svc.ValidateToken(tok); err == nil {
		t.Fatal("expected error for non-HMAC (none) signing method")
	}
}

// TestValidateToken_Garbage verifies a malformed token string is rejected.
func TestValidateToken_Garbage(t *testing.T) {
	svc := newService(testSecret)
	if _, err := svc.ValidateToken("not.a.jwt"); err == nil {
		t.Fatal("expected error for malformed token")
	}
}

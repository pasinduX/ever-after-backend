package functions

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const secret = "test-secret"

func TestAlbumAccessTokenOpensOnlyItsWedding(t *testing.T) {
	token, expires, err := SignAlbumAccess(secret, "w1", time.Hour)
	if err != nil || time.Until(expires) < 59*time.Minute {
		t.Fatalf("sign: %v %v", err, expires)
	}
	if !VerifyAlbumAccess(secret, token, "w1") {
		t.Error("token should open its wedding")
	}
	if VerifyAlbumAccess(secret, token, "w2") {
		t.Error("token must not open another wedding")
	}
	if VerifyAlbumAccess("other-secret", token, "w1") {
		t.Error("token signed with another secret must fail")
	}
	expired, _, _ := SignAlbumAccess(secret, "w1", -time.Minute)
	if VerifyAlbumAccess(secret, expired, "w1") {
		t.Error("expired token must fail")
	}
}

// A guest's album token must never pass as the couple's login.
func TestAlbumAccessTokenIsNotALogin(t *testing.T) {
	token, _, _ := SignAlbumAccess(secret, "w1", time.Hour)
	app := fiber.New()
	app.Get("/", RequireAuth(secret), func(c *fiber.Ctx) error { return c.SendString("in") })
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("album token as login = %d, want 401", resp.StatusCode)
	}

	// And a login token is not an album token.
	login, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "u1", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(secret))
	if VerifyAlbumAccess(secret, login, "w1") {
		t.Error("a login token must not open a password-protected album")
	}
}

func TestOptionalUserID(t *testing.T) {
	login, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "u1", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(secret))
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString(OptionalUserID(c, secret)) })
	for header, want := range map[string]string{"Bearer " + login: "u1", "": "", "Bearer junk": ""} {
		req := httptest.NewRequest("GET", "/", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		resp, _ := app.Test(req)
		body := make([]byte, 16)
		n, _ := resp.Body.Read(body)
		if got := string(body[:n]); got != want {
			t.Errorf("OptionalUserID(%q) = %q, want %q", header, got, want)
		}
	}
}

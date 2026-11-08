package functions

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AlbumAccessHeader carries a guest's album token for password-protected
// weddings. It is separate from Authorization so a guest token can never be
// mistaken for a couple's login.
const AlbumAccessHeader = "X-Album-Access"

const albumAccessType = "album_access"

// albumAccessKey derives a key of its own from the JWT secret, so an album
// token can't be replayed as a login token (and RequireAuth rejects it
// anyway: it carries no subject).
func albumAccessKey(secret string) []byte {
	return []byte(secret + "|album-access")
}

// SignAlbumAccess issues a guest token for one wedding's album.
func SignAlbumAccess(secret, weddingID string, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	expires := now.Add(ttl)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"typ": albumAccessType,
		"wid": weddingID,
		"iat": now.Unix(),
		"exp": expires.Unix(),
	}).SignedString(albumAccessKey(secret))
	return token, expires, err
}

// VerifyAlbumAccess reports whether token grants access to this wedding.
func VerifyAlbumAccess(secret, token, weddingID string) bool {
	if token == "" {
		return false
	}
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return albumAccessKey(secret), nil
	})
	if err != nil || !parsed.Valid {
		return false
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}
	typ, _ := claims["typ"].(string)
	wid, _ := claims["wid"].(string)
	return typ == albumAccessType && wid == weddingID
}

// OptionalUserID returns the signed-in user's ID when the request carries a
// valid login token, or "" for anonymous guests. Public routes use it to let
// a couple see their own private album.
func OptionalUserID(c *fiber.Ctx, secret string) string {
	header := c.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return ""
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ""
	}
	sub, _ := claims["sub"].(string)
	return sub
}

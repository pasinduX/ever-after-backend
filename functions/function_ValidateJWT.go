package functions

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/storyvows/backend/dto"
)

const UserIDKey = "user_id"

func Logger(c *fiber.Ctx) error {
	start := time.Now()
	err := c.Next()
	status := c.Response().StatusCode()
	logAttrs := []any{
		"method", c.Method(),
		"path", c.Path(),
		"status", status,
		"duration", time.Since(start),
		"ip", c.IP(),
		"user_agent", c.Get("User-Agent"),
	}
	if err != nil {
		logAttrs = append(logAttrs, "error", err.Error())
	}
	if status >= 500 {
		slog.Error("request", logAttrs...)
	} else {
		slog.Info("request", logAttrs...)
	}
	return err
}

func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return writeUnauthorized(c, "missing or invalid authorization header")
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			return writeUnauthorized(c, "invalid or expired token")
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return writeUnauthorized(c, "invalid token claims")
		}
		userID, ok := claims["sub"].(string)
		if !ok {
			return writeUnauthorized(c, "invalid token subject")
		}
		c.Locals(UserIDKey, userID)
		return c.Next()
	}
}

func GetUserID(c *fiber.Ctx) string {
	userID, _ := c.Locals(UserIDKey).(string)
	return userID
}

func writeUnauthorized(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Error: msg})
}

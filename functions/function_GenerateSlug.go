package functions

import (
	"strings"

	"github.com/google/uuid"
)

// GenerateSlug produces a short, URL-safe unique identifier suitable for QR slugs.
func GenerateSlug() string {
	raw := uuid.NewString()
	return strings.ReplaceAll(raw[:8], "-", "")
}

package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

// AlbumPhotos returns the photos an album page may show, with fresh links.
// The signature album resolves its mv-photo:// placeholders from this list.
func AlbumPhotos(weddings *service.WeddingService, uploads *service.UploadService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wedding, err := weddings.Get(c.UserContext(), c.Params("id"), getUserID(c))
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		photos, err := uploads.AlbumPhotos(c.UserContext(), wedding.ID)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		// Links are presigned, so the response must not be cached past their TTL.
		c.Set("Cache-Control", "private, no-store")
		return utils.SendJSON(c, fiber.StatusOK, photos)
	}
}

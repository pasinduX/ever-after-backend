package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

// GetAlbumPage returns the newest album page version's progress and the
// newest page that is ready to show (which may be an earlier version).
func GetAlbumPage(albums *service.AlbumService, weddings *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wedding, err := weddings.Get(c.UserContext(), c.Params("id"), getUserID(c))
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		state, err := albums.PageState(c.UserContext(), wedding.ID)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, state)
	}
}

// RegenerateAlbumPage queues a new design from the wedding's current acts,
// for after the couple has reviewed them or wants a fresh look.
func RegenerateAlbumPage(albums *service.AlbumService, weddings *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wedding, err := weddings.Get(c.UserContext(), c.Params("id"), getUserID(c))
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		state, err := albums.PageState(c.UserContext(), wedding.ID)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		if state.Latest != nil && state.Latest.Status == dto.AlbumPageGenerating {
			return utils.SendErrorResponse(c, fiber.StatusConflict, "an album design is already being written")
		}
		if !albums.RegeneratePage(wedding.ID) {
			return utils.SendErrorResponse(c, fiber.StatusServiceUnavailable, "album queue is full; try again shortly")
		}
		return utils.SendJSON(c, fiber.StatusAccepted, fiber.Map{"wedding_id": wedding.ID, "queued": true})
	}
}

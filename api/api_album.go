package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

func GenerateAlbum(svc *service.AlbumService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		weddingID := c.Params("id")
		if weddingID == "" {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "wedding id required")
		}

		var req dto.GenerateAlbumRequest
		if err := c.BodyParser(&req); err != nil {
			req.Style = dto.AlbumStyleCinematic
		}
		if req.Style == "" {
			req.Style = dto.AlbumStyleCinematic
		}

		if err := svc.GenerateAsync(c.UserContext(), weddingID, req.Style); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to start album generation")
		}

		return utils.SendSuccessResponse(c, "album generation started", fiber.Map{
			"wedding_id": weddingID,
			"status":     "processing",
		})
	}
}

func GetAlbumStatus(svc *service.AlbumService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		weddingID := c.Params("id")
		if weddingID == "" {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "wedding id required")
		}

		cfg, err := svc.GetStatus(c.UserContext(), weddingID)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to get album status")
		}

		return utils.SendJSON(c, fiber.StatusOK, cfg)
	}
}

func GetActs(svc *service.AlbumService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		weddingID := c.Params("id")
		if weddingID == "" {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "wedding id required")
		}

		acts, err := svc.GetActs(c.UserContext(), weddingID)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to get acts")
		}

		return utils.SendJSON(c, fiber.StatusOK, fiber.Map{"acts": acts})
	}
}

func UpdateActs(svc *service.AlbumService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		weddingID := c.Params("id")
		if weddingID == "" {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "wedding id required")
		}

		var req dto.ConfirmActsRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}

		if err := svc.ConfirmActs(c.UserContext(), weddingID, req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to update acts")
		}

		return utils.SendSuccessResponse(c, "acts updated", nil)
	}
}

func SetAlbumStyle(svc *service.AlbumService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		weddingID := c.Params("id")
		if weddingID == "" {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "wedding id required")
		}

		var req dto.SetAlbumStyleRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		if req.Style == "" {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "style is required")
		}

		if err := svc.SetStyle(c.UserContext(), weddingID, req.Style); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to set style")
		}

		return utils.SendSuccessResponse(c, "style updated", nil)
	}
}

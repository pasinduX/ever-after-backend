package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

func weddingServiceError(c *fiber.Ctx, err error) error {
	return utils.SendServiceError(c, err)
}

func CreateWedding(svc *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.CreateWeddingRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		result, err := svc.Create(c.UserContext(), getUserID(c), req)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		return utils.SendJSON(c, fiber.StatusCreated, result)
	}
}

func ListWeddings(svc *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		list, err := svc.List(c.UserContext(), getUserID(c))
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to list weddings")
		}
		return utils.SendJSON(c, fiber.StatusOK, list)
	}
}

func GetWedding(svc *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := svc.Get(c.UserContext(), c.Params("id"), getUserID(c))
		if err != nil {
			return weddingServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

func PublicWedding(svc *service.WeddingService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := svc.GetPublicByID(c.UserContext(), c.Params("id"))
		if err != nil {
			return weddingServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

func UpdateWedding(svc *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.UpdateWeddingRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		result, err := svc.Update(c.UserContext(), c.Params("id"), getUserID(c), req)
		if err != nil {
			return weddingServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

func DeleteWedding(svc *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := svc.Delete(c.UserContext(), c.Params("id"), getUserID(c)); err != nil {
			return weddingServiceError(c, err)
		}
		return utils.SendSuccessResponse(c, "wedding deleted", nil)
	}
}

func SetPrivacyWedding(svc *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.PrivacyRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		if err := svc.SetPrivacy(c.UserContext(), c.Params("id"), getUserID(c), req); err != nil {
			return weddingServiceError(c, err)
		}
		return utils.SendSuccessResponse(c, "privacy updated", nil)
	}
}

func GuestViewWedding(svc *service.WeddingService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := svc.GetPublicByIdentifier(c.UserContext(), c.Params("id"))
		if err != nil {
			return weddingServiceError(c, err)
		}
		result.PasswordHash = nil
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

func GuestAccessWedding(svc *service.WeddingService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.GuestAccessRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		result, err := svc.VerifyGuestAccessByIdentifier(c.UserContext(), c.Params("id"), req.Password)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusUnauthorized, err.Error())
		}
		result.PasswordHash = nil
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

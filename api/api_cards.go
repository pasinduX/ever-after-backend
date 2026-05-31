package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

func CreateInvite(svc *service.CardsService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.CreateInviteRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		result, err := svc.CreateInvite(c.UserContext(), getUserID(c), c.Params("id"), req)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		return utils.SendJSON(c, fiber.StatusCreated, result)
	}
}

func GetInvite(svc *service.CardsService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := svc.GetInvite(c.UserContext(), getUserID(c), c.Params("id"))
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

func UpdateInvite(svc *service.CardsService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.UpdateInviteRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		result, err := svc.UpdateInvite(c.UserContext(), getUserID(c), c.Params("id"), req)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

func DeleteInvite(svc *service.CardsService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := svc.DeleteInvite(c.UserContext(), getUserID(c), c.Params("id")); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		return utils.SendSuccessResponse(c, "invite deleted", nil)
	}
}

func CreateThankYou(svc *service.CardsService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.CreateThankYouRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		result, err := svc.CreateThankYou(c.UserContext(), getUserID(c), c.Params("id"), req)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		return utils.SendJSON(c, fiber.StatusCreated, result)
	}
}

func GetThankYou(svc *service.CardsService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := svc.GetThankYou(c.UserContext(), getUserID(c), c.Params("id"))
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

func UpdateThankYou(svc *service.CardsService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.UpdateThankYouRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		result, err := svc.UpdateThankYou(c.UserContext(), getUserID(c), c.Params("id"), req)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

func DeleteThankYou(svc *service.CardsService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := svc.DeleteThankYou(c.UserContext(), getUserID(c), c.Params("id")); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		return utils.SendSuccessResponse(c, "thank you deleted", nil)
	}
}

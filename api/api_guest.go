package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

func guestServiceError(c *fiber.Ctx, err error) error {
	return utils.SendServiceError(c, err)
}

func CreateGuest(svc *service.GuestService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.CreateGuestRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		result, err := svc.Create(c.UserContext(), getUserID(c), c.Params("id"), req)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, err.Error())
		}
		return utils.SendJSON(c, fiber.StatusCreated, result)
	}
}

func ListGuests(svc *service.GuestService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		list, err := svc.List(c.UserContext(), getUserID(c), c.Params("id"))
		if err != nil {
			return guestServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, list)
	}
}

func GetGuest(svc *service.GuestService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := svc.Get(c.UserContext(), getUserID(c), c.Params("id"), c.Params("guestId"))
		if err != nil {
			return guestServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

func UpdateGuest(svc *service.GuestService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.UpdateGuestRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		result, err := svc.Update(c.UserContext(), getUserID(c), c.Params("id"), c.Params("guestId"), req)
		if err != nil {
			return guestServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, result)
	}
}

func DeleteGuest(svc *service.GuestService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := svc.Delete(c.UserContext(), getUserID(c), c.Params("id"), c.Params("guestId")); err != nil {
			return guestServiceError(c, err)
		}
		return utils.SendSuccessResponse(c, "guest deleted", nil)
	}
}

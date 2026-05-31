package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

func SendWhatsAppMessage(svc *service.WhatsAppService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.SendWhatsAppRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		if err := svc.Send(c.UserContext(), req.PhoneNumber, req.Message); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
		}
		return utils.SendSuccessResponse(c, "whatsapp message sent", nil)
	}
}

func SendWhatsAppTemplateMessage(svc *service.WhatsAppService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.SendWhatsAppTemplateRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		if err := svc.SendTemplate(c.UserContext(), req.PhoneNumber, req.ContentSid, req.ContentVariables); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
		}
		return utils.SendSuccessResponse(c, "whatsapp template message sent", nil)
	}
}

func SendWhatsAppTwilioMessage(svc *service.WhatsAppService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.SendWhatsAppRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		if err := svc.SendTwilio(c.UserContext(), req.PhoneNumber, req.Message); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
		}
		return utils.SendSuccessResponse(c, "whatsapp twilio message sent", nil)
	}
}

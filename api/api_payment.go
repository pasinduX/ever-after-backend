package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/integrations"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
	"github.com/stripe/stripe-go/v82/webhook"
)

func Checkout(svc *service.PaymentService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.CheckoutRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		resp, err := svc.CreateCheckout(c.UserContext(), getUserID(c), req)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
		}
		return utils.SendJSON(c, fiber.StatusOK, resp)
	}
}

func StripeWebhook(svc *service.PaymentService, cfg *integrations.Secrets) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := c.Body()
		event, err := webhook.ConstructEvent(body, c.Get("Stripe-Signature"), cfg.StripeWebhookSecret)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid stripe signature")
		}
		if err := svc.HandleWebhook(c.UserContext(), event); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "webhook processing failed")
		}
		return c.SendStatus(fiber.StatusOK)
	}
}

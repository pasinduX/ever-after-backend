package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

func SignUp(svc *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.SignUpRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		if err := svc.SignUp(c.UserContext(), req); err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendSuccessResponse(c, "verification code sent to email", nil)
	}
}

func VerifyEmail(svc *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.VerifyEmailRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		resp, err := svc.VerifyEmail(c.UserContext(), req.Email, req.Code)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, resp)
	}
}

func SignIn(svc *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.SignInRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		resp, err := svc.SignIn(c.UserContext(), req)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, resp)
	}
}

func Refresh(svc *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.RefreshRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		resp, err := svc.RefreshTokens(c.UserContext(), req.RefreshToken)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, resp)
	}
}

func SignOut(svc *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.RefreshRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		if err := svc.SignOut(c.UserContext(), req.RefreshToken); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to sign out")
		}
		return utils.SendSuccessResponse(c, "signed out successfully", nil)
	}
}

func Me(svc *service.AuthService, getUID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := svc.Me(c.UserContext(), getUID(c))
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to fetch user")
		}
		return utils.SendJSON(c, fiber.StatusOK, user)
	}
}

package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	appErrors "github.com/storyvows/backend/errors"
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
			if errors.Is(err, appErrors.ErrEmailTaken) {
				return utils.SendErrorResponse(c, fiber.StatusConflict, "email already in use")
			}
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to create account")
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
			if errors.Is(err, appErrors.ErrInvalidVerificationCode) {
				return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid or expired verification code")
			}
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to verify email")
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
			if errors.Is(err, appErrors.ErrInvalidCreds) {
				return utils.SendErrorResponse(c, fiber.StatusUnauthorized, "invalid email or password")
			}
			if errors.Is(err, appErrors.ErrEmailNotVerified) {
				return utils.SendErrorResponse(c, fiber.StatusUnauthorized, "email not verified")
			}
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to sign in")
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
			if errors.Is(err, appErrors.ErrInvalidToken) {
				return utils.SendErrorResponse(c, fiber.StatusUnauthorized, "invalid or expired refresh token")
			}
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to refresh tokens")
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

package utils

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	apperrors "github.com/storyvows/backend/errors"
)

func SendJSON(c *fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(data)
}

func SendSuccessResponse(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{Message: message, Data: data})
}

func SendErrorResponse(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(dto.ErrorResponse{Error: message})
}

func SendServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrWeddingNotFound):
		return SendErrorResponse(c, fiber.StatusNotFound, "wedding not found")
	case errors.Is(err, apperrors.ErrForbidden):
		return SendErrorResponse(c, fiber.StatusForbidden, "access denied")
	case errors.Is(err, apperrors.ErrInvalidFile):
		return SendErrorResponse(c, fiber.StatusUnsupportedMediaType, "invalid file type")
	case errors.Is(err, apperrors.ErrLimitReached):
		return SendErrorResponse(c, fiber.StatusPaymentRequired, err.Error())
	case errors.Is(err, apperrors.ErrEmailTaken):
		return SendErrorResponse(c, fiber.StatusConflict, "email already in use")
	case errors.Is(err, apperrors.ErrInvalidCreds):
		return SendErrorResponse(c, fiber.StatusUnauthorized, "invalid email or password")
	case errors.Is(err, apperrors.ErrEmailNotVerified):
		return SendErrorResponse(c, fiber.StatusUnauthorized, "email not verified")
	case errors.Is(err, apperrors.ErrInvalidToken):
		return SendErrorResponse(c, fiber.StatusUnauthorized, "invalid or expired token")
	case errors.Is(err, apperrors.ErrInvalidVerificationCode):
		return SendErrorResponse(c, fiber.StatusBadRequest, "invalid or expired verification code")
	case errors.Is(err, apperrors.ErrUploadNotFound):
		return SendErrorResponse(c, fiber.StatusNotFound, "upload not found")
	case errors.Is(err, apperrors.ErrInvalidTier):
		return SendErrorResponse(c, fiber.StatusBadRequest, "invalid tier")
	default:
		slog.Error("service error", "error", err.Error())
		return SendErrorResponse(c, fiber.StatusInternalServerError, "internal server error")
	}
}

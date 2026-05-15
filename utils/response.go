package utils

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
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

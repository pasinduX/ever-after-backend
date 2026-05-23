package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	apperrors "github.com/storyvows/backend/errors"
	"github.com/storyvows/backend/realtime"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

func GuestUpload(svc *service.UploadService, hub *realtime.Hub, maxSize int64) fiber.Handler {
	return func(c *fiber.Ctx) error {
		identifier := c.Params("id")
		if identifier == "" {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "identifier required")
		}
		guestName := c.FormValue("guest_name")

		fileHeader, err := c.FormFile("file")
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "file field required")
		}

		file, err := fileHeader.Open()
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to open file")
		}
		defer file.Close()

		upload, err := svc.GuestUploadByIdentifier(c.UserContext(), identifier, file, fileHeader, guestName)
		if errors.Is(err, apperrors.ErrLimitReached) {
			return utils.SendErrorResponse(c, fiber.StatusPaymentRequired, err.Error())
		}
		if errors.Is(err, apperrors.ErrInvalidFile) {
			return utils.SendErrorResponse(c, fiber.StatusUnsupportedMediaType, err.Error())
		}
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "upload failed")
		}

		hub.Broadcast(upload.WeddingID, upload)
		return utils.SendJSON(c, fiber.StatusCreated, upload)
	}
}

func UploadToFolder(svc *service.UploadService, hub *realtime.Hub, maxSize int64) fiber.Handler {
	return func(c *fiber.Ctx) error {
		weddingID := c.FormValue("wedding_id")
		if weddingID == "" {
			weddingID = c.FormValue("id")
		}
		if weddingID == "" {
			weddingID = c.Params("id")
		}
		if weddingID == "" {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "wedding_id required")
		}

		fileHeader, err := c.FormFile("file")
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "file field required")
		}

		file, err := fileHeader.Open()
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to open file")
		}
		defer file.Close()

		guestName := c.FormValue("guest_name")
		upload, err := svc.GuestUpload(c.UserContext(), weddingID, file, fileHeader, guestName)
		if errors.Is(err, apperrors.ErrLimitReached) {
			return utils.SendErrorResponse(c, fiber.StatusPaymentRequired, err.Error())
		}
		if errors.Is(err, apperrors.ErrInvalidFile) {
			return utils.SendErrorResponse(c, fiber.StatusUnsupportedMediaType, err.Error())
		}
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "upload failed")
		}

		if hub != nil {
			hub.Broadcast(upload.WeddingID, upload)
		}
		return utils.SendJSON(c, fiber.StatusCreated, upload)
	}
}

func ListUploads(svc *service.UploadService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		uploads, err := svc.ListForWedding(c.UserContext(), c.Params("id"))
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to list uploads")
		}
		return utils.SendJSON(c, fiber.StatusOK, uploads)
	}
}

func ApproveUpload(svc *service.UploadService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.ApproveUploadRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		if err := svc.SetApproval(c.UserContext(), c.Params("uploadId"), req.Approved); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to update approval")
		}
		return utils.SendSuccessResponse(c, "updated", nil)
	}
}

func DeleteUpload(svc *service.UploadService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := svc.Delete(c.UserContext(), c.Params("uploadId")); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "delete failed")
		}
		return utils.SendSuccessResponse(c, "deleted", nil)
	}
}

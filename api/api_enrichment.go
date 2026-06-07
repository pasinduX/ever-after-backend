package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/service"
)

type EnrichmentHandler struct {
	enrichSvc  *service.EnrichmentService
	payloadSvc *service.PayloadService
}

func NewEnrichmentHandler(enrichSvc *service.EnrichmentService, payloadSvc *service.PayloadService) *EnrichmentHandler {
	return &EnrichmentHandler{enrichSvc: enrichSvc, payloadSvc: payloadSvc}
}

// POST /:id/enrich — kick off post-wedding deep analysis
func (h *EnrichmentHandler) StartEnrichment(c *fiber.Ctx) error {
	weddingID := c.Params("id")
	if weddingID == "" {
		return fiber.ErrBadRequest
	}
	h.enrichSvc.Enqueue(weddingID)
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message": "enrichment started",
		"wedding_id": weddingID,
	})
}

// GET /:id/enrich/status
func (h *EnrichmentHandler) GetEnrichmentStatus(c *fiber.Ctx) error {
	weddingID := c.Params("id")
	cfg, err := h.enrichSvc.GetStatus(c.Context(), weddingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(cfg)
}

// GET /:id/album/payload
func (h *EnrichmentHandler) GetAlbumPayload(c *fiber.Ctx) error {
	weddingID := c.Params("id")
	payload, err := h.payloadSvc.BuildPayload(c.Context(), weddingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(payload)
}

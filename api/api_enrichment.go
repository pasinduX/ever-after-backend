package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dao"
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

	// Return the stored payload if available — no recomputation needed.
	if stored, err := dao.FindAlbumPayload(c.Context(), h.payloadSvc.DB(), weddingID); err == nil {
		return c.JSON(stored)
	}

	// Fallback: build on-demand (acts exist but payload wasn't stored yet).
	payload, err := h.payloadSvc.BuildPayload(c.Context(), weddingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// Store it so next request is instant.
	_ = dao.UpsertAlbumPayload(c.Context(), h.payloadSvc.DB(), payload)
	return c.JSON(payload)
}

package api

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/service"
)

type EnrichmentHandler struct {
	enrichSvc  *service.EnrichmentService
	payloadSvc *service.PayloadService
	uploadSvc  *service.UploadService
}

func NewEnrichmentHandler(enrichSvc *service.EnrichmentService, payloadSvc *service.PayloadService, uploadSvc *service.UploadService) *EnrichmentHandler {
	return &EnrichmentHandler{enrichSvc: enrichSvc, payloadSvc: payloadSvc, uploadSvc: uploadSvc}
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

	var payload *dto.AlbumPayload
	if stored, err := dao.FindAlbumPayload(c.Context(), h.payloadSvc.DB(), weddingID); err == nil {
		payload = stored
	} else {
		// Fallback: build on-demand (acts exist but payload wasn't stored yet).
		built, err := h.payloadSvc.BuildPayload(c.Context(), weddingID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = dao.UpsertAlbumPayload(c.Context(), h.payloadSvc.DB(), built)
		payload = built
	}

	// Re-sign all asset URLs so the client never receives an expired presigned URL,
	// regardless of when the payload was originally generated and stored.
	h.refreshPayloadURLs(c.Context(), payload)

	return c.JSON(payload)
}

// refreshPayloadURLs replaces every AssetObject URL in the payload with a
// freshly signed URL generated from the upload's stored S3 file key.
func (h *EnrichmentHandler) refreshPayloadURLs(ctx context.Context, payload *dto.AlbumPayload) {
	if h.uploadSvc == nil {
		return
	}
	uploads, err := dao.FindApprovedUploadsByWedding(ctx, h.payloadSvc.DB(), payload.AlbumID)
	if err != nil {
		return
	}
	fresh := make(map[string]string, len(uploads))
	for _, u := range uploads {
		if u.Storage.FileKey != "" {
			fresh[u.ID] = h.uploadSvc.FreshURL(u.Storage.FileKey)
		}
	}

	apply := func(a *dto.AssetObject) {
		if a == nil || a.ID == "" {
			return
		}
		if url, ok := fresh[a.ID]; ok && url != "" {
			a.URL = url
		}
	}

	apply(payload.Cover.Image)
	for ci := range payload.Chapters {
		for bi := range payload.Chapters[ci].Blocks {
			b := &payload.Chapters[ci].Blocks[bi]
			apply(b.Image)
			apply(b.BackgroundImage)
			apply(b.LeftImage)
			apply(b.FallbackImage)
			for ii := range b.Images {
				apply(&b.Images[ii])
			}
		}
	}
}

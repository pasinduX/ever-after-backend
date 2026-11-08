package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

// RerunAnalysis queues a wedding's photos for album analysis. Photos from
// before the current pipeline also get upright derivatives and a measured
// palette. Pass ?all=true to re-analyse every photo, not just outdated or
// failed ones.
func RerunAnalysis(svc *service.AnalysisService, weddings *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wedding, err := weddings.Get(c.UserContext(), c.Params("id"), getUserID(c))
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		queued, err := svc.RerunWedding(c.UserContext(), wedding.ID, c.QueryBool("all", false))
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusAccepted, fiber.Map{"wedding_id": wedding.ID, "queued": queued})
	}
}

// AnalysisStatus reports a wedding's photo analysis progress.
func AnalysisStatus(svc *service.AnalysisService, weddings *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wedding, err := weddings.Get(c.UserContext(), c.Params("id"), getUserID(c))
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		counts, err := svc.WeddingStatus(c.UserContext(), wedding.ID)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, counts)
	}
}

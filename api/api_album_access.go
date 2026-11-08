package api

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dto"
	apperrors "github.com/storyvows/backend/errors"
	"github.com/storyvows/backend/functions"
	"github.com/storyvows/backend/service"
	"github.com/storyvows/backend/utils"
)

const (
	weddingLocal = "wedding"
	// ownerLocal is true when the signed-in couple opened their own album.
	ownerLocal = "wedding_owner"
)

// requestedWedding returns the wedding a RequireWeddingOwner or
// RequireAlbumAccess middleware resolved for this request.
func requestedWedding(c *fiber.Ctx) *dto.Wedding {
	w, _ := c.Locals(weddingLocal).(*dto.Wedding)
	return w
}

// RequireWeddingOwner admits only the couple who own the wedding in :id.
// Routes under /api/weddings/:id that don't check ownership themselves use it.
func RequireWeddingOwner(weddings *service.WeddingService, getUserID func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wedding, err := weddings.Get(c.UserContext(), c.Params("id"), getUserID(c))
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		c.Locals(weddingLocal, wedding)
		return c.Next()
	}
}

// RequireAlbumAccess guards the guest album routes (/api/w/:id/...), where
// :id is a wedding ID or QR slug. It applies the wedding's privacy: public
// albums are open, password-protected ones need the guest token from
// POST /api/w/:id/access, private ones are for the signed-in couple only.
func RequireAlbumAccess(weddings *service.WeddingService, jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := functions.OptionalUserID(c, jwtSecret)
		wedding, err := weddings.OpenAlbum(c.UserContext(), c.Params("id"), userID, c.Get(functions.AlbumAccessHeader))
		if errors.Is(err, apperrors.ErrAlbumPasswordRequired) {
			// Enough for the password screen to say whose album it is.
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "password_required",
				"message": err.Error(),
				"wedding": fiber.Map{"couple_names": wedding.CoupleNames, "template": wedding.Template},
			})
		}
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		c.Locals(weddingLocal, wedding)
		c.Locals(ownerLocal, userID != "" && userID == wedding.OwnerID)
		return c.Next()
	}
}

// GuestWedding returns the wedding behind an album link: everything for the
// couple, and only what the album shows for anyone else (never the owner's
// account, phone number or address).
func GuestWedding() fiber.Handler {
	return func(c *fiber.Ctx) error {
		wedding := requestedWedding(c)
		if owner, _ := c.Locals(ownerLocal).(bool); owner {
			return utils.SendJSON(c, fiber.StatusOK, wedding)
		}
		return utils.SendJSON(c, fiber.StatusOK, publicWedding(wedding))
	}
}

// publicWedding is what guests may see of a wedding.
func publicWedding(w *dto.Wedding) fiber.Map {
	return fiber.Map{
		"id":              w.ID,
		"qr_slug":         w.QRSlug,
		"couple_names":    w.CoupleNames,
		"wedding_date":    w.WeddingDate,
		"wedding_time":    w.WeddingTime,
		"venue":           w.Venue,
		"welcome_message": w.WelcomeMessage,
		"template":        w.Template,
		"tier":            w.Tier,
		"privacy":         w.Privacy,
	}
}

// GuestUploadInfo tells the guest upload page whose wedding it is. Unlike
// the album routes it ignores album privacy: guests at a private wedding
// still share their photos, they just can't browse the album.
func GuestUploadInfo(weddings *service.WeddingService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wedding, err := weddings.FindForGuestUpload(c.UserContext(), c.Params("id"))
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, publicWedding(wedding))
	}
}

// GrantAlbumAccess exchanges the album password for a guest token, sent back
// on later requests in the X-Album-Access header.
func GrantAlbumAccess(weddings *service.WeddingService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req dto.GuestAccessRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid request body")
		}
		wedding, token, expires, err := weddings.GrantAlbumAccess(c.UserContext(), c.Params("id"), req.Password)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, fiber.Map{
			"wedding": wedding, "access_token": token, "expires_at": expires,
		})
	}
}

// GuestAlbumPage returns the AI-designed album page guests see, or
// {"page": null} while there is none (the app then shows the block album).
func GuestAlbumPage(albums *service.AlbumService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, err := albums.ShownPage(c.UserContext(), requestedWedding(c).ID)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, fiber.Map{"page": page})
	}
}

// GuestAlbumPhotos returns the photos the album page may show, with fresh links.
func GuestAlbumPhotos(uploads *service.UploadService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		photos, err := uploads.AlbumPhotos(c.UserContext(), requestedWedding(c).ID)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		c.Set("Cache-Control", "private, no-store")
		return utils.SendJSON(c, fiber.StatusOK, photos)
	}
}

// ListAlbumPageVersions lists every design version, newest first.
func ListAlbumPageVersions(albums *service.AlbumService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		versions, err := albums.PageVersions(c.UserContext(), requestedWedding(c).ID)
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendJSON(c, fiber.StatusOK, versions)
	}
}

// ActivateAlbumPageVersion makes an earlier ready design the one guests see.
func ActivateAlbumPageVersion(albums *service.AlbumService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		version, err := strconv.Atoi(c.Params("version"))
		if err != nil || version < 1 {
			return utils.SendErrorResponse(c, fiber.StatusBadRequest, "invalid version")
		}
		err = albums.ActivatePageVersion(c.UserContext(), requestedWedding(c).ID, version)
		if service.IsPageVersionNotReady(err) {
			return utils.SendErrorResponse(c, fiber.StatusNotFound, err.Error())
		}
		if err != nil {
			return utils.SendServiceError(c, err)
		}
		return utils.SendSuccessResponse(c, "album design restored", fiber.Map{"version": version})
	}
}

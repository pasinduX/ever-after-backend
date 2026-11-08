package apiHandlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/storyvows/backend/api"
	"github.com/storyvows/backend/functions"
	"github.com/storyvows/backend/integrations"
	"github.com/storyvows/backend/realtime"
	"github.com/storyvows/backend/service"
	"go.mongodb.org/mongo-driver/mongo"
)

func NewRouter(
	cfg *integrations.Secrets,
	db *mongo.Database,
	authSvc *service.AuthService,
	weddingSvc *service.WeddingService,
	guestSvc *service.GuestService,
	whatsappSvc *service.WhatsAppService,
	cardsSvc *service.CardsService,
	uploadSvc *service.UploadService,
	paymentSvc *service.PaymentService,
	albumSvc *service.AlbumService,
	analysisSvc *service.AnalysisService,
	payloadSvc *service.PayloadService,
	hub *realtime.Hub,
) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit:    int(cfg.MaxUploadSize),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	app.Use(recover.New())
	app.Use(functions.Logger)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.FrontendCORSOrigins,
		AllowMethods:     "GET,POST,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Accept,Authorization,Content-Type," + functions.AlbumAccessHeader,
		AllowCredentials: true,
		MaxAge:           300,
	}))
	app.Use(limiter.New(limiter.Config{
		Max:        200,
		Expiration: time.Minute,
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	getUID := functions.GetUserID
	// Guest album routes apply the wedding's privacy; the signed-in couple
	// always passes. This public lookup is registered before the owner route
	// below and so answers both guests and owners.
	albumAccess := api.RequireAlbumAccess(weddingSvc, cfg.JWTSecret)
	app.Get("/api/weddings/:id", albumAccess, api.GuestWedding())

	requireAuth := functions.RequireAuth(cfg.JWTSecret)
	app.Post("/api/whatsapp/send", requireAuth, api.SendWhatsAppMessage(whatsappSvc))
	app.Post("/api/whatsapp/send-twilio", requireAuth, api.SendWhatsAppTwilioMessage(whatsappSvc))
	app.Post("/api/whatsapp/send-template", requireAuth, api.SendWhatsAppTemplateMessage(whatsappSvc))

	auth := app.Group("/api/auth")
	auth.Post("/signup", api.SignUp(authSvc))
	auth.Post("/verify", api.VerifyEmail(authSvc))
	auth.Post("/signin", api.SignIn(authSvc))
	auth.Post("/forgot-password", api.ForgotPassword(authSvc))
	auth.Post("/reset-password", api.ResetPassword(authSvc))
	auth.Post("/refresh", api.Refresh(authSvc))
	auth.Post("/signout", requireAuth, api.SignOut(authSvc))
	auth.Get("/me", requireAuth, api.Me(authSvc, getUID))

	weddings := app.Group("/api/weddings", requireAuth)
	weddings.Post("/", api.CreateWedding(weddingSvc, cardsSvc, getUID))
	weddings.Get("/", api.ListWeddings(weddingSvc, getUID))
	weddings.Get("/:id", api.GetWedding(weddingSvc, getUID))
	weddings.Patch("/:id", api.UpdateWedding(weddingSvc, getUID))
	weddings.Delete("/:id", api.DeleteWedding(weddingSvc, getUID))
	weddings.Patch("/:id/privacy", api.SetPrivacyWedding(weddingSvc, getUID))
	weddings.Post("/:id/invite", api.CreateInvite(cardsSvc, getUID))
	weddings.Get("/:id/invite", api.GetInvite(cardsSvc, getUID))
	weddings.Patch("/:id/invite", api.UpdateInvite(cardsSvc, getUID))
	weddings.Delete("/:id/invite", api.DeleteInvite(cardsSvc, getUID))
	weddings.Post("/:id/thankyou", api.CreateThankYou(cardsSvc, getUID))
	weddings.Get("/:id/thankyou", api.GetThankYou(cardsSvc, getUID))
	weddings.Patch("/:id/thankyou", api.UpdateThankYou(cardsSvc, getUID))
	weddings.Delete("/:id/thankyou", api.DeleteThankYou(cardsSvc, getUID))
	weddings.Post("/:id/guests", api.CreateGuest(guestSvc, getUID))
	weddings.Get("/:id/guests", api.ListGuests(guestSvc, getUID))
	weddings.Get("/:id/guests/:guestId", api.GetGuest(guestSvc, getUID))
	weddings.Patch("/:id/guests/:guestId", api.UpdateGuest(guestSvc, getUID))
	weddings.Delete("/:id/guests/:guestId", api.DeleteGuest(guestSvc, getUID))
	// These handlers don't check ownership themselves.
	owner := api.RequireWeddingOwner(weddingSvc, getUID)
	weddings.Get("/:id/uploads", owner, api.ListUploads(uploadSvc))
	weddings.Patch("/:id/uploads/:uploadId/approve", owner, api.ApproveUpload(uploadSvc))
	weddings.Delete("/:id/uploads/:uploadId", owner, api.DeleteUpload(uploadSvc))
	weddings.Get("/:id/album", owner, api.Album(db, uploadSvc))
	weddings.Get("/:id/album/highlights", owner, api.Highlights(db, uploadSvc))
	weddings.Post("/:id/album/generate", owner, api.GenerateAlbum(albumSvc))
	weddings.Get("/:id/album/status", owner, api.GetAlbumStatus(albumSvc))
	weddings.Get("/:id/album/acts", owner, api.GetActs(albumSvc))
	weddings.Patch("/:id/album/acts", owner, api.UpdateActs(albumSvc))
	weddings.Patch("/:id/album/style", owner, api.SetAlbumStyle(albumSvc))

	payloadH := api.NewAlbumPayloadHandler(payloadSvc, uploadSvc)
	weddings.Get("/:id/album/payload", owner, payloadH.GetAlbumPayload)
	weddings.Get("/:id/album/photos", api.AlbumPhotos(weddingSvc, uploadSvc, getUID))
	weddings.Get("/:id/album/page", api.GetAlbumPage(albumSvc, weddingSvc, getUID))
	weddings.Post("/:id/album/page/regenerate", api.RegenerateAlbumPage(albumSvc, weddingSvc, getUID))
	weddings.Get("/:id/album/page/versions", owner, api.ListAlbumPageVersions(albumSvc))
	weddings.Post("/:id/album/page/versions/:version/activate", owner, api.ActivateAlbumPageVersion(albumSvc))
	weddings.Post("/:id/analysis/rerun", api.RerunAnalysis(analysisSvc, weddingSvc, getUID))
	weddings.Get("/:id/analysis/status", api.AnalysisStatus(analysisSvc, weddingSvc, getUID))

	app.Post("/api/uploads", api.UploadToFolder(uploadSvc, hub, cfg.MaxUploadSize))
	weddings.Get("/:id/album/download", owner, api.Download(db, uploadSvc))
	weddings.Get("/:id/wall", owner, hub.ServeSSE)

	guest := app.Group("/api/w/:id")
	guest.Get("/", albumAccess, api.GuestWedding())
	guest.Post("/access", api.GrantAlbumAccess(weddingSvc))
	guest.Get("/upload-info", api.GuestUploadInfo(weddingSvc))
	guest.Post("/uploads", api.GuestUpload(uploadSvc, hub, cfg.MaxUploadSize))
	guest.Get("/album", albumAccess, api.Album(db, uploadSvc))
	guest.Get("/album/highlights", albumAccess, api.Highlights(db, uploadSvc))
	guest.Get("/album/page", albumAccess, api.GuestAlbumPage(albumSvc))
	guest.Get("/album/photos", albumAccess, api.GuestAlbumPhotos(uploadSvc))

	app.Post("/api/checkout", requireAuth, api.Checkout(paymentSvc, getUID))
	app.Post("/api/webhooks/stripe", api.StripeWebhook(paymentSvc, cfg))

	return app
}

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
	uploadSvc *service.UploadService,
	paymentSvc *service.PaymentService,
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
		AllowHeaders:     "Accept,Authorization,Content-Type",
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

	requireAuth := functions.RequireAuth(cfg.JWTSecret)
	getUID := functions.GetUserID

	auth := app.Group("/api/auth")
	auth.Post("/signup", api.SignUp(authSvc))
	auth.Post("/verify", api.VerifyEmail(authSvc))
	auth.Post("/signin", api.SignIn(authSvc))
	auth.Post("/refresh", api.Refresh(authSvc))
	auth.Post("/signout", requireAuth, api.SignOut(authSvc))
	auth.Get("/me", requireAuth, api.Me(authSvc, getUID))

	weddings := app.Group("/api/weddings", requireAuth)
	weddings.Post("/", api.CreateWedding(weddingSvc, getUID))
	weddings.Get("/", api.ListWeddings(weddingSvc, getUID))
	weddings.Get("/:id", api.GetWedding(weddingSvc, getUID))
	weddings.Patch("/:id", api.UpdateWedding(weddingSvc, getUID))
	weddings.Delete("/:id", api.DeleteWedding(weddingSvc, getUID))
	weddings.Patch("/:id/privacy", api.SetPrivacyWedding(weddingSvc, getUID))
	weddings.Get("/:id/uploads", api.ListUploads(uploadSvc))
	weddings.Patch("/:id/uploads/:uploadId/approve", api.ApproveUpload(uploadSvc))
	weddings.Delete("/:id/uploads/:uploadId", api.DeleteUpload(uploadSvc))
	weddings.Get("/:id/album", api.Album(db))
	weddings.Get("/:id/album/highlights", api.Highlights(db))
	weddings.Get("/:id/album/download", api.Download(db))
	weddings.Get("/:id/wall", hub.ServeSSE)

	guest := app.Group("/api/w/:slug")
	guest.Get("/", api.GuestViewWedding(weddingSvc))
	guest.Post("/access", api.GuestAccessWedding(weddingSvc))
	guest.Post("/uploads", api.GuestUpload(uploadSvc, hub, cfg.MaxUploadSize))

	app.Post("/api/checkout", requireAuth, api.Checkout(paymentSvc, getUID))
	app.Post("/api/webhooks/stripe", api.StripeWebhook(paymentSvc, cfg))

	return app
}

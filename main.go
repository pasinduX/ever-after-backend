package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/storyvows/backend/apiHandlers"
	"github.com/storyvows/backend/dbConfig"
	"github.com/storyvows/backend/integrations"
	"github.com/storyvows/backend/realtime"
	"github.com/storyvows/backend/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := integrations.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, db, err := dbConfig.Connect(ctx, cfg.MongoURL, cfg.MongoDBName)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	authSvc := service.NewAuthService(db, cfg)
	weddingSvc := service.NewWeddingService(db, cfg)
	guestSvc := service.NewGuestService(db)
	whatsappSvc := service.NewWhatsAppService(cfg)
	s3Client, err := service.NewS3Client(cfg)
	if err != nil {
		slog.Error("failed to init s3 client", "error", err)
		os.Exit(1)
	}
	analysisSvc, err := service.NewAnalysisService(db, cfg, s3Client)
	if err != nil {
		slog.Error("failed to init analysis service", "error", err)
		os.Exit(1)
	}
	analysisSvc.Start()
	cardsSvc := service.NewCardsService(db, analysisSvc)
	uploadSvc, err := service.NewUploadService(db, cfg, s3Client, analysisSvc)
	if err != nil {
		slog.Error("failed to init upload service", "error", err)
		os.Exit(1)
	}
	paymentSvc := service.NewPaymentService(db, cfg)
	hub := realtime.NewHub()

	app := apiHandlers.NewRouter(cfg, db, authSvc, weddingSvc, guestSvc, whatsappSvc, cardsSvc, uploadSvc, paymentSvc, hub)

	slog.Info("starting server", "port", cfg.Port, "env", cfg.Env)
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down...")

	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("shutdown complete")
}

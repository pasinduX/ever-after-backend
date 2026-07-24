package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/integrations"
	"go.mongodb.org/mongo-driver/mongo"
)

const enrichWorkers = 20
const enrichTimeout = 5 * time.Minute

type EnrichmentService struct {
	db      *mongo.Database
	cfg     *integrations.Secrets
	analyis *AnalysisService
	jobs    chan string
}

func NewEnrichmentService(db *mongo.Database, cfg *integrations.Secrets, analysis *AnalysisService) *EnrichmentService {
	return &EnrichmentService{
		db:      db,
		cfg:     cfg,
		analyis: analysis,
		jobs:    make(chan string, 500),
	}
}

func (e *EnrichmentService) Start() {
	for i := 0; i < enrichWorkers; i++ {
		go e.worker()
	}
}

func (e *EnrichmentService) Enqueue(weddingID string) {
	select {
	case e.jobs <- weddingID:
	default:
		slog.Warn("enrichment queue full, dropping job", "wedding_id", weddingID)
	}
}

func (e *EnrichmentService) worker() {
	for weddingID := range e.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), enrichTimeout)
		if err := e.enrichWedding(ctx, weddingID); err != nil {
			slog.Error("enrichment failed", "wedding_id", weddingID, "error", err)
		}
		cancel()
	}
}

func (e *EnrichmentService) GetStatus(ctx context.Context, weddingID string) (*dto.EnrichmentConfig, error) {
	cfg, err := dao.FindEnrichmentConfig(ctx, e.db, weddingID)
	if err == dao.ErrNoRows {
		return &dto.EnrichmentConfig{
			WeddingID: weddingID,
			Status:    dto.EnrichmentStatusPending,
		}, nil
	}
	return cfg, err
}

func (e *EnrichmentService) enrichWedding(ctx context.Context, weddingID string) error {
	uploads, err := dao.FindUnenrichedUploadsByWedding(ctx, e.db, weddingID)
	if err != nil {
		return err
	}

	total := len(uploads)
	cfg := &dto.EnrichmentConfig{
		WeddingID:   weddingID,
		Status:      dto.EnrichmentStatusProcessing,
		TotalPhotos: total,
		Enriched:    0,
	}
	if err := dao.UpsertEnrichmentConfig(ctx, e.db, cfg); err != nil {
		return err
	}

	var mu sync.Mutex
	enriched := 0
	sem := make(chan struct{}, enrichWorkers)

	var wg sync.WaitGroup
	for _, u := range uploads {
		wg.Add(1)
		sem <- struct{}{}
		go func(upload *dto.Upload) {
			defer wg.Done()
			defer func() { <-sem }()

			enrichment, err := e.analyzePhoto(ctx, upload)
			if err != nil {
				slog.Warn("enrich photo failed", "upload_id", upload.ID, "error", err)
				return
			}
			if err := dao.UpdateUploadEnrichment(ctx, e.db, upload.ID, enrichment); err != nil {
				slog.Warn("save enrichment failed", "upload_id", upload.ID, "error", err)
				return
			}
			mu.Lock()
			enriched++
			mu.Unlock()
		}(u)
	}
	wg.Wait()

	cfg.Status = dto.EnrichmentStatusCompleted
	cfg.Enriched = enriched
	return dao.UpsertEnrichmentConfig(ctx, e.db, cfg)
}

func (e *EnrichmentService) analyzePhoto(ctx context.Context, upload *dto.Upload) (dto.UploadEnrichment, error) {
	prompt := `Deeply analyze this wedding photo. Return only valid JSON:
{"story_phase":"ceremony","moment":"ring_exchange","emotion_score":0.92,"hero_score":88,"studio_probability":0.85}
story_phase: anticipation|ceremony|family_bonds|just_the_two|celebration|final_dance
moment: short descriptor of the specific moment (e.g. first_kiss, ring_exchange, first_dance, group_laugh)
emotion_score: 0.0-1.0 (1.0=peak emotional intensity)
hero_score: 0-100 (suitability as a hero/featured photo)
studio_probability: 0.0-1.0 (likelihood this is a professional studio-quality shot)`

	jsonText, err := e.analyis.callOpenAIForJSON(ctx, prompt, fmt.Sprintf("Image URL: %s", upload.FileURL))
	if err != nil {
		return dto.UploadEnrichment{}, err
	}

	var parsed struct {
		StoryPhase        string   `json:"story_phase"`
		Moment            string   `json:"moment"`
		EmotionScore      *float64 `json:"emotion_score"`
		HeroScore         *int     `json:"hero_score"`
		StudioProbability *float64 `json:"studio_probability"`
	}
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		return dto.UploadEnrichment{}, fmt.Errorf("parse enrichment JSON: %w; response=%s", err, jsonText)
	}

	now := time.Now()
	return dto.UploadEnrichment{
		StoryPhase:        parsed.StoryPhase,
		Moment:            parsed.Moment,
		EmotionScore:      parsed.EmotionScore,
		HeroScore:         parsed.HeroScore,
		StudioProbability: parsed.StudioProbability,
		EnrichedAt:        &now,
	}, nil
}

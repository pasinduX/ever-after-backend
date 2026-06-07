package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpsertEnrichmentConfig(ctx context.Context, db *mongo.Database, cfg *dto.EnrichmentConfig) error {
	cfg.UpdatedAt = time.Now()
	filter := bson.M{"wedding_id": cfg.WeddingID}
	update := bson.M{"$set": cfg}
	opts := options.Update().SetUpsert(true)
	_, err := db.Collection("enrichment_configs").UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("dao UpsertEnrichmentConfig: %w", err)
	}
	return nil
}

func FindEnrichmentConfig(ctx context.Context, db *mongo.Database, weddingID string) (*dto.EnrichmentConfig, error) {
	var cfg dto.EnrichmentConfig
	err := db.Collection("enrichment_configs").FindOne(ctx, bson.M{"wedding_id": weddingID}).Decode(&cfg)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("dao FindEnrichmentConfig: %w", err)
	}
	return &cfg, nil
}

func FindUnenrichedUploadsByWedding(ctx context.Context, db *mongo.Database, weddingID string) ([]*dto.Upload, error) {
	filter := bson.M{
		"wedding_id": weddingID,
		"is_approved": true,
		"file_type":   "photo",
		"enrichment.enriched_at": bson.M{"$exists": false},
	}
	cursor, err := db.Collection("uploads").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("dao FindUnenrichedUploadsByWedding: %w", err)
	}
	defer cursor.Close(ctx)

	var uploads []*dto.Upload
	if err := cursor.All(ctx, &uploads); err != nil {
		return nil, fmt.Errorf("dao FindUnenrichedUploadsByWedding decode: %w", err)
	}
	return uploads, nil
}

func UpdateUploadEnrichment(ctx context.Context, db *mongo.Database, uploadID string, e dto.UploadEnrichment) error {
	now := time.Now()
	e.EnrichedAt = &now
	_, err := db.Collection("uploads").UpdateOne(ctx,
		bson.M{"_id": uploadID},
		bson.M{"$set": bson.M{"enrichment": e}},
	)
	if err != nil {
		return fmt.Errorf("dao UpdateUploadEnrichment: %w", err)
	}
	return nil
}

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

func UpsertAlbumConfig(ctx context.Context, db *mongo.Database, cfg *dto.AlbumConfig) error {
	cfg.UpdatedAt = time.Now()
	_, err := db.Collection("album_configs").UpdateOne(
		ctx,
		bson.M{"wedding_id": cfg.WeddingID},
		bson.M{"$set": cfg},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("dao UpsertAlbumConfig: %w", err)
	}
	return nil
}

func FindAlbumConfig(ctx context.Context, db *mongo.Database, weddingID string) (*dto.AlbumConfig, error) {
	var cfg dto.AlbumConfig
	err := db.Collection("album_configs").FindOne(ctx, bson.M{"wedding_id": weddingID}).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("dao FindAlbumConfig: %w", err)
	}
	return &cfg, nil
}

func UpsertStoryActs(ctx context.Context, db *mongo.Database, weddingID string, acts []*dto.StoryAct) error {
	if _, err := db.Collection("story_acts").DeleteMany(ctx, bson.M{"wedding_id": weddingID}); err != nil {
		return fmt.Errorf("dao UpsertStoryActs delete: %w", err)
	}
	if len(acts) == 0 {
		return nil
	}
	docs := make([]any, len(acts))
	for i, a := range acts {
		docs[i] = a
	}
	if _, err := db.Collection("story_acts").InsertMany(ctx, docs); err != nil {
		return fmt.Errorf("dao UpsertStoryActs insert: %w", err)
	}
	return nil
}

func FindStoryActs(ctx context.Context, db *mongo.Database, weddingID string) ([]*dto.StoryAct, error) {
	opts := options.Find().SetSort(bson.D{{Key: "order", Value: 1}})
	cursor, err := db.Collection("story_acts").Find(ctx, bson.M{"wedding_id": weddingID}, opts)
	if err != nil {
		return nil, fmt.Errorf("dao FindStoryActs: %w", err)
	}
	defer cursor.Close(ctx)

	var acts []*dto.StoryAct
	if err := cursor.All(ctx, &acts); err != nil {
		return nil, fmt.Errorf("dao FindStoryActs decode: %w", err)
	}
	return acts, nil
}

func UpdateStoryAct(ctx context.Context, db *mongo.Database, actID string, photoIDs []string, confirmed bool) error {
	_, err := db.Collection("story_acts").UpdateOne(
		ctx,
		bson.M{"_id": actID},
		bson.M{"$set": bson.M{
			"photo_ids": photoIDs,
			"confirmed": confirmed,
			"updated_at": time.Now(),
		}},
	)
	if err != nil {
		return fmt.Errorf("dao UpdateStoryAct: %w", err)
	}
	return nil
}

package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateInviteConfig(ctx context.Context, db *mongo.Database, cfg *dto.InviteConfig) error {
	_, err := db.Collection("invite_configs").InsertOne(ctx, cfg)
	if err != nil {
		return fmt.Errorf("dao CreateInviteConfig: %w", err)
	}
	return nil
}

func FindInviteConfigByWedding(ctx context.Context, db *mongo.Database, weddingID string) (*dto.InviteConfig, error) {
	var cfg dto.InviteConfig
	err := db.Collection("invite_configs").FindOne(ctx, bson.M{"wedding_id": weddingID}).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("dao FindInviteConfigByWedding: %w", err)
	}
	return &cfg, nil
}

func UpdateInviteConfig(ctx context.Context, db *mongo.Database, cfg *dto.InviteConfig) error {
	_, err := db.Collection("invite_configs").ReplaceOne(ctx, bson.M{"_id": cfg.ID}, cfg)
	if err != nil {
		return fmt.Errorf("dao UpdateInviteConfig: %w", err)
	}
	return nil
}

func DeleteInviteConfig(ctx context.Context, db *mongo.Database, weddingID string) error {
	_, err := db.Collection("invite_configs").DeleteOne(ctx, bson.M{"wedding_id": weddingID})
	if err != nil {
		return fmt.Errorf("dao DeleteInviteConfig: %w", err)
	}
	return nil
}

func CreateThankYouConfig(ctx context.Context, db *mongo.Database, cfg *dto.ThankYouConfig) error {
	_, err := db.Collection("thankyou_configs").InsertOne(ctx, cfg)
	if err != nil {
		return fmt.Errorf("dao CreateThankYouConfig: %w", err)
	}
	return nil
}

func FindThankYouConfigByWedding(ctx context.Context, db *mongo.Database, weddingID string) (*dto.ThankYouConfig, error) {
	var cfg dto.ThankYouConfig
	err := db.Collection("thankyou_configs").FindOne(ctx, bson.M{"wedding_id": weddingID}).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("dao FindThankYouConfigByWedding: %w", err)
	}
	return &cfg, nil
}

func UpdateThankYouConfig(ctx context.Context, db *mongo.Database, cfg *dto.ThankYouConfig) error {
	_, err := db.Collection("thankyou_configs").ReplaceOne(ctx, bson.M{"_id": cfg.ID}, cfg)
	if err != nil {
		return fmt.Errorf("dao UpdateThankYouConfig: %w", err)
	}
	return nil
}

func DeleteThankYouConfig(ctx context.Context, db *mongo.Database, weddingID string) error {
	_, err := db.Collection("thankyou_configs").DeleteOne(ctx, bson.M{"wedding_id": weddingID})
	if err != nil {
		return fmt.Errorf("dao DeleteThankYouConfig: %w", err)
	}
	return nil
}

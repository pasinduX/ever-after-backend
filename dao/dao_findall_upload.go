package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FindUploadsByWedding(ctx context.Context, db *mongo.Database, weddingID string) ([]*dto.Upload, error) {
	cursor, err := db.Collection("uploads").Find(ctx, bson.M{"wedding_id": weddingID})
	if err != nil {
		return nil, fmt.Errorf("dao FindUploadsByWedding: %w", err)
	}
	defer cursor.Close(ctx)

	var uploads []*dto.Upload
	if err := cursor.All(ctx, &uploads); err != nil {
		return nil, fmt.Errorf("dao FindUploadsByWedding decode: %w", err)
	}
	return uploads, nil
}

func FindApprovedUploadsByWedding(ctx context.Context, db *mongo.Database, weddingID string) ([]*dto.Upload, error) {
	cursor, err := db.Collection("uploads").Find(ctx, bson.M{"wedding_id": weddingID, "is_approved": true})
	if err != nil {
		return nil, fmt.Errorf("dao FindApprovedUploadsByWedding: %w", err)
	}
	defer cursor.Close(ctx)

	var uploads []*dto.Upload
	if err := cursor.All(ctx, &uploads); err != nil {
		return nil, fmt.Errorf("dao FindApprovedUploadsByWedding decode: %w", err)
	}
	return uploads, nil
}

func FindApprovedUploadsByWeddingSorted(ctx context.Context, db *mongo.Database, weddingID string) ([]*dto.Upload, error) {
	opts := options.Find().SetSort(bson.D{
		{Key: "timeline.captured_at", Value: 1},
		{Key: "uploaded_at", Value: 1},
	})
	cursor, err := db.Collection("uploads").Find(ctx,
		bson.M{"wedding_id": weddingID, "is_approved": true},
		opts,
	)
	if err != nil {
		return nil, fmt.Errorf("dao FindApprovedUploadsByWeddingSorted: %w", err)
	}
	defer cursor.Close(ctx)

	var uploads []*dto.Upload
	if err := cursor.All(ctx, &uploads); err != nil {
		return nil, fmt.Errorf("dao FindApprovedUploadsByWeddingSorted decode: %w", err)
	}
	return uploads, nil
}

func FindRandomPhotoHighlights(ctx context.Context, db *mongo.Database, weddingID string, limit int) ([]*dto.Upload, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"wedding_id": weddingID, "is_approved": true, "file_type": "photo"}}},
		{{"$sample", bson.M{"size": limit}}},
	}
	cursor, err := db.Collection("uploads").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("dao FindRandomPhotoHighlights: %w", err)
	}
	defer cursor.Close(ctx)

	var uploads []*dto.Upload
	if err := cursor.All(ctx, &uploads); err != nil {
		return nil, fmt.Errorf("dao FindRandomPhotoHighlights decode: %w", err)
	}
	return uploads, nil
}

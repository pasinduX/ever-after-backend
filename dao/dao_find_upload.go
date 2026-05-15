package dao

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
)

func FindUploadByID(ctx context.Context, db *mongo.Database, id string) (*dto.Upload, error) {
	var u dto.Upload
	err := db.Collection("uploads").FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if err != nil {
		return nil, fmt.Errorf("dao FindUploadByID: %w", err)
	}
	return &u, nil
}

func CountUploadsByWedding(ctx context.Context, db *mongo.Database, weddingID string) (int, error) {
	count, err := db.Collection("uploads").CountDocuments(ctx, bson.M{"wedding_id": weddingID})
	if err != nil {
		return 0, fmt.Errorf("dao CountUploadsByWedding: %w", err)
	}
	return int(count), nil
}

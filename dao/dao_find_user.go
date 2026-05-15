package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func FindUserByEmail(ctx context.Context, db *mongo.Database, email string) (*dto.User, error) {
	var user dto.User
	err := db.Collection("users").FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("dao FindUserByEmail: %w", err)
	}
	return &user, nil
}

func FindUserByID(ctx context.Context, db *mongo.Database, id string) (*dto.User, error) {
	var user dto.User
	err := db.Collection("users").FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("dao FindUserByID: %w", err)
	}
	return &user, nil
}

func CountUsersByEmail(ctx context.Context, db *mongo.Database, email string) (int, error) {
	count, err := db.Collection("users").CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		return 0, fmt.Errorf("dao CountUsersByEmail: %w", err)
	}
	return int(count), nil
}

var ErrNoRows = mongo.ErrNoDocuments

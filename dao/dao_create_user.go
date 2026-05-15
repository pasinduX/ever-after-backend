package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateUser(ctx context.Context, db *mongo.Database, user *dto.User) error {
	_, err := db.Collection("users").InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("dao CreateUser: %w", err)
	}
	return nil
}

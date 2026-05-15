package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func UpdateUser(ctx context.Context, db *mongo.Database, u *dto.User) error {
	u.UpdatedAt = time.Now()
	_, err := db.Collection("users").ReplaceOne(ctx, bson.M{"_id": u.ID}, u)
	if err != nil {
		return fmt.Errorf("dao UpdateUser: %w", err)
	}
	return nil
}

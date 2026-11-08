package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ClaimPasswordResetAttempt atomically spends one guess against the user's
// reset code and returns the user. The increment happens in the same write as
// the limit check, so parallel requests can't squeeze in extra guesses.
// Returns ErrNoRows when the user doesn't exist, has no pending reset, or has
// used up maxAttempts.
func ClaimPasswordResetAttempt(ctx context.Context, db *mongo.Database, email string, maxAttempts int) (*dto.User, error) {
	filter := bson.M{
		"email":                    email,
		"password_reset_code_hash": bson.M{"$exists": true, "$ne": ""},
		"$or": bson.A{
			bson.M{"password_reset_attempts": bson.M{"$exists": false}},
			bson.M{"password_reset_attempts": bson.M{"$lt": maxAttempts}},
		},
	}
	update := bson.M{"$inc": bson.M{"password_reset_attempts": 1}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var user dto.User
	if err := db.Collection("users").FindOneAndUpdate(ctx, filter, update, opts).Decode(&user); err != nil {
		return nil, fmt.Errorf("dao ClaimPasswordResetAttempt: %w", err)
	}
	return &user, nil
}

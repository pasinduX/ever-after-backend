package dao

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// weddingScopedCollections all key their documents on wedding_id, so removing a
// wedding means clearing each of them too. Without this the rows are orphaned:
// nothing else ever reads or cleans them up.
var weddingScopedCollections = []string{
	"uploads",
	"album_configs",
	"album_payloads",
	"story_acts",
	"invite_configs",
	"thankyou_configs",
	"enrichment_configs",
	"guests",
}

// DeleteWedding removes the wedding document and every record scoped to it.
//
// The wedding row goes last: if a cascade step fails partway, the wedding is
// still listed and the owner can retry, rather than the record vanishing and
// stranding its children permanently.
func DeleteWedding(ctx context.Context, db *mongo.Database, weddingID string) error {
	for _, collection := range weddingScopedCollections {
		if _, err := db.Collection(collection).DeleteMany(ctx, bson.M{"wedding_id": weddingID}); err != nil {
			return fmt.Errorf("dao DeleteWedding %s: %w", collection, err)
		}
	}

	res, err := db.Collection("weddings").DeleteOne(ctx, bson.M{"_id": weddingID})
	if err != nil {
		return fmt.Errorf("dao DeleteWedding: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrNoRows
	}
	return nil
}

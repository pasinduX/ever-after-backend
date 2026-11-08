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

// StartAlbumPage records a new "generating" version for a wedding and returns
// it. Versions count up from 1.
func StartAlbumPage(ctx context.Context, db *mongo.Database, id, weddingID string) (*dto.AlbumPage, error) {
	version := 1
	if latest, err := FindLatestAlbumPage(ctx, db, weddingID); err == nil {
		version = latest.Version + 1
	} else if err != ErrNoRows {
		return nil, err
	}
	now := time.Now()
	page := &dto.AlbumPage{
		ID: id, WeddingID: weddingID, Version: version, Status: dto.AlbumPageGenerating,
		CreatedAt: now, UpdatedAt: now,
	}
	if _, err := db.Collection("album_pages").InsertOne(ctx, page); err != nil {
		return nil, fmt.Errorf("dao StartAlbumPage: %w", err)
	}
	return page, nil
}

// SaveAlbumPage replaces a version with its finished state.
func SaveAlbumPage(ctx context.Context, db *mongo.Database, page *dto.AlbumPage) error {
	page.UpdatedAt = time.Now()
	if _, err := db.Collection("album_pages").ReplaceOne(ctx, bson.M{"_id": page.ID}, page); err != nil {
		return fmt.Errorf("dao SaveAlbumPage: %w", err)
	}
	return nil
}

// FindLatestAlbumPage returns the newest version, whatever its status.
func FindLatestAlbumPage(ctx context.Context, db *mongo.Database, weddingID string) (*dto.AlbumPage, error) {
	return findAlbumPage(ctx, db, bson.M{"wedding_id": weddingID})
}

// FindLatestReadyAlbumPage returns the newest version that can be shown.
func FindLatestReadyAlbumPage(ctx context.Context, db *mongo.Database, weddingID string) (*dto.AlbumPage, error) {
	return findAlbumPage(ctx, db, bson.M{"wedding_id": weddingID, "status": dto.AlbumPageReady})
}

func findAlbumPage(ctx context.Context, db *mongo.Database, filter bson.M) (*dto.AlbumPage, error) {
	var page dto.AlbumPage
	opts := options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}})
	if err := db.Collection("album_pages").FindOne(ctx, filter, opts).Decode(&page); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNoRows
		}
		return nil, fmt.Errorf("dao findAlbumPage: %w", err)
	}
	return &page, nil
}

// InterruptStaleAlbumPages marks versions left "generating" by a restart as
// fallbacks, so the app doesn't wait on them forever.
func InterruptStaleAlbumPages(ctx context.Context, db *mongo.Database, olderThan time.Time) error {
	_, err := db.Collection("album_pages").UpdateMany(ctx,
		bson.M{"status": dto.AlbumPageGenerating, "updated_at": bson.M{"$lt": olderThan}},
		bson.M{"$set": bson.M{"status": dto.AlbumPageFallback, "fallback_reason": dto.AlbumPageInterrupted, "updated_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("dao InterruptStaleAlbumPages: %w", err)
	}
	return nil
}

// ActivateAlbumPage makes one ready version the one guests see.
func ActivateAlbumPage(ctx context.Context, db *mongo.Database, weddingID string, version int) error {
	res, err := db.Collection("album_pages").UpdateOne(ctx,
		bson.M{"wedding_id": weddingID, "version": version, "status": dto.AlbumPageReady},
		bson.M{"$set": bson.M{"active": true}},
	)
	if err != nil {
		return fmt.Errorf("dao ActivateAlbumPage: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrNoRows
	}
	_, err = db.Collection("album_pages").UpdateMany(ctx,
		bson.M{"wedding_id": weddingID, "version": bson.M{"$ne": version}},
		bson.M{"$set": bson.M{"active": false}},
	)
	if err != nil {
		return fmt.Errorf("dao ActivateAlbumPage others: %w", err)
	}
	return nil
}

// FindActiveAlbumPage returns the version guests see, if one was activated.
func FindActiveAlbumPage(ctx context.Context, db *mongo.Database, weddingID string) (*dto.AlbumPage, error) {
	return findAlbumPage(ctx, db, bson.M{"wedding_id": weddingID, "status": dto.AlbumPageReady, "active": true})
}

// ListAlbumPages returns every version, newest first, without the documents.
func ListAlbumPages(ctx context.Context, db *mongo.Database, weddingID string) ([]*dto.AlbumPage, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "version", Value: -1}}).
		SetProjection(bson.M{"html": 0, "diagnostics": 0})
	cursor, err := db.Collection("album_pages").Find(ctx, bson.M{"wedding_id": weddingID}, opts)
	if err != nil {
		return nil, fmt.Errorf("dao ListAlbumPages: %w", err)
	}
	pages := []*dto.AlbumPage{}
	if err := cursor.All(ctx, &pages); err != nil {
		return nil, fmt.Errorf("dao ListAlbumPages decode: %w", err)
	}
	return pages, nil
}

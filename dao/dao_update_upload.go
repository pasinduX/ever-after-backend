package dao

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SetUploadApproval(ctx context.Context, db *mongo.Database, uploadID string, approved bool) error {
	_, err := db.Collection("uploads").UpdateOne(ctx, bson.M{"_id": uploadID}, bson.M{"$set": bson.M{"is_approved": approved}})
	if err != nil {
		return fmt.Errorf("dao SetUploadApproval: %w", err)
	}
	return nil
}

func SetUploadAnalysisStatus(ctx context.Context, db *mongo.Database, uploadID string, status dto.AnalysisStatus, errorMessage *string) error {
	update := bson.M{
		"analysis_status": status,
		"analysis.status": status,
	}
	if errorMessage != nil {
		update["analysis_error"] = *errorMessage
		update["analysis.error"] = *errorMessage
	} else {
		update["analysis_error"] = nil
		update["analysis.error"] = nil
	}
	_, err := db.Collection("uploads").UpdateOne(ctx, bson.M{"_id": uploadID}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("dao SetUploadAnalysisStatus: %w", err)
	}
	return nil
}

// UpdateUploadAnalysis stores a finished analysis. The whole analysis
// subdocument is replaced so no field the analyser produced is lost, and the
// legacy top-level copies older readers use are kept in step.
func UpdateUploadAnalysis(ctx context.Context, db *mongo.Database, uploadID string, upload *dto.Upload) error {
	update := bson.M{
		"analysis_status": upload.AnalysisStatus,
		"analysis":        upload.Analysis,
		"analysis_error":  upload.AnalysisError,
	}
	if upload.Category != "" {
		update["category"] = upload.Category
	}
	if upload.QualityScore != nil {
		update["quality_score"] = *upload.QualityScore
	}
	if upload.DetectedFaces != nil {
		update["detected_faces"] = *upload.DetectedFaces
	}
	if len(upload.SceneTags) > 0 {
		update["scene_tags"] = upload.SceneTags
	}
	if upload.AIInsights != nil {
		update["ai_insights"] = upload.AIInsights
	}
	_, err := db.Collection("uploads").UpdateOne(ctx, bson.M{"_id": uploadID}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("dao UpdateUploadAnalysis: %w", err)
	}
	return nil
}

// UpdateUploadMedia stores rebuilt derivatives and everything measured from
// the pixels, for uploads processed by an older image pipeline.
func UpdateUploadMedia(ctx context.Context, db *mongo.Database, uploadID string, upload *dto.Upload) error {
	update := bson.M{
		"storage.medium_url":        upload.Storage.MediumURL,
		"storage.thumbnail_url":     upload.Storage.ThumbnailURL,
		"metadata.pipeline_version": upload.Metadata.PipelineVersion,
		"exif.orientation":          upload.EXIF.Orientation,
	}
	set := func(key string, value any, ok bool) {
		if ok {
			update[key] = value
		}
	}
	set("metadata.width", upload.Metadata.Width, upload.Metadata.Width != nil)
	set("metadata.height", upload.Metadata.Height, upload.Metadata.Height != nil)
	set("metadata.aspect_ratio", upload.Metadata.AspectRatio, upload.Metadata.AspectRatio != nil)
	set("metadata.orientation", upload.Metadata.Orientation, upload.Metadata.Orientation != nil)
	set("orientation", upload.Metadata.Orientation, upload.Metadata.Orientation != nil)
	set("metadata.palette", upload.Metadata.Palette, len(upload.Metadata.Palette) > 0)
	set("metadata.tone", upload.Metadata.Tone, upload.Metadata.Tone != nil)
	set("phash", upload.PHash, upload.PHash != "")
	set("analysis.processing.thumbnail", upload.Analysis.Processing.Thumbnail, upload.Analysis.Processing.Thumbnail != "")
	_, err := db.Collection("uploads").UpdateOne(ctx, bson.M{"_id": uploadID}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("dao UpdateUploadMedia: %w", err)
	}
	return nil
}

// ResetUploadAnalysisForWedding marks a wedding's photos for re-analysis and
// returns their IDs. Unless all is set, only photos that failed or predate
// the current album analysis version are reset.
func ResetUploadAnalysisForWedding(ctx context.Context, db *mongo.Database, weddingID string, albumVersion int, all bool) ([]string, error) {
	filter := bson.M{"wedding_id": weddingID, "file_type": dto.FileTypePhoto}
	if !all {
		filter["$or"] = bson.A{
			bson.M{"analysis_status": dto.AnalysisStatusFailed},
			bson.M{"analysis.album.version": bson.M{"$not": bson.M{"$gte": albumVersion}}},
		}
	}
	cursor, err := db.Collection("uploads").Find(ctx, filter, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("dao ResetUploadAnalysisForWedding find: %w", err)
	}
	var rows []struct {
		ID string `bson:"_id"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("dao ResetUploadAnalysisForWedding decode: %w", err)
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	if len(ids) == 0 {
		return ids, nil
	}
	_, err = db.Collection("uploads").UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		bson.M{"$set": bson.M{"analysis_status": dto.AnalysisStatusPending, "analysis.status": dto.AnalysisStatusPending}},
	)
	if err != nil {
		return nil, fmt.Errorf("dao ResetUploadAnalysisForWedding update: %w", err)
	}
	return ids, nil
}

// AnalysisCounts summarises a wedding's photo analysis progress.
type AnalysisCounts struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Succeeded  int `json:"succeeded"`
	Failed     int `json:"failed"`
	// AlbumReady counts photos analysed at the current album version.
	AlbumReady int `json:"album_ready"`
}

func CountUploadAnalysisForWedding(ctx context.Context, db *mongo.Database, weddingID string, albumVersion int) (*AnalysisCounts, error) {
	filter := bson.M{"wedding_id": weddingID, "file_type": dto.FileTypePhoto}
	cursor, err := db.Collection("uploads").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{"_id": "$analysis_status", "n": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return nil, fmt.Errorf("dao CountUploadAnalysisForWedding: %w", err)
	}
	var groups []struct {
		Status dto.AnalysisStatus `bson:"_id"`
		N      int                `bson:"n"`
	}
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, fmt.Errorf("dao CountUploadAnalysisForWedding decode: %w", err)
	}
	counts := &AnalysisCounts{}
	for _, g := range groups {
		counts.Total += g.N
		switch g.Status {
		case dto.AnalysisStatusProcessing:
			counts.Processing += g.N
		case dto.AnalysisStatusSucceeded:
			counts.Succeeded += g.N
		case dto.AnalysisStatusFailed:
			counts.Failed += g.N
		default:
			counts.Pending += g.N
		}
	}
	ready, err := db.Collection("uploads").CountDocuments(ctx, bson.M{
		"wedding_id":             weddingID,
		"file_type":              dto.FileTypePhoto,
		"analysis.album.version": bson.M{"$gte": albumVersion},
	})
	if err != nil {
		return nil, fmt.Errorf("dao CountUploadAnalysisForWedding ready: %w", err)
	}
	counts.AlbumReady = int(ready)
	return counts, nil
}

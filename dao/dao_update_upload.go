package dao

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
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

func UpdateUploadAnalysis(ctx context.Context, db *mongo.Database, uploadID string, upload *dto.Upload) error {
	update := bson.M{
		"analysis_status": upload.AnalysisStatus,
		"analysis.status": upload.Analysis.Status,
	}
	if upload.Category != "" {
		update["category"] = upload.Category
		update["analysis.category"] = upload.Analysis.Category
	}
	if upload.QualityScore != nil {
		update["quality_score"] = *upload.QualityScore
		update["analysis.quality_score"] = *upload.QualityScore
	}
	if upload.DetectedFaces != nil {
		update["detected_faces"] = *upload.DetectedFaces
		update["analysis.detected_faces"] = *upload.DetectedFaces
	}
	if upload.Orientation != nil {
		update["orientation"] = *upload.Orientation
		update["metadata.orientation"] = *upload.Orientation
	}
	if upload.TakenAt != nil {
		update["taken_at"] = upload.TakenAt
		update["timeline.captured_at"] = upload.TakenAt
	}
	if upload.Location != nil {
		update["location"] = *upload.Location
	}
	if len(upload.SceneTags) > 0 {
		update["scene_tags"] = upload.SceneTags
		update["analysis.scene_tags"] = upload.SceneTags
	}
	if upload.AnalysisError != nil {
		update["analysis_error"] = *upload.AnalysisError
		update["analysis.error"] = *upload.AnalysisError
	}
	if upload.AIInsights != nil {
		update["ai_insights"] = upload.AIInsights
	}
	if upload.Analysis.EmotionScore != nil {
		update["analysis.emotion_score"] = *upload.Analysis.EmotionScore
	}
	if upload.Analysis.FeaturedScore != nil {
		update["analysis.featured_score"] = *upload.Analysis.FeaturedScore
	}
	if upload.Analysis.SafeScore != nil {
		update["analysis.safe_score"] = *upload.Analysis.SafeScore
	}
	if upload.Analysis.Processing != (dto.ProcessingStages{}) {
		update["analysis.processing"] = upload.Analysis.Processing
	}
	_, err := db.Collection("uploads").UpdateOne(ctx, bson.M{"_id": uploadID}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("dao UpdateUploadAnalysis: %w", err)
	}
	return nil
}

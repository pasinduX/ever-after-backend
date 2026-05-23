package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/integrations"
	"go.mongodb.org/mongo-driver/mongo"
)

const openAIEndpoint = "https://api.openai.com/v1/responses"

// AnalysisService processes uploads asynchronously and stores AI metadata.
type AnalysisService struct {
	db         *mongo.Database
	cfg        *integrations.Secrets
	s3         *s3.Client
	jobs       chan string
	httpClient *http.Client
}

func NewAnalysisService(db *mongo.Database, cfg *integrations.Secrets, s3Client *s3.Client) (*AnalysisService, error) {
	if s3Client == nil {
		return nil, errors.New("s3 client is required for analysis service")
	}
	return &AnalysisService{
		db:         db,
		cfg:        cfg,
		s3:         s3Client,
		jobs:       make(chan string, 100),
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (a *AnalysisService) Start() {
	go a.worker()
}

func (a *AnalysisService) Enqueue(uploadID string) {
	select {
	case a.jobs <- uploadID:
	default:
		// if the queue is full, drop the job instead of blocking the HTTP upload path.
		slog.Warn("analysis queue full, dropping upload", "upload_id", uploadID)
	}
}

func (a *AnalysisService) worker() {
	for uploadID := range a.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		_ = a.processUpload(ctx, uploadID)
		cancel()
	}
}

func (a *AnalysisService) processUpload(ctx context.Context, uploadID string) error {
	upload, err := dao.FindUploadByID(ctx, a.db, uploadID)
	if err != nil {
		return err
	}

	if upload.AnalysisStatus == dto.AnalysisStatusProcessing {
		return nil
	}

	if err := dao.SetUploadAnalysisStatus(ctx, a.db, uploadID, dto.AnalysisStatusProcessing, nil); err != nil {
		return err
	}

	if a.cfg.OpenAIAPIKey == "" {
		errMsg := "OPENAI_API_KEY is not configured"
		_ = dao.SetUploadAnalysisStatus(ctx, a.db, uploadID, dto.AnalysisStatusFailed, &errMsg)
		return errors.New(errMsg)
	}

	if upload.FileType != dto.FileTypePhoto {
		// video analysis is not implemented yet, mark as succeeded with minimal metadata.
		return dao.UpdateUploadAnalysis(ctx, a.db, uploadID, &dto.Upload{
			AnalysisStatus: dto.AnalysisStatusSucceeded,
			AnalysisError:  nil,
		})
	}

	result, err := a.analyzePhoto(ctx, upload.FileURL)
	if err != nil {
		errMsg := err.Error()
		_ = dao.SetUploadAnalysisStatus(ctx, a.db, uploadID, dto.AnalysisStatusFailed, &errMsg)
		return err
	}

	return dao.UpdateUploadAnalysis(ctx, a.db, uploadID, result)
}

func (a *AnalysisService) analyzePhoto(ctx context.Context, imageURL string) (*dto.Upload, error) {
	body, err := json.Marshal(map[string]any{
		"model": a.cfg.OpenAIModel,
		"input": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": "Analyze this wedding photo and return only valid JSON with keys: category, quality_score, detected_faces, orientation, scene_tags. Use category values ceremony, candid, dancing, family or other."},
					map[string]any{"type": "input_image", "image_url": imageURL},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.cfg.OpenAIAPIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai response %d: %s", resp.StatusCode, string(data))
	}

	var raw struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	jsonText := ""
	for _, output := range raw.Output {
		for _, content := range output.Content {
			if content.Type == "output_text" || content.Type == "text" {
				jsonText = content.Text
				break
			}
		}
		if jsonText != "" {
			break
		}
	}

	if jsonText == "" {
		return nil, errors.New("openai response missing output text")
	}

	jsonText = sanitizeOpenAIJSON(jsonText)
	var parsed struct {
		Category      string   `json:"category"`
		QualityScore  *float64 `json:"quality_score,omitempty"`
		DetectedFaces *int     `json:"detected_faces,omitempty"`
		Orientation   string   `json:"orientation,omitempty"`
		SceneTags     []string `json:"scene_tags,omitempty"`
		EmotionScore  *float64 `json:"emotion_score,omitempty"`
		FeaturedScore *int     `json:"featured_score,omitempty"`
		SafeScore     *float64 `json:"safe_score,omitempty"`
		CapturedAt    *string  `json:"captured_at,omitempty"`
	}
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse openai response JSON: %w; response=%s", err, jsonText)
	}

	category := normalizeUploadCategory(parsed.Category)
	analysis := dto.UploadAnalysis{
		Status:        dto.AnalysisStatusSucceeded,
		Category:      category,
		SceneTags:     parsed.SceneTags,
		DetectedFaces: parsed.DetectedFaces,
		QualityScore:  parsed.QualityScore,
		EmotionScore:  parsed.EmotionScore,
		FeaturedScore: parsed.FeaturedScore,
		SafeScore:     parsed.SafeScore,
		Error:         nil,
		Processing: dto.ProcessingStages{
			Thumbnail:      dto.AnalysisStatusSucceeded,
			AIAnalysis:     dto.AnalysisStatusSucceeded,
			Moderation:     dto.AnalysisStatusPending,
			DuplicateCheck: dto.AnalysisStatusPending,
		},
	}

	if analysis.FeaturedScore == nil {
		computed := computeFeaturedScore(parsed.QualityScore, parsed.EmotionScore, parsed.DetectedFaces, category)
		analysis.FeaturedScore = &computed
	}

	upload := &dto.Upload{
		Category:       category,
		AnalysisStatus: dto.AnalysisStatusSucceeded,
		QualityScore:   parsed.QualityScore,
		DetectedFaces:  parsed.DetectedFaces,
		Orientation:    nil,
		SceneTags:      parsed.SceneTags,
		AIInsights: map[string]any{
			"raw_category": parsed.Category,
		},
		Analysis: analysis,
	}

	if parsed.Orientation != "" {
		upload.Orientation = &parsed.Orientation
		upload.Metadata.Orientation = &parsed.Orientation
	}

	if parsed.CapturedAt != nil {
		if t, err := time.Parse(time.RFC3339, *parsed.CapturedAt); err == nil {
			upload.TakenAt = &t
			upload.Timeline = dto.UploadTimeline{CapturedAt: &t}
		}
	}

	return upload, nil
}

func computeFeaturedScore(quality, emotion *float64, faces *int, category dto.UploadCategory) int {
	score := 0.0
	if quality != nil {
		score += *quality * 60.0
	}
	if emotion != nil {
		score += *emotion * 30.0
	}
	if faces != nil {
		score += float64(min(*faces, 10)) * 1.5
	}
	if category == dto.CategoryCeremony || category == dto.CategoryFamily {
		score += 10.0
	}
	if score > 100 {
		score = 100
	}
	return int(score)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeUploadCategory(value string) dto.UploadCategory {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(dto.CategoryCeremony):
		return dto.CategoryCeremony
	case string(dto.CategoryCandid):
		return dto.CategoryCandid
	case string(dto.CategoryDancing):
		return dto.CategoryDancing
	case string(dto.CategoryFamily):
		return dto.CategoryFamily
	default:
		return dto.CategoryOther
	}
}

func sanitizeOpenAIJSON(text string) string {
	text = strings.TrimSpace(text)

	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSpace(text)
		if strings.HasPrefix(strings.ToLower(text), "json") {
			text = strings.TrimSpace(text[4:])
		}
		if idx := strings.LastIndex(text, "```"); idx != -1 {
			text = strings.TrimSpace(text[:idx])
		}
	}

	if strings.HasPrefix(text, "`") && strings.HasSuffix(text, "`") {
		text = strings.Trim(text, "`")
	}

	text = strings.TrimSpace(text)
	if i := strings.Index(text, "{"); i != -1 {
		if j := strings.LastIndex(text, "}"); j != -1 && j > i {
			text = strings.TrimSpace(text[i : j+1])
		}
	}

	return strings.TrimSpace(text)
}

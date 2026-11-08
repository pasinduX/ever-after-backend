package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/integrations"
	"go.mongodb.org/mongo-driver/mongo"
)

const openAIEndpoint = "https://api.openai.com/v1/responses"
const analysisWorkers = 20

// maxConcurrentDecodes bounds how many full-size photos are decoded at once
// while rebuilding media. A 24MP photo decodes to ~100MB of pixels, so twenty
// workers decoding together could exhaust memory.
const maxConcurrentDecodes = 4

// maxOriginalBytes caps what re-analysis will download to rebuild media.
const maxOriginalBytes = 80 << 20

// AnalysisService processes uploads asynchronously and stores AI metadata.
type AnalysisService struct {
	db          *mongo.Database
	cfg         *integrations.Secrets
	s3          *s3.Client
	urls        *fileURLs
	jobs        chan string
	httpClient  *http.Client
	decodeSlots chan struct{}
}

func NewAnalysisService(db *mongo.Database, cfg *integrations.Secrets, s3Client *s3.Client) (*AnalysisService, error) {
	if s3Client == nil {
		return nil, errors.New("s3 client is required for analysis service")
	}
	return &AnalysisService{
		db:   db,
		cfg:  cfg,
		s3:   s3Client,
		urls: newFileURLs(cfg, s3Client),
		// Buffer sized for a full wedding (3000 photos) without dropping jobs.
		jobs:        make(chan string, 5000),
		httpClient:  &http.Client{Timeout: 90 * time.Second},
		decodeSlots: make(chan struct{}, maxConcurrentDecodes),
	}, nil
}

func (a *AnalysisService) Start() {
	for i := 0; i < analysisWorkers; i++ {
		go a.worker()
	}
}

func (a *AnalysisService) Enqueue(uploadID string) bool {
	select {
	case a.jobs <- uploadID:
		return true
	default:
		// if the queue is full, drop the job instead of blocking the HTTP upload path.
		// The photo stays "pending", so a later re-analysis picks it up.
		slog.Warn("analysis queue full, dropping upload", "upload_id", uploadID)
		return false
	}
}

func (a *AnalysisService) worker() {
	for uploadID := range a.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		if err := a.processUpload(ctx, uploadID); err != nil {
			slog.Warn("photo analysis failed", "upload_id", uploadID, "error", err.Error())
		}
		cancel()
	}
}

// RerunWedding queues a wedding's photos for album analysis. By default only
// photos that failed or were analysed by an older version are queued; all
// re-analyses every photo. It returns how many photos were queued.
func (a *AnalysisService) RerunWedding(ctx context.Context, weddingID string, all bool) (int, error) {
	ids, err := dao.ResetUploadAnalysisForWedding(ctx, a.db, weddingID, albumAnalysisVersion, all)
	if err != nil {
		return 0, err
	}
	queued := 0
	for _, id := range ids {
		if a.Enqueue(id) {
			queued++
		}
	}
	return queued, nil
}

func (a *AnalysisService) WeddingStatus(ctx context.Context, weddingID string) (*dao.AnalysisCounts, error) {
	return dao.CountUploadAnalysisForWedding(ctx, a.db, weddingID, albumAnalysisVersion)
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
	fail := func(err error) error {
		msg := err.Error()
		_ = dao.SetUploadAnalysisStatus(ctx, a.db, uploadID, dto.AnalysisStatusFailed, &msg)
		return err
	}

	if upload.FileType != dto.FileTypePhoto {
		// video analysis is not implemented yet, mark as succeeded with minimal metadata.
		return dao.UpdateUploadAnalysis(ctx, a.db, uploadID, &dto.Upload{
			AnalysisStatus: dto.AnalysisStatusSucceeded,
			Analysis:       dto.UploadAnalysis{Status: dto.AnalysisStatusSucceeded, Category: upload.Category},
		})
	}

	// Uploads from before the current image pipeline get upright derivatives
	// and a measured palette first. The analysis then reads the rebuilt copy.
	var rebuilt []byte
	if upload.Metadata.PipelineVersion < imagePipelineVersion {
		if rebuilt, err = a.rebuildMedia(ctx, upload); err != nil {
			slog.Warn("media rebuild failed, analysing existing copy", "upload_id", uploadID, "error", err.Error())
		}
	}

	if a.cfg.OpenAIAPIKey == "" {
		return fail(errors.New("OPENAI_API_KEY is not configured"))
	}

	imageURL, err := a.analysisImage(ctx, upload, rebuilt)
	if err != nil {
		return fail(err)
	}

	result, err := a.AnalyzePhoto(ctx, imageURL, a.contextFor(ctx, upload))
	if err != nil {
		return fail(err)
	}
	return dao.UpdateUploadAnalysis(ctx, a.db, uploadID, result)
}

// analysisImage returns the photo as a data URL, preferring the upright
// 1600px rendition. Sending bytes instead of a presigned link means the model
// never fails on an expired or unreachable URL, and it never sees the
// sideways or multi-megabyte original.
func (a *AnalysisService) analysisImage(ctx context.Context, upload *dto.Upload, medium []byte) (string, error) {
	if len(medium) == 0 {
		originalKey := upload.Storage.FileKey
		if originalKey == "" {
			originalKey = upload.FileKey
		}
		key := originalKey
		if mediumKey, ok := a.urls.extractKey(upload.Storage.MediumURL); ok {
			key = mediumKey
		} else if upload.Storage.MediumURL != "" && upload.Storage.MediumURL != upload.Storage.OriginalURL {
			// A public-base URL we can't map back to a key: use the derivative's
			// conventional key instead.
			key = derivativeKey(originalKey, "medium")
		}
		if key == originalKey && !visionReadable(upload.MimeType) {
			return "", fmt.Errorf("%s photos can't be analysed yet: no JPEG copy could be made", upload.MimeType)
		}
		data, err := a.getObject(ctx, key, maxOriginalBytes)
		if err != nil {
			return "", fmt.Errorf("download photo for analysis: %w", err)
		}
		medium = data
	}
	mimeType := http.DetectContentType(medium)
	if !visionReadable(mimeType) {
		return "", fmt.Errorf("%s photos can't be analysed yet: no JPEG copy could be made", upload.MimeType)
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(medium), nil
}

func visionReadable(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	}
	return false
}

// rebuildMedia re-runs the image pipeline on the original: upright
// derivatives, display dimensions, perceptual hash and measured palette. It
// returns the medium rendition (or nil) for the analysis to reuse.
func (a *AnalysisService) rebuildMedia(ctx context.Context, upload *dto.Upload) ([]byte, error) {
	fileKey := upload.Storage.FileKey
	if fileKey == "" {
		fileKey = upload.FileKey
	}
	original, err := a.getObject(ctx, fileKey, maxOriginalBytes)
	if err != nil {
		return nil, err
	}

	exifData := extractEXIF(original)
	select {
	case a.decodeSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	media := analyseImage(original, exifData.Orientation)
	<-a.decodeSlots

	patch := &dto.Upload{
		Storage: upload.Storage,
		EXIF:    dto.UploadEXIF{Orientation: exifData.Orientation},
		Metadata: dto.UploadMetadata{
			PipelineVersion: imagePipelineVersion,
		},
	}
	if media.Decoded {
		dims := media.Dims
		patch.Metadata.Width = intPtrOrNil(dims.Width)
		patch.Metadata.Height = intPtrOrNil(dims.Height)
		patch.Metadata.AspectRatio = floatPtrOrNil(dims.AspectRatio)
		patch.Metadata.Orientation = stringPtrOrNil(dims.Orientation)
		patch.Metadata.Palette = media.Palette
		patch.Metadata.Tone = media.Tone
		patch.PHash = media.PHash
		patch.Analysis.Processing.Thumbnail = dto.AnalysisStatusSucceeded

		if key, ok := storeDerivative(ctx, a.s3, a.cfg.S3Bucket, fileKey, "thumb", media.Thumbnail); ok {
			if url, err := a.urls.build(key); err == nil {
				patch.Storage.ThumbnailURL = url
			}
		}
		if key, ok := storeDerivative(ctx, a.s3, a.cfg.S3Bucket, fileKey, "medium", media.Medium); ok {
			if url, err := a.urls.build(key); err == nil {
				patch.Storage.MediumURL = url
			}
		}
	}
	if err := dao.UpdateUploadMedia(ctx, a.db, upload.ID, patch); err != nil {
		return nil, err
	}
	if media.Medium != nil {
		return media.Medium, nil
	}
	if media.Decoded && visionReadable(http.DetectContentType(original)) {
		return original, nil
	}
	return nil, nil
}

func (a *AnalysisService) getObject(ctx context.Context, key string, limit int64) ([]byte, error) {
	out, err := a.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.cfg.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	data, err := io.ReadAll(io.LimitReader(out.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("object %s is larger than %d bytes", key, limit)
	}
	return data, nil
}

// contextFor gives the analyser the wedding's style and how this photo's
// capture time relates to the wedding day.
func (a *AnalysisService) contextFor(ctx context.Context, upload *dto.Upload) photoContext {
	pctx := photoContext{ShotOn: "unknown", ByGuest: upload.GuestName != nil && *upload.GuestName != ""}
	wedding, err := dao.FindWeddingByID(ctx, a.db, upload.WeddingID)
	if err != nil {
		return pctx
	}
	pctx.CeremonyStyle = wedding.CeremonyStyle
	pctx.VenueType = wedding.VenueType
	pctx.Theme = wedding.WeddingTheme
	if captured := upload.Timeline.CapturedAt; captured != nil && !wedding.WeddingDate.IsZero() {
		switch shot := deriveShootType(&dto.Upload{Timeline: dto.UploadTimeline{CapturedAt: captured}}, wedding.WeddingDate); shot {
		case "pre_shoot":
			pctx.ShotOn = "before the wedding day"
		default:
			pctx.ShotOn = "on or after the wedding day"
		}
	}
	return pctx
}

// categoryFromAnalysis turns the model's event_type/basic_scene labels into the
// UploadCategory the album grouping actually switches on. Without this every
// upload stays CategoryOther and assignActsByCategory collapses the whole
// wedding into a single "Anticipation" act.
func categoryFromAnalysis(eventType, basicScene string, faceCount *int) dto.UploadCategory {
	isGroup := faceCount != nil && *faceCount >= 3

	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "ceremony":
		return dto.CategoryCeremony
	case "reception":
		return dto.CategoryDancing
	case "getting_ready":
		return dto.CategoryCandid
	case "pre_shoot":
		// Staged portraits: a group shot reads as family, a couple shot as candid.
		if isGroup {
			return dto.CategoryFamily
		}
		return dto.CategoryCandid
	}

	// event_type was absent or "other" — fall back to the coarser scene label.
	switch strings.ToLower(strings.TrimSpace(basicScene)) {
	case "ceremony":
		return dto.CategoryCeremony
	case "people":
		if isGroup {
			return dto.CategoryFamily
		}
		return dto.CategoryCandid
	}

	return dto.CategoryOther
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

func (a *AnalysisService) callOpenAIForJSON(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	if a.cfg.OpenAIAPIKey == "" {
		return "", errors.New("OPENAI_API_KEY is not configured")
	}

	body, err := json.Marshal(map[string]any{
		"model": a.cfg.OpenAIModel,
		"input": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": systemPrompt},
					map[string]any{"type": "input_text", "text": userPrompt},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.cfg.OpenAIAPIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai response %d: %s", resp.StatusCode, string(data))
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
		return "", err
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
		return "", errors.New("openai response missing output text")
	}

	return sanitizeOpenAIJSON(jsonText), nil
}

func (a *AnalysisService) GenerateInviteConfig(ctx context.Context, wedding *dto.Wedding) (*dto.CreateInviteRequest, error) {
	props := map[string]any{
		"couple_names":    wedding.CoupleNames,
		"wedding_date":    wedding.WeddingDate.Format("2006-01-02"),
		"wedding_time":    wedding.WeddingTime,
		"venue":           wedding.Venue,
		"address":         wedding.Address,
		"whatsapp_number": wedding.WhatsAppNumber,
		"ages":            wedding.Ages,
		"welcome_message": wedding.WelcomeMessage,
		"template":        wedding.Template,
		"lighting":        wedding.Lighting,
		"story_style":     wedding.StoryStyle,
		"ceremony_style":  wedding.CeremonyStyle,
		"venue_type":      wedding.VenueType,
		"wedding_mood":    wedding.WeddingMood,
		"wedding_theme":   wedding.WeddingTheme,
	}

	userPrompt := fmt.Sprintf(`Create a creative wedding invitation JSON object using these wedding details:

%v

Use a beautiful, meaningful hashtag in the style of #IvorySymphony2026. Craft a tagline that feels like "An Ivory Symphony of Love" and connects deeply to the wedding story.

Return only valid JSON matching the following schema:
{
  "couple": string,
  "hashtag": string,
  "intro": {
    "lines": [string],
    "tagline": string,
    "bg_image": string
  },
  "story": {
    "title": string,
    "events": [{"year": string, "text": string}]
  },
  "details": {
    "date": string,
    "time": string,
    "venue": string,
    "address": string,
    "dress": string
  },
  "countdown": {"target_iso": string, "label": string},
  "qr": {"title": string, "subtitle": string, "url": string},
  "outro": {"line": string, "signature": string},
  "thankyou": {
    "title": string,
    "message": string,
    "signature": string,
    "couple": string,
    "note_from": string,
    "bg_image": string
  }
}`, props)

	jsonText, err := a.callOpenAIForJSON(ctx, "You are a creative wedding content generator. Use the provided wedding data to write an elegant invitation and thank-you section.", userPrompt)
	if err != nil {
		return nil, err
	}

	var req dto.CreateInviteRequest
	if err := json.Unmarshal([]byte(jsonText), &req); err != nil {
		return nil, fmt.Errorf("failed to parse invite JSON: %w; response=%s", err, jsonText)
	}
	return &req, nil
}

func (a *AnalysisService) GenerateThankYouConfig(ctx context.Context, wedding *dto.Wedding) (*dto.CreateThankYouRequest, error) {
	props := map[string]any{
		"couple_names":    wedding.CoupleNames,
		"wedding_date":    wedding.WeddingDate.Format("2006-01-02"),
		"wedding_time":    wedding.WeddingTime,
		"venue":           wedding.Venue,
		"address":         wedding.Address,
		"whatsapp_number": wedding.WhatsAppNumber,
		"ages":            wedding.Ages,
		"welcome_message": wedding.WelcomeMessage,
		"template":        wedding.Template,
		"lighting":        wedding.Lighting,
		"story_style":     wedding.StoryStyle,
		"ceremony_style":  wedding.CeremonyStyle,
		"venue_type":      wedding.VenueType,
		"wedding_mood":    wedding.WeddingMood,
		"wedding_theme":   wedding.WeddingTheme,
	}

	userPrompt := fmt.Sprintf(`Create a creative wedding thank-you card JSON object using these wedding details:

%v

Use an elegant wedding hashtag in the style of #IvorySymphony2026 and keep the tone warm, poetic, and deeply meaningful. Mention the feeling of "An Ivory Symphony of Love" in the copy when appropriate.

Return only valid JSON matching the following schema:
{
  "template": string,
  "couple": string,
  "date": string,
  "venue": string,
  "hashtag": string,
  "hero_image": string,
  "portrait": string,
  "intro": [string],
  "message": string,
  "signature": string,
  "gallery": [string],
  "closing": string
}
`, props)

	jsonText, err := a.callOpenAIForJSON(ctx, "You are a creative wedding content generator. Use the provided wedding data to write an elegant thank-you card.", userPrompt)
	if err != nil {
		return nil, err
	}

	var req dto.CreateThankYouRequest
	if err := json.Unmarshal([]byte(jsonText), &req); err != nil {
		return nil, fmt.Errorf("failed to parse thank-you JSON: %w; response=%s", err, jsonText)
	}
	return &req, nil
}

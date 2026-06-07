package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/integrations"
	"go.mongodb.org/mongo-driver/mongo"
)

type AlbumService struct {
	db         *mongo.Database
	cfg        *integrations.Secrets
	jobs       chan string
	httpClient *http.Client
}

func NewAlbumService(db *mongo.Database, cfg *integrations.Secrets) *AlbumService {
	return &AlbumService{
		db:         db,
		cfg:        cfg,
		jobs:       make(chan string, 50),
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}
}

func (s *AlbumService) Start() {
	go s.worker()
}

func (s *AlbumService) Enqueue(weddingID string) {
	select {
	case s.jobs <- weddingID:
	default:
		slog.Warn("album generation queue full", "wedding_id", weddingID)
	}
}

func (s *AlbumService) worker() {
	for weddingID := range s.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		if err := s.generateActs(ctx, weddingID); err != nil {
			slog.Error("album generation failed", "wedding_id", weddingID, "error", err)
			errMsg := err.Error()
			_ = dao.UpsertAlbumConfig(ctx, s.db, &dto.AlbumConfig{
				WeddingID: weddingID,
				Status:    dto.AlbumStatusFailed,
				Error:     &errMsg,
			})
		}
		cancel()
	}
}

func (s *AlbumService) GenerateAsync(ctx context.Context, weddingID string, style dto.AlbumStyle) error {
	if style == "" {
		style = dto.AlbumStyleCinematic
	}
	cfg := &dto.AlbumConfig{
		WeddingID: weddingID,
		Style:     style,
		Status:    dto.AlbumStatusProcessing,
	}
	if err := dao.UpsertAlbumConfig(ctx, s.db, cfg); err != nil {
		return err
	}
	s.Enqueue(weddingID)
	return nil
}

func (s *AlbumService) GetStatus(ctx context.Context, weddingID string) (*dto.AlbumConfig, error) {
	cfg, err := dao.FindAlbumConfig(ctx, s.db, weddingID)
	if errors.Is(err, mongo.ErrNoDocuments) || (err != nil && strings.Contains(err.Error(), "no documents")) {
		return &dto.AlbumConfig{
			WeddingID: weddingID,
			Status:    dto.AlbumStatusPending,
		}, nil
	}
	return cfg, err
}

func (s *AlbumService) GetActs(ctx context.Context, weddingID string) ([]*dto.StoryAct, error) {
	return dao.FindStoryActs(ctx, s.db, weddingID)
}

func (s *AlbumService) ConfirmActs(ctx context.Context, req dto.ConfirmActsRequest) error {
	for _, confirmation := range req.Acts {
		if err := dao.UpdateStoryAct(ctx, s.db, confirmation.ID, confirmation.PhotoIDs, confirmation.Confirmed); err != nil {
			return err
		}
	}
	return nil
}

func (s *AlbumService) SetStyle(ctx context.Context, weddingID string, style dto.AlbumStyle) error {
	cfg, err := s.GetStatus(ctx, weddingID)
	if err != nil {
		return err
	}
	cfg.WeddingID = weddingID
	cfg.Style = style
	return dao.UpsertAlbumConfig(ctx, s.db, cfg)
}

// --- generation pipeline ---

func (s *AlbumService) generateActs(ctx context.Context, weddingID string) error {
	wedding, err := dao.FindWeddingByID(ctx, s.db, weddingID)
	if err != nil {
		return fmt.Errorf("wedding not found: %w", err)
	}

	uploads, err := dao.FindApprovedUploadsByWeddingSorted(ctx, s.db, weddingID)
	if err != nil {
		return fmt.Errorf("failed to load uploads: %w", err)
	}

	if len(uploads) < 5 {
		return errors.New("not enough photos to generate acts (minimum 5)")
	}

	// Pass 1: temporal segmentation
	segs := temporalSegments(uploads)

	// Pass 2: summarise segments
	summaries := summariseSegments(segs)

	// Pass 3: assign acts via OpenAI (fallback to category-based if OpenAI unavailable)
	var acts []*dto.StoryAct
	if s.cfg.OpenAIAPIKey != "" {
		acts, err = s.assignActsViaAI(ctx, wedding, segs, summaries)
	}
	if err != nil || len(acts) == 0 {
		if err != nil {
			slog.Warn("AI act assignment failed, using category fallback", "error", err)
		}
		acts = assignActsByCategory(weddingID, uploads)
	}

	if err := dao.UpsertStoryActs(ctx, s.db, weddingID, acts); err != nil {
		return err
	}

	// Compute overall confidence
	avgConf := 0.0
	needsReview := false
	for _, a := range acts {
		avgConf += a.Confidence
		if a.NeedsReview {
			needsReview = true
		}
	}
	if len(acts) > 0 {
		avgConf /= float64(len(acts))
	}

	return dao.UpsertAlbumConfig(ctx, s.db, &dto.AlbumConfig{
		WeddingID:   weddingID,
		Status:      dto.AlbumStatusCompleted,
		ActsCount:   len(acts),
		Confidence:  avgConf,
		NeedsReview: needsReview,
	})
}

// --- Pass 1: temporal segmentation ---

type photoSegment struct {
	photos    []*dto.Upload
	startTime time.Time
	endTime   time.Time
	hardStart bool
}

func photoTime(u *dto.Upload) time.Time {
	if u.Timeline.CapturedAt != nil {
		return *u.Timeline.CapturedAt
	}
	if u.TakenAt != nil {
		return *u.TakenAt
	}
	return u.UploadedAt
}

func temporalSegments(uploads []*dto.Upload) []photoSegment {
	if len(uploads) == 0 {
		return nil
	}

	sorted := make([]*dto.Upload, len(uploads))
	copy(sorted, uploads)
	sort.Slice(sorted, func(i, j int) bool {
		return photoTime(sorted[i]).Before(photoTime(sorted[j]))
	})

	var segs []photoSegment
	cur := photoSegment{
		photos:    []*dto.Upload{sorted[0]},
		startTime: photoTime(sorted[0]),
	}

	for i := 1; i < len(sorted); i++ {
		t := photoTime(sorted[i])
		prev := photoTime(sorted[i-1])
		gap := t.Sub(prev)
		if gap < 0 {
			gap = -gap
		}

		if gap > 15*time.Minute {
			cur.endTime = photoTime(sorted[i-1])
			segs = append(segs, cur)
			cur = photoSegment{
				photos:    []*dto.Upload{sorted[i]},
				startTime: t,
				hardStart: true,
			}
		} else {
			cur.photos = append(cur.photos, sorted[i])
		}
	}
	cur.endTime = photoTime(sorted[len(sorted)-1])
	segs = append(segs, cur)
	return segs
}

// --- Pass 2: segment summaries ---

type segmentSummary struct {
	Index            int     `json:"index"`
	PhotoCount       int     `json:"photo_count"`
	TimePosition     float64 `json:"time_position"`
	DominantLabel    string  `json:"dominant_label"`
	LabelConsistency float64 `json:"label_consistency"`
	StudioPercent    float64 `json:"studio_percent"`
	HardStart        bool    `json:"hard_start"`
}

func summariseSegments(segs []photoSegment) []segmentSummary {
	if len(segs) == 0 {
		return nil
	}

	dayStart := segs[0].startTime
	dayEnd := segs[len(segs)-1].endTime
	totalDuration := dayEnd.Sub(dayStart).Minutes()

	summaries := make([]segmentSummary, len(segs))
	for i, seg := range segs {
		labelCount := make(map[string]int)
		studioCount := 0
		for _, u := range seg.photos {
			labelCount[string(u.Analysis.Category)]++
			if u.Analysis.StudioProbability != nil && *u.Analysis.StudioProbability > 0.6 {
				studioCount++
			}
		}

		dominant := "other"
		maxCount := 0
		for label, count := range labelCount {
			if count > maxCount {
				maxCount = count
				dominant = label
			}
		}

		consistency := 0.0
		studioPercent := 0.0
		if len(seg.photos) > 0 {
			consistency = float64(maxCount) / float64(len(seg.photos))
			studioPercent = float64(studioCount) / float64(len(seg.photos))
		}

		timePos := 0.0
		if totalDuration > 0 {
			timePos = seg.startTime.Sub(dayStart).Minutes() / totalDuration
		}

		summaries[i] = segmentSummary{
			Index:            i,
			PhotoCount:       len(seg.photos),
			TimePosition:     timePos,
			DominantLabel:    dominant,
			LabelConsistency: consistency,
			StudioPercent:    studioPercent,
			HardStart:        seg.hardStart,
		}
	}
	return summaries
}

// --- Pass 3: AI act assignment ---

type segmentAssignment struct {
	SegmentIndex int     `json:"segment_index"`
	Act          string  `json:"act"`
	Confidence   float64 `json:"confidence"`
}

func (s *AlbumService) assignActsViaAI(ctx context.Context, wedding *dto.Wedding, segs []photoSegment, summaries []segmentSummary) ([]*dto.StoryAct, error) {
	summaryJSON, err := json.Marshal(summaries)
	if err != nil {
		return nil, err
	}

	coupleNames := strings.Join([]string(wedding.CoupleNames), " & ")
	systemPrompt := "You are an expert wedding photo organiser. Analyse photo segments and assign wedding story acts. Return only valid JSON."
	userPrompt := fmt.Sprintf(`Assign story acts to these wedding photo segments for %s.

Wedding: venue=%s, date=%s, style=%s

Photo segments (JSON):
%s

Assign each segment to exactly one act: ANTICIPATION, CEREMONY, FAMILY_BONDS, JUST_THE_TWO, CELEBRATION, FINAL_DANCE

Typical order: ANTICIPATION → CEREMONY → FAMILY_BONDS → JUST_THE_TWO → CELEBRATION → FINAL_DANCE
Use dominant_label and time_position as primary signals.

Return JSON array only:
[{"segment_index": 0, "act": "CEREMONY", "confidence": 0.91}]

Confidence: >0.85 = clear signal, 0.60-0.85 = some ambiguity, <0.60 = unclear`,
		coupleNames, wedding.Venue, wedding.WeddingDate.Format("2006-01-02"), wedding.StoryStyle,
		string(summaryJSON),
	)

	jsonText, err := s.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var assignments []segmentAssignment
	if err := json.Unmarshal([]byte(jsonText), &assignments); err != nil {
		return nil, fmt.Errorf("parse act assignments: %w; response=%s", err, jsonText)
	}

	return buildStoryActs(wedding.ID, segs, assignments), nil
}

func buildStoryActs(weddingID string, segs []photoSegment, assignments []segmentAssignment) []*dto.StoryAct {
	assignMap := make(map[int]segmentAssignment, len(assignments))
	for _, a := range assignments {
		assignMap[a.SegmentIndex] = a
	}

	actPhotos := make(map[dto.ActLabel][]*dto.Upload)
	actConfs := make(map[dto.ActLabel][]float64)

	for i, seg := range segs {
		a, ok := assignMap[i]
		if !ok {
			continue
		}
		label := dto.ActLabel(a.Act)
		actPhotos[label] = append(actPhotos[label], seg.photos...)
		actConfs[label] = append(actConfs[label], a.Confidence)
	}

	actOrder := []dto.ActLabel{
		dto.ActAnticipation, dto.ActCeremony, dto.ActFamilyBonds,
		dto.ActJustTheTwo, dto.ActCelebration, dto.ActFinalDance,
	}

	acts := make([]*dto.StoryAct, 0, 6)
	for order, label := range actOrder {
		photos, ok := actPhotos[label]
		if !ok || len(photos) == 0 {
			continue
		}

		confs := actConfs[label]
		avgConf := 0.0
		for _, c := range confs {
			avgConf += c
		}
		avgConf /= float64(len(confs))

		photoIDs := make([]string, len(photos))
		for i, p := range photos {
			photoIDs[i] = p.ID
		}

		act := &dto.StoryAct{
			ID:          uuid.NewString(),
			WeddingID:   weddingID,
			Label:       label,
			Title:       actTitle(label),
			Quote:       actQuote(label),
			Confidence:  avgConf,
			NeedsReview: avgConf < 0.60,
			PhotoIDs:    photoIDs,
			HeroPhotoID: selectHeroPhoto(photos),
			Order:       order,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		acts = append(acts, act)
	}
	return acts
}

func selectHeroPhoto(photos []*dto.Upload) string {
	if len(photos) == 0 {
		return ""
	}
	best := photos[0]
	bestScore := -1.0
	for _, u := range photos {
		score := 0.0
		if u.Analysis.StudioProbability != nil {
			score += *u.Analysis.StudioProbability * 50.0
		}
		if u.Analysis.FeaturedScore != nil {
			score += float64(*u.Analysis.FeaturedScore) * 0.5
		}
		if score > bestScore {
			bestScore = score
			best = u
		}
	}
	return best.ID
}

// --- Category-based fallback ---

func assignActsByCategory(weddingID string, uploads []*dto.Upload) []*dto.StoryAct {
	buckets := map[dto.ActLabel][]*dto.Upload{
		dto.ActAnticipation: {},
		dto.ActCeremony:     {},
		dto.ActFamilyBonds:  {},
		dto.ActCelebration:  {},
	}

	for _, u := range uploads {
		switch u.Analysis.Category {
		case dto.CategoryCeremony:
			buckets[dto.ActCeremony] = append(buckets[dto.ActCeremony], u)
		case dto.CategoryFamily:
			buckets[dto.ActFamilyBonds] = append(buckets[dto.ActFamilyBonds], u)
		case dto.CategoryDancing:
			buckets[dto.ActCelebration] = append(buckets[dto.ActCelebration], u)
		default:
			buckets[dto.ActAnticipation] = append(buckets[dto.ActAnticipation], u)
		}
	}

	actOrder := []dto.ActLabel{dto.ActAnticipation, dto.ActCeremony, dto.ActFamilyBonds, dto.ActCelebration}
	acts := make([]*dto.StoryAct, 0, 4)
	for order, label := range actOrder {
		photos := buckets[label]
		if len(photos) == 0 {
			continue
		}
		photoIDs := make([]string, len(photos))
		for i, p := range photos {
			photoIDs[i] = p.ID
		}
		acts = append(acts, &dto.StoryAct{
			ID:          uuid.NewString(),
			WeddingID:   weddingID,
			Label:       label,
			Title:       actTitle(label),
			Quote:       actQuote(label),
			Confidence:  0.70,
			NeedsReview: false,
			PhotoIDs:    photoIDs,
			HeroPhotoID: selectHeroPhoto(photos),
			Order:       order,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}
	return acts
}

// --- OpenAI helper ---

func (s *AlbumService) callOpenAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if s.cfg.OpenAIAPIKey == "" {
		return "", errors.New("OPENAI_API_KEY not configured")
	}

	body, err := json.Marshal(map[string]any{
		"model": s.cfg.OpenAIModel,
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.cfg.OpenAIAPIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai %d: %s", resp.StatusCode, string(data))
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

	for _, output := range raw.Output {
		for _, content := range output.Content {
			if content.Type == "output_text" || content.Type == "text" {
				return sanitizeOpenAIJSON(content.Text), nil
			}
		}
	}
	return "", errors.New("openai response missing output text")
}

// --- Act metadata ---

func actTitle(label dto.ActLabel) string {
	switch label {
	case dto.ActAnticipation:
		return "The Anticipation"
	case dto.ActCeremony:
		return "The Ceremony"
	case dto.ActFamilyBonds:
		return "Family Bonds"
	case dto.ActJustTheTwo:
		return "Just the Two of Us"
	case dto.ActCelebration:
		return "The Celebration"
	case dto.ActFinalDance:
		return "The Final Dance"
	default:
		return "Memories"
	}
}

func actQuote(label dto.ActLabel) string {
	switch label {
	case dto.ActAnticipation:
		return "The quiet moments before everything changed."
	case dto.ActCeremony:
		return "The vows, the first look, the beginning."
	case dto.ActFamilyBonds:
		return "The people who shaped who you are."
	case dto.ActJustTheTwo:
		return "A golden hour that belonged only to you."
	case dto.ActCelebration:
		return "The energy that filled the room after dark."
	case dto.ActFinalDance:
		return "The last song, the first night."
	default:
		return "Moments worth remembering."
	}
}

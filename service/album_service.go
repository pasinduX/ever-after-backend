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
	payload    *PayloadService
	jobs       chan string
	httpClient *http.Client
}

func NewAlbumService(db *mongo.Database, cfg *integrations.Secrets, payload *PayloadService) *AlbumService {
	return &AlbumService{
		db:         db,
		cfg:        cfg,
		payload:    payload,
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

func (s *AlbumService) ConfirmActs(ctx context.Context, weddingID string, req dto.ConfirmActsRequest) error {
	for _, confirmation := range req.Acts {
		if err := dao.UpdateStoryAct(ctx, s.db, confirmation.ID, confirmation.PhotoIDs, confirmation.Confirmed); err != nil {
			return err
		}
	}
	// Acts changed — rebuild and store the payload.
	if s.payload != nil {
		if p, err := s.payload.BuildPayload(ctx, weddingID); err == nil {
			_ = dao.UpsertAlbumPayload(ctx, s.db, p)
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
	if err := dao.UpsertAlbumConfig(ctx, s.db, cfg); err != nil {
		return err
	}
	// Style changed — rebuild and store the payload.
	if s.payload != nil {
		if p, err := s.payload.BuildPayload(ctx, weddingID); err == nil {
			_ = dao.UpsertAlbumPayload(ctx, s.db, p)
		}
	}
	return nil
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

	if err := dao.UpsertAlbumConfig(ctx, s.db, &dto.AlbumConfig{
		WeddingID:   weddingID,
		Status:      dto.AlbumStatusCompleted,
		ActsCount:   len(acts),
		Confidence:  avgConf,
		NeedsReview: needsReview,
	}); err != nil {
		return err
	}

	// Fetch the album config to get the requested style.
	albumCfg, _ := dao.FindAlbumConfig(ctx, s.db, weddingID)
	style := dto.AlbumStyleCinematic
	if albumCfg != nil && albumCfg.Style != "" {
		style = albumCfg.Style
	}

	// Build and store the payload.
	// Prefer AI-generated payload (full creative control); fall back to Go builder.
	var payload *dto.AlbumPayload
	if s.cfg.OpenAIAPIKey != "" {
		var aiErr error
		payload, aiErr = s.generatePayloadViaAI(ctx, wedding, uploads, style)
		if aiErr != nil {
			slog.Warn("AI payload generation failed, falling back to Go builder", "wedding_id", weddingID, "error", aiErr)
		}
	}
	if payload == nil && s.payload != nil {
		var fbErr error
		payload, fbErr = s.payload.BuildPayload(ctx, weddingID)
		if fbErr != nil {
			slog.Warn("Go payload build also failed", "wedding_id", weddingID, "error", fbErr)
		}
	}
	if payload != nil {
		_ = dao.UpsertAlbumPayload(ctx, s.db, payload)
	}
	return nil
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

func parseSegmentAssignments(jsonText string) ([]segmentAssignment, error) {
	var assignments []segmentAssignment
	if err := json.Unmarshal([]byte(jsonText), &assignments); err == nil {
		return assignments, nil
	}

	var assignment segmentAssignment
	if err := json.Unmarshal([]byte(jsonText), &assignment); err != nil {
		return nil, err
	}
	return []segmentAssignment{assignment}, nil
}

func (s *AlbumService) assignActsViaAI(ctx context.Context, wedding *dto.Wedding, segs []photoSegment, summaries []segmentSummary) ([]*dto.StoryAct, error) {
	summaryJSON, err := json.Marshal(summaries)
	if err != nil {
		return nil, err
	}

	coupleNames := strings.Join([]string(wedding.CoupleNames), " & ")
	systemPrompt := "You are an expert wedding photo organiser. Analyse photo segments and assign wedding story acts. Return only valid JSON."
	userPrompt := fmt.Sprintf(`Assign story acts to these wedding photo segments.

Wedding details:
- Couple: %s
- Date: %s
- Venue: %s
- Venue type: %s
- Ceremony style: %s
- Story style: %s
- Wedding mood: %s
- Wedding theme: %s
- Lighting: %s

Use these details as context: venue type (indoor/outdoor/beach/garden) influences which acts appear early vs late; ceremony style affects the CEREMONY act duration; mood and theme hint at whether JUST_THE_TWO or FINAL_DANCE will be prominent.

Photo segments (JSON):
%s

Assign each segment to exactly one act: ANTICIPATION, CEREMONY, FAMILY_BONDS, JUST_THE_TWO, CELEBRATION, FINAL_DANCE

Typical order: ANTICIPATION → CEREMONY → FAMILY_BONDS → JUST_THE_TWO → CELEBRATION → FINAL_DANCE
Use dominant_label and time_position as primary signals; use wedding context above as tiebreakers.

Return JSON array only:
[{"segment_index": 0, "act": "CEREMONY", "confidence": 0.91}]

Confidence: >0.85 = clear signal, 0.60-0.85 = some ambiguity, <0.60 = unclear`,
		coupleNames,
		wedding.WeddingDate.Format("02 January 2006"),
		wedding.Venue,
		wedding.VenueType,
		wedding.CeremonyStyle,
		wedding.StoryStyle,
		wedding.WeddingMood,
		wedding.WeddingTheme,
		wedding.Lighting,
		string(summaryJSON),
	)

	jsonText, err := s.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	assignments, err := parseSegmentAssignments(jsonText)
	if err != nil {
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

// --- AI payload generation ---

// compactImageInfo is the minimal projection sent to the AI — no URLs.
// The AI references images by stable id only; URLs are injected after by hydratePayload.
type compactImageInfo struct {
	ID          string   `json:"id"`
	Orientation string   `json:"orientation,omitempty"`
	Category    string   `json:"category"`
	SceneTags   []string `json:"scene_tags,omitempty"`
	Faces       int      `json:"faces"`
	Quality     float64  `json:"quality"`
}

// preProcessUploads filters approved + usable photos, deduplicates by phash/content_hash,
// and sorts by category then upload time to build a stable timeline for the AI.
func preProcessUploads(uploads []*dto.Upload) []*dto.Upload {
	seen := make(map[string]struct{})
	out := make([]*dto.Upload, 0, len(uploads))
	for _, u := range uploads {
		if !u.IsApproved {
			continue
		}
		if v, ok := u.AIInsights["is_usable"]; ok {
			if b, ok2 := v.(bool); ok2 && !b {
				continue
			}
		}
		key := u.PHash
		if key == "" {
			key = u.ContentHash
		}
		if key != "" {
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
		}
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool {
		ci, cj := string(out[i].Analysis.Category), string(out[j].Analysis.Category)
		if ci != cj {
			return ci < cj
		}
		return photoTime(out[i]).Before(photoTime(out[j]))
	})
	return out
}

func deriveQuality(u *dto.Upload) float64 {
	if u.Analysis.QualityScore != nil {
		return *u.Analysis.QualityScore
	}
	if u.QualityScore != nil {
		return *u.QualityScore
	}
	return 0.5
}

func (s *AlbumService) generatePayloadViaAI(
	ctx context.Context,
	wedding *dto.Wedding,
	uploads []*dto.Upload,
	style dto.AlbumStyle,
) (*dto.AlbumPayload, error) {
	// Pre-process: filter, dedup by phash/content_hash, sort by category + time.
	filtered := preProcessUploads(uploads)
	if len(filtered) == 0 {
		return nil, errors.New("no usable photos after pre-processing")
	}

	// Build uploadMap for later hydration + compact list for the AI (no URLs).
	uploadMap := make(map[string]*dto.Upload, len(filtered))
	compactImages := make([]compactImageInfo, 0, len(filtered))
	for _, u := range filtered {
		uploadMap[u.ID] = u
		info := compactImageInfo{
			ID:        u.ID,
			Category:  string(u.Analysis.Category),
			SceneTags: u.Analysis.SceneTags,
			Quality:   deriveQuality(u),
		}
		if u.Orientation != nil {
			info.Orientation = *u.Orientation
		}
		if u.Analysis.DetectedFaces != nil {
			info.Faces = *u.Analysis.DetectedFaces
		} else if u.DetectedFaces != nil {
			info.Faces = *u.DetectedFaces
		}
		compactImages = append(compactImages, info)
	}

	coupleNames := strings.Join([]string(wedding.CoupleNames), " & ")
	theme := themeForStyle(style, wedding.WeddingMood, wedding.WeddingTheme)

	colorGrade := "muted"
	vignette := 0.25
	if style == dto.AlbumStyleRomantic {
		colorGrade = "warm"
		vignette = 0.10
	}

	// AI input: wedding facts + style brief + compact image list (no URLs).
	aiInput := map[string]any{
		"wedding": map[string]any{
			"couple_names":    []string(wedding.CoupleNames),
			"wedding_date":    wedding.WeddingDate.Format("2006-01-02"),
			"venue":           wedding.Venue,
			"tier":            string(wedding.Tier),
			"ceremony_style":  wedding.CeremonyStyle,
			"mood":            wedding.WeddingMood,
			"theme":           wedding.WeddingTheme,
			"story_style":     wedding.StoryStyle,
			"lighting":        wedding.Lighting,
			"welcome_message": wedding.WelcomeMessage,
		},
		"style_brief": map[string]any{
			"style": string(style),
			"mood":  wedding.WeddingMood,
		},
		"images": compactImages,
	}

	inputJSON, err := json.Marshal(aiInput)
	if err != nil {
		return nil, err
	}

	systemPrompt := `You are a cinematic wedding film editor. Build a narrative arc across chapters.
Reference images by id only. Never invent ids. Prefer portrait + quality >= 0.8 + faces >= 2 for cover/fullscreen/finale.
Use grids/filmstrips for the long tail. Keep 4–7 blocks per chapter.
Return ONLY valid JSON — no markdown, no code fences, no commentary.`

	userPrompt := fmt.Sprintf(`Design a cinematic wedding album from the following data. Return ONLY valid JSON.

%s

RULES:
- 3–5 chapters; act values must be from: ANTICIPATION, CEREMONY, CELEBRATION, FINALE
- Last chapter MUST be act "FINALE" containing a "finale" block
- Include every image id in at least one block; never invent ids
- Every image reference: {"id": "<exact id>"} — do not include urls
- 4–7 blocks per chapter (FINALE may have 1–2 blocks)
- Cover: pick the highest quality portrait with faces >= 2
- gallery_grid/masonry columns must be 2 or 3 only — never 4 or more
- Motions — fullscreen_image: {"reveal":"fade-in","ken_burns":true,"parallax":0.3,"duration":1.4}
            filmstrip:        {"reveal":"slide-left","duration":0.5}
            masonry/gallery:  {"reveal":"fade-up","duration":0.6}
            quote/section:    {"reveal":"fade-up","duration":0.5}
            finale:           {"reveal":"fade-in","ken_burns":true,"parallax":0.2,"duration":2.4}
- Apply filter {"grade":"%s","vignette":%g} to every image block
- finale block: put the couple name in top-level "title" and the closing line in top-level "subtitle" — NOT inside an overlay

Return this exact JSON structure:
{
  "version": "1.0",
  "album_id": "%s",
  "style": "%s",
  "theme": {"primary_color": "%s", "secondary_color": "%s", "font": "%s", "mood": "%s"},
  "cover": {"image": {"id": "..."}, "title": "%s", "subtitle": "%s · %s"},
  "chapters": [
    {
      "id": "ch_anticipation",
      "title": "The Anticipation",
      "act": "ANTICIPATION",
      "blocks": [
        {"id": "b1", "type": "section_title", "title": "...", "motion": {"reveal": "fade-up", "duration": 0.5}},
        {"id": "b2", "type": "quote", "layout": "centered", "text": "...", "motion": {"reveal": "fade-up", "duration": 0.8}},
        {"id": "b3", "type": "fullscreen_image", "image": {"id": "..."}, "height": "full",
          "overlay": {"title": "...", "alignment": "bottom-left", "reveal_delay": 0.6},
          "motion": {"reveal": "fade-in", "ken_burns": true, "parallax": 0.3, "duration": 1.4},
          "filter": {"grade": "%s", "vignette": %g}},
        {"id": "b4", "type": "gallery_grid", "columns": 3, "images": [{"id": "..."}],
          "motion": {"reveal": "fade-up", "duration": 0.6}, "filter": {"grade": "%s", "vignette": %g}}
      ]
    },
    {
      "id": "ch_finale",
      "title": "Finale",
      "act": "FINALE",
      "blocks": [
        {"id": "b_finale", "type": "finale", "background_image": {"id": "..."},
          "title": "The End — and the Beginning",
          "subtitle": "Thank you for being part of our story",
          "motion": {"reveal": "fade-in", "ken_burns": true, "parallax": 0.2, "duration": 2.4}}
      ]
    }
  ],
  "soundtrack": {"autoplay": false, "volume": 0.35}
}`,
		string(inputJSON),
		colorGrade, vignette,
		wedding.ID, string(style),
		theme.PrimaryColor, theme.SecondaryColor, theme.Font, theme.Mood,
		coupleNames, wedding.WeddingDate.Format("2 January 2006"), wedding.Venue,
		colorGrade, vignette,
		colorGrade, vignette,
	)

	// Text-only call — the AI is a creative director working from metadata, not image pixels.
	jsonText, err := s.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var payload dto.AlbumPayload
	if err := json.Unmarshal([]byte(jsonText), &payload); err != nil {
		return nil, fmt.Errorf("parse AI payload JSON: %w; snippet=%s", err, jsonText[:min(len(jsonText), 300)])
	}

	payload.Version = "1.0"
	payload.AlbumID = wedding.ID
	if payload.Style == "" {
		payload.Style = style
	}

	// Hydrate: swap id-only AssetObjects for full objects (URL + dimensions).
	// Blocks referencing unknown ids are dropped rather than crashing.
	hydratePayload(&payload, uploadMap)

	if len(payload.Chapters) == 0 {
		return nil, errors.New("AI returned no chapters")
	}

	return &payload, nil
}

// hydratePayload resolves every AssetObject's id to a live URL + dimensions from uploadMap.
// A block whose primary image id is unknown is dropped entirely.
// Unknown ids inside image arrays are filtered out.
func hydratePayload(payload *dto.AlbumPayload, uploadMap map[string]*dto.Upload) {
	fill := func(a *dto.AssetObject) bool {
		if a == nil || a.ID == "" {
			return false
		}
		u, ok := uploadMap[a.ID]
		if !ok {
			return false
		}
		a.URL = u.Storage.OriginalURL
		if a.URL == "" {
			a.URL = u.FileURL
		}
		if u.Metadata.Width != nil {
			a.Width = *u.Metadata.Width
		}
		if u.Metadata.Height != nil {
			a.Height = *u.Metadata.Height
		}
		if u.Metadata.AspectRatio != nil {
			a.AspectRatio = *u.Metadata.AspectRatio
		}
		return true
	}

	fill(payload.Cover.Image)

	for ci := range payload.Chapters {
		var kept []dto.AlbumBlock
		for bi := range payload.Chapters[ci].Blocks {
			blk := &payload.Chapters[ci].Blocks[bi]
			if blk.Image != nil && !fill(blk.Image) {
				continue
			}
			fill(blk.BackgroundImage)
			fill(blk.LeftImage)
			fill(blk.FallbackImage)
			var valid []dto.AssetObject
			for ii := range blk.Images {
				if fill(&blk.Images[ii]) {
					valid = append(valid, blk.Images[ii])
				}
			}
			blk.Images = valid
			kept = append(kept, *blk)
		}
		payload.Chapters[ci].Blocks = kept
	}
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

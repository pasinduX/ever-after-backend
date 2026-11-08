package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/storyvows/backend/dto"
)

var weddingDay = time.Date(2026, 12, 12, 0, 0, 0, 0, time.UTC)

type photoSpec struct {
	id     string
	minute int
	moment string
	hero   float64
	cover  float64
	couple bool
	fit    string
	issues []string
	phash  string
}

func makeUpload(p photoSpec) *dto.Upload {
	at := weddingDay.Add(10*time.Hour + time.Duration(p.minute)*time.Minute)
	w, h := 1600, 1067
	fit := p.fit
	if fit == "" {
		fit = dto.AlbumFitKeep
	}
	return &dto.Upload{
		ID: p.id, FileType: dto.FileTypePhoto, IsApproved: true, PHash: p.phash,
		Timeline: dto.UploadTimeline{CapturedAt: &at},
		Metadata: dto.UploadMetadata{Width: &w, Height: &h,
			Palette: []dto.PaletteSwatch{{Hex: "#f4ede2", Share: 0.5, Role: "ground"}}},
		Analysis: dto.UploadAnalysis{Album: &dto.AlbumPhotoAnalysis{
			Version: albumAnalysisVersion, Fit: fit, Issues: p.issues, Moment: p.moment,
			HeroScore: p.hero, CoverScore: p.cover, CouplePresent: p.couple, PeopleCount: peopleIn(p),
			AestheticScore: 0.6, EmotionScore: 0.5, AltText: "alt " + p.id,
			FocalPoint: dto.FocalPoint{X: 0.5, Y: 0.4}, TextArea: "top",
		}},
	}
}

func peopleIn(p photoSpec) int {
	if p.couple {
		return 2
	}
	if p.moment == "details" {
		return 0
	}
	return 1
}

func testWedding() *dto.Wedding {
	return &dto.Wedding{
		ID: "w1", CoupleNames: dto.StringArray{"Julian", "Sofia"}, WeddingDate: weddingDay,
		Venue: "The Willow House", CeremonyStyle: "Kandyan (Sri Lankan)", Template: "golden-hour",
	}
}

func TestBuildPageBriefPicksAnchorsAndExcludesRuinedPhotos(t *testing.T) {
	uploads := []*dto.Upload{
		makeUpload(photoSpec{id: "ready", minute: 0, moment: "getting_ready", hero: 0.5, cover: 0.3}),
		makeUpload(photoSpec{id: "portrait", minute: 30, moment: "couple_portrait", hero: 0.8, cover: 0.95, couple: true}),
		makeUpload(photoSpec{id: "rings", minute: 60, moment: "ring_exchange", hero: 0.9, cover: 0.4, couple: true}),
		makeUpload(photoSpec{id: "blur", minute: 61, moment: "vows", hero: 0.99, fit: dto.AlbumFitSkip, issues: []string{"blurry"}}),
		makeUpload(photoSpec{id: "cluttered", minute: 90, moment: "guests_candid", hero: 0.3, fit: dto.AlbumFitSkip, issues: []string{"cluttered"}}),
		makeUpload(photoSpec{id: "dance", minute: 300, moment: "first_dance", hero: 0.7, couple: true}),
		makeUpload(photoSpec{id: "late", minute: 400, moment: "send_off", hero: 0.4}),
	}
	acts := []*dto.StoryAct{
		{Label: dto.ActAnticipation, Title: "Anticipation", Order: 0, PhotoIDs: []string{"ready", "portrait"}},
		{Label: dto.ActCeremony, Title: "The Ceremony", Order: 1, PhotoIDs: []string{"rings", "blur", "cluttered"}},
		{Label: dto.ActFinalDance, Title: "The Last Dance", Order: 5, PhotoIDs: []string{"dance", "late"}},
	}

	brief, rules := buildPageBrief(testWedding(), acts, uploads)

	if brief.CoverPhotoID != "portrait" {
		t.Errorf("cover = %q, want the high cover-score couple frame", brief.CoverPhotoID)
	}
	if len(brief.PeakPhotoIDs) == 0 || brief.PeakPhotoIDs[0] != "rings" {
		t.Errorf("peak = %v, want rings first (the blurry vows frame must not win)", brief.PeakPhotoIDs)
	}
	if strings.Join(brief.ClosingPhotoIDs, ",") != "dance,late" {
		t.Errorf("closing = %v, want the first dance first", brief.ClosingPhotoIDs)
	}
	if strings.Join(rules.RequiredPhotoIDs, ",") != "rings" {
		t.Errorf("required = %v", rules.RequiredPhotoIDs)
	}

	ids := map[string]bool{}
	for _, p := range brief.Photos {
		ids[p.ID] = true
	}
	if ids["blur"] {
		t.Error("a blurry skip must stay out of the spreads")
	}
	if !ids["cluttered"] {
		t.Error("a merely cluttered guest photo must stay in")
	}
	// Every photo may still be referenced (the contact sheet shows them all).
	if _, ok := rules.KnownPhotoIDs["blur"]; !ok || len(rules.KnownPhotoIDs) != 7 {
		t.Errorf("known ids = %d, want all 7", len(rules.KnownPhotoIDs))
	}
	if len(brief.Chapters) != 3 || brief.Chapters[1].Title != "The Ceremony" {
		t.Errorf("chapters = %+v", brief.Chapters)
	}
	if brief.Wedding.Date != "12 December 2026" || brief.TotalPhotos != 7 || brief.MinFeatured < 1 {
		t.Errorf("brief header = %+v total=%d min=%d", brief.Wedding, brief.TotalPhotos, brief.MinFeatured)
	}
	if p := brief.Photos[0]; p.FocalPoint == nil || len(p.Palette) == 0 || p.Orientation != "landscape" {
		t.Errorf("photo evidence missing: %+v", p)
	}
}

func TestBuildPageBriefCapsLargeWeddingsAndSkipsBursts(t *testing.T) {
	var uploads []*dto.Upload
	var ceremony, party []string
	for i := 0; i < 400; i++ {
		id := fmt.Sprintf("c%d", i)
		// Ten bursts of identical frames among the ceremony photos.
		phash := fmt.Sprintf("p:%016x", uint64(i/40)*0x1111111111111111)
		uploads = append(uploads, makeUpload(photoSpec{id: id, minute: i / 10, moment: "vows", hero: float64(i%10) / 10, phash: phash}))
		ceremony = append(ceremony, id)
	}
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("r%d", i)
		uploads = append(uploads, makeUpload(photoSpec{id: id, minute: 200 + i, moment: "dancing", hero: 0.5}))
		party = append(party, id)
	}
	acts := []*dto.StoryAct{
		{Label: dto.ActCeremony, Title: "Ceremony", Order: 1, PhotoIDs: ceremony},
		{Label: dto.ActCelebration, Title: "Party", Order: 4, PhotoIDs: party},
	}

	brief, _ := buildPageBrief(testWedding(), acts, uploads)

	if len(brief.Photos) > maxBriefPhotos {
		t.Fatalf("brief has %d photos, cap is %d", len(brief.Photos), maxBriefPhotos)
	}
	fromCeremony := 0
	for _, p := range brief.Photos {
		if strings.HasPrefix(p.ID, "c") {
			fromCeremony++
		}
	}
	if fromCeremony > 10 {
		t.Errorf("%d ceremony photos chosen from 10 bursts of near-identical frames", fromCeremony)
	}
	if len(brief.Photos)-fromCeremony < 50 {
		t.Errorf("the party chapter should fill the remaining room, got %d", len(brief.Photos)-fromCeremony)
	}
	if brief.MinFeatured > maxRequiredFeature {
		t.Errorf("min featured = %d, cap is %d", brief.MinFeatured, maxRequiredFeature)
	}
}

func TestBuildPageBriefWithoutActsStillWorks(t *testing.T) {
	uploads := []*dto.Upload{
		makeUpload(photoSpec{id: "a", minute: 0, hero: 0.5, cover: 0.6, couple: true}),
		makeUpload(photoSpec{id: "b", minute: 5, hero: 0.7}),
	}
	brief, _ := buildPageBrief(testWedding(), nil, uploads)
	if len(brief.Photos) != 2 || len(brief.Chapters) != 1 || brief.CoverPhotoID != "a" {
		t.Errorf("brief = %d photos, %d chapters, cover %q", len(brief.Photos), len(brief.Chapters), brief.CoverPhotoID)
	}
}

func TestDesignPageRepairsThenAccepts(t *testing.T) {
	rules := pageRules()
	var calls []*pageRepair
	compose := func(_ context.Context, _ pageBrief, repair *pageRepair) (pageComposition, pageUsage, error) {
		calls = append(calls, repair)
		c := pageComposition{DesignRationale: "warm", HTML: validAlbumPage}
		if len(calls) == 1 {
			c.HTML = strings.Replace(validAlbumPage, "mv-photo://a1", "mv-photo://ghost", 1)
		}
		return c, pageUsage{Model: "gpt-test", InputTokens: 100, OutputTokens: 50, Latency: time.Second}, nil
	}
	page := &dto.AlbumPage{}

	designPage(context.Background(), compose, pageBrief{}, rules, 2, page)

	if page.Status != dto.AlbumPageReady || page.Attempts != 2 || page.HTML != validAlbumPage {
		t.Fatalf("page = %s after %d attempts", page.Status, page.Attempts)
	}
	if calls[1] == nil || !strings.Contains(strings.Join(calls[1].Issues, " "), "ghost") {
		t.Errorf("the repair call should carry the checker's problems, got %+v", calls[1])
	}
	if page.InputTokens != 200 || page.OutputTokens != 100 || page.LatencyMS != 2000 || page.Model != "gpt-test" {
		t.Errorf("usage should add up across attempts: %+v", page)
	}
}

func TestDesignPageFallsBackWithReasons(t *testing.T) {
	invalid := func(context.Context, pageBrief, *pageRepair) (pageComposition, pageUsage, error) {
		return pageComposition{HTML: "<p>not an album</p>"}, pageUsage{}, nil
	}
	page := &dto.AlbumPage{}
	designPage(context.Background(), invalid, pageBrief{}, pageRules(), 2, page)
	if page.Status != dto.AlbumPageFallback || page.FallbackReason != dto.AlbumPageInvalid || page.Attempts != 3 || len(page.Diagnostics) == 0 {
		t.Errorf("invalid pages: %s %s after %d attempts", page.Status, page.FallbackReason, page.Attempts)
	}

	quota := func(context.Context, pageBrief, *pageRepair) (pageComposition, pageUsage, error) {
		return pageComposition{}, pageUsage{}, errQuotaExhausted
	}
	page = &dto.AlbumPage{}
	designPage(context.Background(), quota, pageBrief{}, pageRules(), 2, page)
	if page.FallbackReason != dto.AlbumPageQuotaExhausted || page.Attempts != 1 {
		t.Errorf("quota: %s after %d attempts (should not retry)", page.FallbackReason, page.Attempts)
	}

	down := func(context.Context, pageBrief, *pageRepair) (pageComposition, pageUsage, error) {
		return pageComposition{}, pageUsage{}, errors.New("timeout")
	}
	page = &dto.AlbumPage{}
	designPage(context.Background(), down, pageBrief{}, pageRules(), 2, page)
	if page.FallbackReason != dto.AlbumPageUnavailable {
		t.Errorf("model error reason = %s", page.FallbackReason)
	}
}

// OpenAI rejects a strict schema unless every object lists all of its
// properties as required and forbids extra keys; the composition struct
// must also read every field.
func TestPageCompositionSchemaIsStrictAndDecodable(t *testing.T) {
	schema := pageCompositionSchema()
	props := schema["properties"].(map[string]any)
	if len(schema["required"].([]string)) != len(props) {
		t.Error("every top-level property must be required")
	}
	sample := `{"palette":[{"role":"paper","hex":"#f4ede2"}],"typography":{"display":"Cormorant","body":"Inter"},"design_rationale":"why","html":"<!doctype html>"}`
	var c pageComposition
	if err := json.Unmarshal([]byte(sample), &c); err != nil {
		t.Fatal(err)
	}
	if c.Palette[0].Hex != "#f4ede2" || c.Typography.Display != "Cormorant" || c.DesignRationale != "why" || c.HTML == "" {
		t.Errorf("decoded = %+v", c)
	}
}

func TestQuotaErrorDetection(t *testing.T) {
	if !isQuotaError(429, []byte(`{"error":{"code":"insufficient_quota"}}`)) {
		t.Error("insufficient_quota is a quota error")
	}
	if isQuotaError(429, []byte(`{"error":{"code":"rate_limit_exceeded"}}`)) {
		t.Error("a plain rate limit is not a quota error")
	}
}

func TestBuildPageBriefOpensAndPeaksOnDifferentFrames(t *testing.T) {
	uploads := []*dto.Upload{
		makeUpload(photoSpec{id: "portrait", minute: 0, moment: "couple_portrait", hero: 0.85, cover: 0.80, couple: true}),
		makeUpload(photoSpec{id: "dance", minute: 300, moment: "first_dance", hero: 0.88, cover: 0.82, couple: true}),
	}
	brief, rules := buildPageBrief(testWedding(), nil, uploads)
	if brief.CoverPhotoID != "portrait" {
		t.Errorf("cover = %q, want the portrait: the first dance belongs at the end", brief.CoverPhotoID)
	}
	if len(brief.PeakPhotoIDs) == 0 || brief.PeakPhotoIDs[0] != "dance" {
		t.Errorf("peak = %v, want the dance rather than the cover again", brief.PeakPhotoIDs)
	}
	if _, ok := rules.Subjects["portrait"]; !ok {
		t.Error("photos with people should carry where their subject is")
	}
}

package service

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/storyvows/backend/dto"
)

func sampleResult() photoAnalysisResult {
	r := photoAnalysisResult{
		Fit:               "keep",
		Issues:            []string{"none"},
		AestheticScore:    0.82,
		CoverScore:        0.9,
		HeroScore:         0.87,
		EmotionScore:      0.95,
		StudioProbability: 0.8,
		Moment:            "ring_exchange",
		EventType:         "ceremony",
		StoryPhase:        "ceremony",
		Scene:             "ceremony",
		IndoorOutdoor:     "outdoor",
		ShotStyle:         "candid",
		CouplePresent:     true,
		PeopleCount:       2,
		TextArea:          "top",
		ClothColors:       []string{"white", "gold"},
		AltText:           "The couple exchanging rings under a floral arch",
		Caption:           "The moment the room went quiet",
		CaptionWorthy:     true,
	}
	r.FocalPoint.X, r.FocalPoint.Y = 0.45, 0.38
	return r
}

func TestUploadFromPhotoAnalysisFillsAlbumAndLegacyScores(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	got := uploadFromPhotoAnalysis(sampleResult(), "gpt-test", now)

	album := got.Analysis.Album
	if album == nil {
		t.Fatal("expected album analysis")
	}
	if album.Version != albumAnalysisVersion || album.Model != "gpt-test" || !album.AnalysedAt.Equal(now) {
		t.Errorf("provenance = v%d %q %v", album.Version, album.Model, album.AnalysedAt)
	}
	if album.Moment != "ring_exchange" || !album.CouplePresent || album.FocalPoint.X != 0.45 {
		t.Errorf("album fields not mapped: %+v", album)
	}
	if len(album.Issues) != 0 {
		t.Errorf(`"none" should not be stored as an issue, got %v`, album.Issues)
	}
	if album.Caption == "" || !album.CaptionWorthy {
		t.Error("expected the caption to be kept")
	}

	// The album pipeline already ranks photos by these; they used to be empty.
	if got.Analysis.FeaturedScore == nil || *got.Analysis.FeaturedScore != 87 {
		t.Errorf("featured score = %v, want 87", got.Analysis.FeaturedScore)
	}
	if got.Analysis.QualityScore == nil || *got.Analysis.QualityScore != 0.82 {
		t.Errorf("quality score = %v, want 0.82", got.Analysis.QualityScore)
	}
	if got.Analysis.StudioProbability == nil || got.Analysis.EmotionScore == nil {
		t.Error("expected studio probability and emotion score")
	}
	if got.Category != dto.CategoryCeremony || got.Analysis.Category != dto.CategoryCeremony {
		t.Errorf("category = %q/%q, want ceremony", got.Category, got.Analysis.Category)
	}
	if got.Analysis.EventType != "ceremony" || len(got.Analysis.ClothColors) != 2 {
		t.Errorf("event type / cloth colours not kept: %q %v", got.Analysis.EventType, got.Analysis.ClothColors)
	}
	if usable, _ := got.AIInsights["is_usable"].(bool); !usable {
		t.Error("a keep photo should be usable")
	}
	if strings.Join(got.SceneTags, ",") != "outdoor,ceremony,ring_exchange" {
		t.Errorf("scene tags = %v", got.SceneTags)
	}
}

func TestUploadFromPhotoAnalysisNormalisesModelSlips(t *testing.T) {
	r := sampleResult()
	r.Fit = "unsure"
	r.HeroScore = 92 // answered as a percentage
	r.AestheticScore = -0.2
	r.CoverScore = 1.2 // slightly over, not a percentage
	r.CaptionWorthy = false
	r.PeopleCount = -3

	got := uploadFromPhotoAnalysis(r, "m", time.Now()).Analysis.Album

	if got.Fit != dto.AlbumFitMaybe {
		t.Errorf("unknown fit = %q, want maybe", got.Fit)
	}
	if got.HeroScore != 0.92 {
		t.Errorf("percentage hero score = %v, want 0.92", got.HeroScore)
	}
	if got.AestheticScore != 0 || got.CoverScore != 1 {
		t.Errorf("clamped scores = %v, %v; want 0, 1", got.AestheticScore, got.CoverScore)
	}
	if got.Caption != "" || got.CaptionWorthy {
		t.Error("a caption the model marked unworthy should be dropped")
	}
	if got.PeopleCount != 0 {
		t.Errorf("people count = %d, want 0", got.PeopleCount)
	}
}

func TestUploadFromPhotoAnalysisSkipIsUnusable(t *testing.T) {
	r := sampleResult()
	r.Fit = "skip"

	got := uploadFromPhotoAnalysis(r, "m", time.Now())

	if usable, _ := got.AIInsights["is_usable"].(bool); usable {
		t.Error("a skipped photo must not be marked usable")
	}
	if kept := preProcessUploads([]*dto.Upload{{ID: "a", IsApproved: true, Analysis: got.Analysis}}); len(kept) != 0 {
		t.Error("album pre-processing should drop skipped photos")
	}
}

// OpenAI rejects a strict schema unless every object lists all of its
// properties as required and forbids extra keys.
func TestWeddingPhotoSchemaIsStrict(t *testing.T) {
	var check func(path string, node map[string]any)
	check = func(path string, node map[string]any) {
		switch node["type"] {
		case "object":
			props, _ := node["properties"].(map[string]any)
			required, _ := node["required"].([]string)
			keys := make([]string, 0, len(props))
			for k := range props {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			sort.Strings(required)
			if strings.Join(keys, ",") != strings.Join(required, ",") {
				t.Errorf("%s: required %v does not match properties %v", path, required, keys)
			}
			if node["additionalProperties"] != false {
				t.Errorf("%s: additionalProperties must be false", path)
			}
			for k, v := range props {
				check(path+"."+k, v.(map[string]any))
			}
		case "array":
			check(path+"[]", node["items"].(map[string]any))
		}
	}
	check("$", weddingPhotoSchema())
}

// Every schema field must land in the Go struct, or the answer is silently lost.
func TestPhotoAnalysisResultCoversSchema(t *testing.T) {
	props := weddingPhotoSchema()["properties"].(map[string]any)
	tags := structJSONTags(photoAnalysisResult{})
	for key := range props {
		if !tags[key] {
			t.Errorf("schema property %q has no field in photoAnalysisResult", key)
		}
	}
}

func TestResponseOutputText(t *testing.T) {
	ok := `{"status":"completed","output":[{"content":[{"type":"output_text","text":"{\"fit\":\"keep\"}"}]}]}`
	if text, err := responseOutputText([]byte(ok)); err != nil || text != `{"fit":"keep"}` {
		t.Errorf("completed response = %q, %v", text, err)
	}

	refused := `{"status":"completed","output":[{"content":[{"type":"refusal","refusal":"I can't help with that."}]}]}`
	if _, err := responseOutputText([]byte(refused)); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Errorf("refusal should be an error, got %v", err)
	}

	cut := `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`
	if _, err := responseOutputText([]byte(cut)); err == nil || !strings.Contains(err.Error(), "max_output_tokens") {
		t.Errorf("incomplete response should name the reason, got %v", err)
	}
}

func TestHeroRankPrefersStrongCoupleFrames(t *testing.T) {
	album := func(hero float64, couple bool, fit string) *dto.Upload {
		return &dto.Upload{Analysis: dto.UploadAnalysis{Album: &dto.AlbumPhotoAnalysis{
			HeroScore: hero, EmotionScore: 0.5, AestheticScore: 0.5, CouplePresent: couple, Fit: fit,
		}}}
	}
	strong := album(0.9, true, dto.AlbumFitKeep)
	weak := album(0.5, false, dto.AlbumFitKeep)
	maybe := album(0.92, true, dto.AlbumFitMaybe)

	if heroRank(strong) <= heroRank(weak) {
		t.Error("a strong couple frame should outrank a weak one")
	}
	if heroRank(strong) <= heroRank(maybe) {
		t.Error(`a "maybe" frame should not carry a full page over a strong keeper`)
	}
	strong.ID, weak.ID = "strong", "weak"
	if got := selectHeroPhoto([]*dto.Upload{weak, strong}); got != "strong" {
		t.Errorf("hero = %q, want strong", got)
	}
}

func TestDerivativeKeyReplacesExtension(t *testing.T) {
	if got := derivativeKey("weddings/w1/abc.HEIC", "thumb"); got != "weddings/w1/abc_thumb.jpg" {
		t.Errorf("derivativeKey = %q", got)
	}
}

func structJSONTags(v any) map[string]bool {
	tags := map[string]bool{}
	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		name := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		tags[name] = true
	}
	return tags
}

func TestStoryPhasePinnedForUnambiguousMoments(t *testing.T) {
	r := sampleResult()
	r.Moment, r.StoryPhase = "first_dance", "celebration" // seen from gpt-4.1-mini
	if got := uploadFromPhotoAnalysis(r, "m", time.Now()).Analysis.Album.StoryPhase; got != "final_dance" {
		t.Errorf("first dance phase = %q, want final_dance", got)
	}
	r.Moment, r.StoryPhase = "details", "celebration"
	if got := uploadFromPhotoAnalysis(r, "m", time.Now()).Analysis.Album.StoryPhase; got != "celebration" {
		t.Errorf("details keep the model's phase, got %q", got)
	}
}

func TestMomentPhasesUseKnownValues(t *testing.T) {
	known := map[string]bool{}
	for _, m := range weddingMoments {
		known[m] = true
	}
	for moment := range momentPhases {
		if !known[moment] {
			t.Errorf("momentPhases has %q, which the schema never returns", moment)
		}
	}
}

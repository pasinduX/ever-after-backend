package service

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/integrations"
)

// TestLiveAlbumPage runs the whole chain on real photos: the image pipeline,
// the AI photo analysis, acts by story phase, the brief, and the art
// director with checks and repairs. It calls paid APIs, so it only runs on
// request, and it never touches the database:
//
//	MOONVEILS_LIVE_PAGE=1 MOONVEILS_LIVE_PAGE_OUT=/tmp/page.html go test ./service -run LiveAlbumPage -v -timeout 30m
//
// Photo IDs are the file names' last part (gallery-rings.jpg -> rings), the
// same IDs the frontend's /album-preview gives its bundled samples, so a
// saved page can be viewed there with ?page=<url of the saved file>.
func TestLiveAlbumPage(t *testing.T) {
	if os.Getenv("MOONVEILS_LIVE_PAGE") != "1" {
		t.Skip("set MOONVEILS_LIVE_PAGE=1 to call OpenAI")
	}
	_ = godotenv.Load("../.env")
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY is not set")
	}
	cfg := &integrations.Secrets{
		OpenAIAPIKey:        key,
		OpenAIModelAnalysis: firstNonEmpty(os.Getenv("OPENAI_MODEL_ANALYSIS"), os.Getenv("OPENAI_MODEL"), "gpt-4.1-mini"),
		OpenAIModelPage:     firstNonEmpty(os.Getenv("OPENAI_MODEL_PAGE"), "gpt-5.5"),
		AlbumPageEnabled:    true,
		AlbumPageRepairs:    2,
	}
	analysis := &AnalysisService{cfg: cfg, httpClient: &http.Client{Timeout: 90 * time.Second}}
	albums := &AlbumService{cfg: cfg, pageClient: &http.Client{Timeout: pageRequestTimeout + 30*time.Second}}

	dir := firstNonEmpty(os.Getenv("MOONVEILS_LIVE_PHOTOS"), "../../Moonveils/src/assets")
	paths, _ := filepath.Glob(filepath.Join(dir, "*.jpg"))
	if len(paths) == 0 {
		t.Skipf("no JPEGs in %s", dir)
	}

	wedding := &dto.Wedding{
		ID: "live", CoupleNames: dto.StringArray{"Julian", "Sofia"}, WeddingDate: weddingDay,
		Venue: "The Willow House", VenueType: "Outdoor Garden", CeremonyStyle: "Church ceremony",
		WeddingMood: "Romantic", WeddingTheme: "Garden Wedding", Lighting: "Golden Hour Sunset",
		Template: "golden-hour",
	}
	pctx := photoContext{CeremonyStyle: wedding.CeremonyStyle, VenueType: wedding.VenueType, Theme: wedding.WeddingTheme, ShotOn: "on or after the wedding day"}

	var uploads []*dto.Upload
	for i, path := range paths {
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		media := analyseImage(original, extractEXIF(original).Orientation)
		image := media.Medium
		if image == nil {
			image = original
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		analysed, err := analysis.AnalyzePhoto(ctx, "data:image/jpeg;base64,"+base64.StdEncoding.EncodeToString(image), pctx)
		cancel()
		if err != nil {
			t.Fatalf("analyse %s: %v", path, err)
		}
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		id := name[strings.LastIndex(name, "-")+1:]
		w, h := media.Dims.Width, media.Dims.Height
		at := weddingDay.Add(time.Duration(10+i) * time.Hour)
		analysed.ID, analysed.FileType, analysed.IsApproved, analysed.PHash = id, dto.FileTypePhoto, true, media.PHash
		analysed.Timeline = dto.UploadTimeline{CapturedAt: &at}
		analysed.Metadata = dto.UploadMetadata{Width: &w, Height: &h, Palette: media.Palette, Tone: media.Tone}
		uploads = append(uploads, analysed)
		t.Logf("%-8s %-16s phase=%-12s hero=%.2f cover=%.2f", id, analysed.Analysis.Album.Moment,
			analysed.Analysis.Album.StoryPhase, analysed.Analysis.Album.HeroScore, analysed.Analysis.Album.CoverScore)
	}

	brief, rules := buildPageBrief(wedding, actsByStoryPhase(uploads), uploads)
	t.Logf("brief: %d chapters, %d photos, cover=%s peak=%v closing=%v, min_featured=%d, max_words=%d",
		len(brief.Chapters), len(brief.Photos), brief.CoverPhotoID, brief.PeakPhotoIDs, brief.ClosingPhotoIDs,
		brief.MinFeatured, brief.MaxVisibleWords)

	page := &dto.AlbumPage{}
	ctx, cancel := context.WithTimeout(context.Background(), pageStageBudget)
	defer cancel()
	designPage(ctx, albums.composePage, brief, rules, cfg.AlbumPageRepairs, page)

	t.Logf("result: %s %s after %d attempts, model=%s, tokens in/out=%d/%d, %.0fs",
		page.Status, page.FallbackReason, page.Attempts, page.Model, page.InputTokens, page.OutputTokens,
		float64(page.LatencyMS)/1000)
	if page.Status != dto.AlbumPageReady {
		t.Fatalf("page fell back: %s %v", page.FallbackReason, page.Diagnostics)
	}
	t.Logf("fonts: %s / %s; palette: %v", page.DisplayFont, page.BodyFont, page.Palette)
	t.Logf("rationale: %s", page.DesignRationale)
	t.Logf("%d bytes, %d visible words", len(page.HTML), visibleWordCount(page.HTML))
	if out := os.Getenv("MOONVEILS_LIVE_PAGE_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(page.HTML), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("saved to %s", out)
	}
}

// actsByStoryPhase groups photos into acts by their analysed story phase,
// standing in for the album pipeline's time-based act assignment.
func actsByStoryPhase(uploads []*dto.Upload) []*dto.StoryAct {
	order := []dto.ActLabel{dto.ActAnticipation, dto.ActCeremony, dto.ActFamilyBonds, dto.ActJustTheTwo, dto.ActCelebration, dto.ActFinalDance}
	byLabel := map[dto.ActLabel]*dto.StoryAct{}
	var acts []*dto.StoryAct
	for i, label := range order {
		act := &dto.StoryAct{Label: label, Title: actTitle(label), Order: i}
		byLabel[label] = act
		acts = append(acts, act)
	}
	for _, u := range uploads {
		label := dto.ActLabel(strings.ToUpper(u.Analysis.Album.StoryPhase))
		if act, ok := byLabel[label]; ok {
			act.PhotoIDs = append(act.PhotoIDs, u.ID)
		} else {
			byLabel[dto.ActCelebration].PhotoIDs = append(byLabel[dto.ActCelebration].PhotoIDs, u.ID)
		}
	}
	return acts
}

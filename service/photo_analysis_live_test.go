package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/storyvows/backend/integrations"
)

// TestLiveWeddingPhotoAnalysis runs real photos through the image pipeline
// and the OpenAI analysis. It calls a paid API, so it only runs on request:
//
//	MOONVEILS_LIVE_ANALYSIS=1 go test ./service -run LiveWeddingPhotoAnalysis -v
//
// Point MOONVEILS_LIVE_PHOTOS at a folder of JPEGs; it defaults to the
// frontend's bundled sample photos.
func TestLiveWeddingPhotoAnalysis(t *testing.T) {
	if os.Getenv("MOONVEILS_LIVE_ANALYSIS") != "1" {
		t.Skip("set MOONVEILS_LIVE_ANALYSIS=1 to call OpenAI")
	}
	_ = godotenv.Load("../.env")
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY is not set")
	}
	model := os.Getenv("OPENAI_MODEL_ANALYSIS")
	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
	}
	if model == "" {
		model = "gpt-4.1-mini"
	}

	dir := os.Getenv("MOONVEILS_LIVE_PHOTOS")
	if dir == "" {
		dir = "../../Moonveils/src/assets"
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.jpg"))
	if len(paths) == 0 {
		t.Skipf("no JPEGs in %s", dir)
	}

	svc := &AnalysisService{
		cfg:        &integrations.Secrets{OpenAIAPIKey: key, OpenAIModelAnalysis: model},
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}
	pctx := photoContext{CeremonyStyle: "Kandyan (Sri Lankan)", VenueType: "Outdoor Garden", Theme: "Garden Wedding", ShotOn: "on or after the wedding day"}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			media := analyseImage(original, extractEXIF(original).Orientation)
			if !media.Decoded || len(media.Palette) == 0 {
				t.Fatalf("pipeline did not decode or measure %s", path)
			}
			image := media.Medium
			if image == nil {
				image = original
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			start := time.Now()
			got, err := svc.AnalyzePhoto(ctx, "data:image/jpeg;base64,"+base64.StdEncoding.EncodeToString(image), pctx)
			if err != nil {
				t.Fatalf("analysis failed: %v", err)
			}
			album := got.Analysis.Album
			if album == nil || album.Version != albumAnalysisVersion || album.AltText == "" {
				t.Fatalf("incomplete analysis: %+v", album)
			}
			out, _ := json.MarshalIndent(map[string]any{
				"seconds":  time.Since(start).Round(100 * time.Millisecond).Seconds(),
				"category": got.Category,
				"album":    album,
				"palette":  media.Palette,
				"tone":     media.Tone,
			}, "", "  ")
			t.Logf("%s", out)
		})
	}
}

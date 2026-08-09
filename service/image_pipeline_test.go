package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	return buf.Bytes()
}

func decodeDims(t *testing.T, data []byte) (int, int) {
	t.Helper()
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode derivative: %v", err)
	}
	return cfg.Width, cfg.Height
}

func TestAnalyseImageLandscapeProducesBoundedDerivatives(t *testing.T) {
	src := makeJPEG(t, 4000, 3000)

	got := analyseImage(src)

	if !got.Decoded {
		t.Fatal("expected image to decode")
	}
	if got.Dims.Width != 4000 || got.Dims.Height != 3000 {
		t.Errorf("dims = %dx%d, want 4000x3000", got.Dims.Width, got.Dims.Height)
	}
	if got.Dims.Orientation != "landscape" {
		t.Errorf("orientation = %q, want landscape", got.Dims.Orientation)
	}
	if got.PHash == "" {
		t.Error("expected a perceptual hash")
	}

	tw, th := decodeDims(t, got.Thumbnail)
	if tw != thumbnailMaxDim {
		t.Errorf("thumbnail width = %d, want %d", tw, thumbnailMaxDim)
	}
	if th > thumbnailMaxDim {
		t.Errorf("thumbnail height = %d, exceeds max %d", th, thumbnailMaxDim)
	}

	mw, _ := decodeDims(t, got.Medium)
	if mw != mediumMaxDim {
		t.Errorf("medium width = %d, want %d", mw, mediumMaxDim)
	}

	// The whole point: derivatives must be dramatically cheaper than the original.
	if len(got.Thumbnail) >= len(src) {
		t.Errorf("thumbnail (%d bytes) is not smaller than source (%d bytes)", len(got.Thumbnail), len(src))
	}
	if len(got.Medium) >= len(src) {
		t.Errorf("medium (%d bytes) is not smaller than source (%d bytes)", len(got.Medium), len(src))
	}
}

func TestAnalyseImagePortraitConstrainsLongEdge(t *testing.T) {
	got := analyseImage(makeJPEG(t, 1200, 3000))

	if !got.Decoded {
		t.Fatal("expected image to decode")
	}
	if got.Dims.Orientation != "portrait" {
		t.Errorf("orientation = %q, want portrait", got.Dims.Orientation)
	}

	_, th := decodeDims(t, got.Thumbnail)
	if th != thumbnailMaxDim {
		t.Errorf("portrait thumbnail height = %d, want %d (long edge should be capped)", th, thumbnailMaxDim)
	}
}

// A source already within budget shouldn't cost an extra S3 object.
func TestAnalyseImageSkipsDerivativesForSmallSource(t *testing.T) {
	got := analyseImage(makeJPEG(t, 300, 200))

	if !got.Decoded {
		t.Fatal("expected image to decode")
	}
	if got.Thumbnail != nil {
		t.Errorf("expected no thumbnail for a 300x200 source, got %d bytes", len(got.Thumbnail))
	}
	if got.Medium != nil {
		t.Errorf("expected no medium for a 300x200 source, got %d bytes", len(got.Medium))
	}
}

// Formats with no registered decoder (HEIC/AVIF) must degrade to "serve the
// original", not fail the upload.
func TestAnalyseImageUndecodableIsNotFatal(t *testing.T) {
	got := analyseImage([]byte("not an image"))

	if got.Decoded {
		t.Error("expected Decoded=false for undecodable input")
	}
	if got.Thumbnail != nil || got.Medium != nil {
		t.Error("expected no derivatives for undecodable input")
	}
	if got.PHash != "" {
		t.Error("expected no perceptual hash for undecodable input")
	}
}

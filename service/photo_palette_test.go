package service

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

func TestPaletteMergesNearShadesAndSurfacesAccent(t *testing.T) {
	t.Parallel()
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	blues := []color.RGBA{{0x50, 0x88, 0xc0, 255}, {0x48, 0x80, 0xb8, 255}, {0x58, 0x90, 0xc8, 255}}
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			switch {
			case x < 120: // 60% sky in three almost identical blues
				img.Set(x, y, blues[(x+y)%3])
			case x < 170: // 25% foliage
				img.Set(x, y, color.RGBA{0x2f, 0x5a, 0x2a, 255})
			case x < 190: // 10% deep shadow
				img.Set(x, y, color.RGBA{0x0a, 0x0b, 0x0c, 255})
			default: // 5% red subject
				img.Set(x, y, color.RGBA{0xd8, 0x30, 0x2a, 255})
			}
		}
	}
	colors, tone := paletteAndTone(img)
	if len(colors) != 4 {
		t.Fatalf("expected sky, foliage, shadow and red swatches, got %#v", colors)
	}
	if colors[0].Role != "ground" || colors[0].Share < 0.55 {
		t.Fatalf("the merged sky should be the ground swatch, got %#v", colors[0])
	}
	roles := map[string]string{}
	for _, swatch := range colors {
		roles[swatch.Role] = swatch.Hex
	}
	if roles["accent"] == "" || roles["shadow"] == "" {
		t.Fatalf("expected an accent and a shadow swatch, got %#v", colors)
	}
	var r, g, b int
	if _, err := fmtSscan(roles["accent"], &r, &g, &b); err != nil || r < 180 || g > 80 {
		t.Fatalf("the accent should be the red subject, got %s", roles["accent"])
	}
	if tone == nil || tone.Key != "mid" || tone.Saturation <= 0 {
		t.Fatalf("unexpected tone %#v", tone)
	}
}

func TestToneSeparatesWarmFromCoolPhotos(t *testing.T) {
	t.Parallel()
	fill := func(c color.RGBA) image.Image {
		img := image.NewRGBA(image.Rect(0, 0, 40, 40))
		for y := 0; y < 40; y++ {
			for x := 0; x < 40; x++ {
				img.Set(x, y, c)
			}
		}
		return img
	}
	_, warm := paletteAndTone(fill(color.RGBA{0xe0, 0x90, 0x40, 255}))
	_, cool := paletteAndTone(fill(color.RGBA{0x30, 0x70, 0xc0, 255}))
	_, night := paletteAndTone(fill(color.RGBA{0x10, 0x12, 0x1a, 255}))
	if warm.Temperature <= 0.3 || cool.Temperature >= -0.3 {
		t.Fatalf("expected warm > 0.3 and cool < -0.3, got %v and %v", warm.Temperature, cool.Temperature)
	}
	if night.Key != "low" || warm.Key == "low" {
		t.Fatalf("unexpected keys night=%s warm=%s", night.Key, warm.Key)
	}
}

func fmtSscan(hex string, r, g, b *int) (int, error) {
	return fmt.Sscanf(hex, "#%02x%02x%02x", r, g, b)
}

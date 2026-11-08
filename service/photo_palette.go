package service

import (
	"fmt"
	"image"
	"math"
	"sort"

	"github.com/storyvows/backend/dto"
)

const (
	paletteSampleSize    = 96
	paletteClusters      = 7
	paletteIterations    = 12
	paletteMergeDistance = 10.0 // CIE76 ΔE below which two swatches read as one colour
	paletteMinShare      = 0.015
	paletteMaxSwatches   = 6
	accentMinShare       = 0.02
	accentMinChroma      = 22.0
)

type labPixel struct {
	l, a, b  float64
	r, g, bl float64 // sRGB 0-255, averaged for swatch hex values
	chroma   float64
}

type paletteCluster struct {
	l, a, b  float64
	r, g, bl float64
	count    int
}

// Ported from VoyaLoom (internal/service/media_palette.go).
//
// paletteAndTone clusters a photo's colours in CIELAB, where distance tracks
// perceived difference, so three near-identical sky blues collapse into one
// swatch and a small vivid subject still surfaces as an accent. It is
// deterministic for the same pixels.
func paletteAndTone(src image.Image) ([]dto.PaletteSwatch, *dto.PhotoTone) {
	pixels := samplePixels(src)
	if len(pixels) == 0 {
		return nil, nil
	}
	clusters := kMeans(pixels, paletteClusters, paletteIterations)
	clusters = mergeSimilar(clusters)
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].count > clusters[j].count })

	total := float64(len(pixels))
	colors := make([]dto.PaletteSwatch, 0, paletteMaxSwatches)
	for _, cluster := range clusters {
		share := float64(cluster.count) / total
		if share < paletteMinShare || len(colors) == paletteMaxSwatches {
			continue
		}
		colors = append(colors, dto.PaletteSwatch{
			Hex:   fmt.Sprintf("#%02x%02x%02x", clampByte(cluster.r), clampByte(cluster.g), clampByte(cluster.bl)),
			Share: math.Round(share*1000) / 1000,
			Role:  toneRole(cluster.l),
		})
	}
	assignGroundAndAccent(colors, clusters, total)
	return colors, summarizeTone(pixels)
}

func samplePixels(src image.Image) []labPixel {
	bounds := src.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return nil
	}
	stepX := max(1, bounds.Dx()/paletteSampleSize)
	stepY := max(1, bounds.Dy()/paletteSampleSize)
	pixels := make([]labPixel, 0, paletteSampleSize*paletteSampleSize)
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r16, g16, b16, _ := src.At(x, y).RGBA()
			r, g, b := float64(r16>>8), float64(g16>>8), float64(b16>>8)
			l, a, bb := srgbToLab(r, g, b)
			pixels = append(pixels, labPixel{l: l, a: a, b: bb, r: r, g: g, bl: b, chroma: math.Hypot(a, bb)})
		}
	}
	return pixels
}

// kMeans seeds deterministically with farthest-point selection starting from
// the median-lightness pixel, then runs Lloyd iterations.
func kMeans(pixels []labPixel, k, iterations int) []paletteCluster {
	if len(pixels) < k {
		k = len(pixels)
	}
	byLightness := make([]int, len(pixels))
	for i := range byLightness {
		byLightness[i] = i
	}
	sort.Slice(byLightness, func(i, j int) bool { return pixels[byLightness[i]].l < pixels[byLightness[j]].l })
	centers := []paletteCluster{clusterFrom(pixels[byLightness[len(byLightness)/2]])}
	nearest := make([]float64, len(pixels))
	for i := range nearest {
		nearest[i] = math.MaxFloat64
	}
	for len(centers) < k {
		last := centers[len(centers)-1]
		farthest, farthestDistance := 0, -1.0
		for i, pixel := range pixels {
			if d := labDistance(pixel.l, pixel.a, pixel.b, last.l, last.a, last.b); d < nearest[i] {
				nearest[i] = d
			}
			if nearest[i] > farthestDistance {
				farthest, farthestDistance = i, nearest[i]
			}
		}
		if farthestDistance <= 0 {
			break
		}
		centers = append(centers, clusterFrom(pixels[farthest]))
	}

	assignments := make([]int, len(pixels))
	for iteration := 0; iteration < iterations; iteration++ {
		sums := make([]paletteCluster, len(centers))
		for i, pixel := range pixels {
			best, bestDistance := 0, math.MaxFloat64
			for c, center := range centers {
				if d := labDistance(pixel.l, pixel.a, pixel.b, center.l, center.a, center.b); d < bestDistance {
					best, bestDistance = c, d
				}
			}
			assignments[i] = best
			sums[best].add(pixel)
		}
		for c := range centers {
			if sums[c].count > 0 {
				centers[c] = sums[c].mean()
			} else {
				centers[c].count = 0
			}
		}
	}
	return centers
}

func mergeSimilar(clusters []paletteCluster) []paletteCluster {
	merged := make([]paletteCluster, 0, len(clusters))
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].count > clusters[j].count })
	for _, cluster := range clusters {
		if cluster.count == 0 {
			continue
		}
		absorbed := false
		for i := range merged {
			if labDistance(cluster.l, cluster.a, cluster.b, merged[i].l, merged[i].a, merged[i].b) < paletteMergeDistance {
				merged[i] = weightedMerge(merged[i], cluster)
				absorbed = true
				break
			}
		}
		if !absorbed {
			merged = append(merged, cluster)
		}
	}
	return merged
}

// assignGroundAndAccent marks the largest swatch as the ground and the most
// saturated swatch with a real presence as the accent, so designers can build
// a background and a highlight colour straight from the photo.
func assignGroundAndAccent(colors []dto.PaletteSwatch, clusters []paletteCluster, total float64) {
	if len(colors) == 0 {
		return
	}
	colors[0].Role = "ground"
	accent, accentChroma := -1, accentMinChroma
	for i := 1; i < len(colors) && i < len(clusters); i++ {
		share := float64(clusters[i].count) / total
		chroma := math.Hypot(clusters[i].a, clusters[i].b)
		if share >= accentMinShare && chroma > accentChroma {
			accent, accentChroma = i, chroma
		}
	}
	if accent > 0 {
		colors[accent].Role = "accent"
	}
}

func summarizeTone(pixels []labPixel) *dto.PhotoTone {
	var lightness, chroma, warmth, warmthWeight float64
	for _, pixel := range pixels {
		lightness += pixel.l
		chroma += pixel.chroma
		// Neutral pixels (white sky, grey stone) say little about warmth.
		if weight := math.Min(pixel.chroma/20, 1); weight > 0 {
			warmth += (pixel.b + 0.3*pixel.a) * weight
			warmthWeight += weight
		}
	}
	count := float64(len(pixels))
	tone := &dto.PhotoTone{
		Brightness: round3(lightness / count / 100),
		Saturation: round3(math.Min(chroma/count/50, 1)),
	}
	if warmthWeight > 0 {
		tone.Temperature = round3(math.Max(-1, math.Min(1, warmth/warmthWeight/30)))
	}
	switch {
	case tone.Brightness < 0.35:
		tone.Key = "low"
	case tone.Brightness > 0.68:
		tone.Key = "high"
	default:
		tone.Key = "mid"
	}
	return tone
}

func toneRole(lightness float64) string {
	switch {
	case lightness < 22:
		return "shadow"
	case lightness > 88:
		return "highlight"
	default:
		return "mid"
	}
}

func clusterFrom(pixel labPixel) paletteCluster {
	return paletteCluster{l: pixel.l, a: pixel.a, b: pixel.b, r: pixel.r, g: pixel.g, bl: pixel.bl, count: 1}
}

func (c *paletteCluster) add(pixel labPixel) {
	c.l += pixel.l
	c.a += pixel.a
	c.b += pixel.b
	c.r += pixel.r
	c.g += pixel.g
	c.bl += pixel.bl
	c.count++
}

func (c paletteCluster) mean() paletteCluster {
	n := float64(c.count)
	return paletteCluster{l: c.l / n, a: c.a / n, b: c.b / n, r: c.r / n, g: c.g / n, bl: c.bl / n, count: c.count}
}

func weightedMerge(x, y paletteCluster) paletteCluster {
	total := float64(x.count + y.count)
	wx, wy := float64(x.count)/total, float64(y.count)/total
	return paletteCluster{
		l: x.l*wx + y.l*wy, a: x.a*wx + y.a*wy, b: x.b*wx + y.b*wy,
		r: x.r*wx + y.r*wy, g: x.g*wx + y.g*wy, bl: x.bl*wx + y.bl*wy,
		count: x.count + y.count,
	}
}

func labDistance(l1, a1, b1, l2, a2, b2 float64) float64 {
	dl, da, db := l1-l2, a1-a2, b1-b2
	return math.Sqrt(dl*dl + da*da + db*db)
}

// srgbToLab converts 8-bit sRGB to CIELAB under a D65 white point.
func srgbToLab(r, g, b float64) (float64, float64, float64) {
	linear := func(channel float64) float64 {
		channel /= 255
		if channel <= 0.04045 {
			return channel / 12.92
		}
		return math.Pow((channel+0.055)/1.055, 2.4)
	}
	rl, gl, bl := linear(r), linear(g), linear(b)
	x := (0.4124*rl + 0.3576*gl + 0.1805*bl) / 0.95047
	y := 0.2126*rl + 0.7152*gl + 0.0722*bl
	z := (0.0193*rl + 0.1192*gl + 0.9505*bl) / 1.08883
	f := func(t float64) float64 {
		if t > 0.008856 {
			return math.Cbrt(t)
		}
		return 7.787*t + 16.0/116
	}
	fx, fy, fz := f(x), f(y), f(z)
	return 116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)
}

func clampByte(value float64) uint8 {
	return uint8(math.Max(0, math.Min(255, math.Round(value))))
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}

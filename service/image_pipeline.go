package service

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png" // register the PNG decoder for image.Decode

	"github.com/corona10/goimagehash"
	"github.com/nfnt/resize"
)

// Derivative sizes. The album grid and lightbox never need the full-resolution
// original, so serving one costs bandwidth and render time for nothing.
const (
	thumbnailMaxDim = 480
	thumbnailQuality = 78
	mediumMaxDim     = 1600
	mediumQuality    = 82
)

type imageDimensions struct {
	Width       int
	Height      int
	AspectRatio float64
	Orientation string
}

// mediaAnalysis is everything we derive from an uploaded image's pixels.
type mediaAnalysis struct {
	Decoded   bool
	Dims      imageDimensions
	PHash     string
	Thumbnail []byte // JPEG, nil when the source is already small enough
	Medium    []byte // JPEG, nil when the source is already small enough
}

// analyseImage decodes the image exactly once and derives dimensions, the
// perceptual hash and the downscaled derivatives from that single decode.
// Formats we have no decoder for (HEIC/AVIF, and WebP unless registered) return
// Decoded=false; callers fall back to serving the original.
func analyseImage(data []byte) mediaAnalysis {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return mediaAnalysis{}
	}

	bounds := img.Bounds()
	result := mediaAnalysis{
		Decoded: true,
		Dims:    dimensionsOf(bounds.Dx(), bounds.Dy()),
	}

	if h, err := goimagehash.PerceptionHash(img); err == nil {
		result.PHash = h.ToString()
	}

	result.Thumbnail = encodeScaled(img, thumbnailMaxDim, thumbnailQuality)
	result.Medium = encodeScaled(img, mediumMaxDim, mediumQuality)

	return result
}

func dimensionsOf(width, height int) imageDimensions {
	d := imageDimensions{Width: width, Height: height}
	if height > 0 {
		d.AspectRatio = float64(width) / float64(height)
	}
	if width >= height {
		d.Orientation = "landscape"
	} else {
		d.Orientation = "portrait"
	}
	return d
}

// encodeScaled downscales so the longest edge is at most maxDim, preserving
// aspect ratio. Returns nil when the source is already within budget, so we
// don't spend an S3 object storing a copy of the original.
func encodeScaled(img image.Image, maxDim uint, quality int) []byte {
	bounds := img.Bounds()
	width, height := uint(bounds.Dx()), uint(bounds.Dy())
	if width == 0 || height == 0 {
		return nil
	}
	if width <= maxDim && height <= maxDim {
		return nil
	}

	targetW, targetH := maxDim, uint(0)
	if height > width {
		targetW, targetH = 0, maxDim
	}

	scaled := resize.Resize(targetW, targetH, img, resize.Lanczos3)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: quality}); err != nil {
		return nil
	}
	return buf.Bytes()
}

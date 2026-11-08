package service

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png" // register the PNG decoder for image.Decode

	"github.com/corona10/goimagehash"
	"github.com/nfnt/resize"
	"github.com/storyvows/backend/dto"
	_ "golang.org/x/image/webp" // register the WebP decoder for image.Decode
)

// Derivative sizes. The album grid and lightbox never need the full-resolution
// original, so serving one costs bandwidth and render time for nothing.
const (
	thumbnailMaxDim  = 480
	thumbnailQuality = 78
	mediumMaxDim     = 1600
	mediumQuality    = 82
)

// imagePipelineVersion is stamped on every upload this pipeline processes.
// Bump it when derivatives, dimensions or palettes change in a way that old
// uploads should be rebuilt for; re-analysis rebuilds anything older.
//
//	1: EXIF orientation applied to derivatives and dimensions; measured palette.
const imagePipelineVersion = 1

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
	Palette   []dto.PaletteSwatch
	Tone      *dto.PhotoTone
	Thumbnail []byte // JPEG, nil when the source is already small enough
	Medium    []byte // JPEG, nil when the source is already small enough
}

// analyseImage decodes the image exactly once and derives dimensions, the
// perceptual hash, the measured palette and the downscaled derivatives.
//
// exifOrientation is the photo's EXIF orientation tag. Phones often store the
// pixels sideways and rely on that tag; our re-encoded JPEGs drop it, so the
// rotation is baked into the derivatives here. It is applied after scaling,
// which keeps rotating a 24MP photo from costing seconds per upload.
//
// Formats we have no decoder for (HEIC/AVIF) return Decoded=false; callers
// fall back to serving the original.
func analyseImage(data []byte, exifOrientation int) mediaAnalysis {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return mediaAnalysis{}
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if swapsAxes(exifOrientation) {
		width, height = height, width
	}
	result := mediaAnalysis{
		Decoded: true,
		Dims:    dimensionsOf(width, height),
	}

	// The working image is the display-oriented medium rendition (or the
	// whole photo when it is already small). Everything else derives from it.
	working := img
	if scaled := scaleToFit(img, mediumMaxDim); scaled != nil {
		working = applyEXIFOrientation(scaled, exifOrientation)
		result.Medium = encodeJPEG(working, mediumQuality)
	} else {
		working = applyEXIFOrientation(img, exifOrientation)
		// The original keeps its EXIF tag and browsers honour it, but a
		// sideways original still needs an upright copy for everything that
		// ignores EXIF, including the AI analysis.
		if exifOrientation >= 2 && exifOrientation <= 8 {
			result.Medium = encodeJPEG(working, mediumQuality)
		}
	}

	if thumb := scaleToFit(working, thumbnailMaxDim); thumb != nil {
		result.Thumbnail = encodeJPEG(thumb, thumbnailQuality)
	} else if result.Medium != nil {
		result.Thumbnail = result.Medium
	}

	if h, err := goimagehash.PerceptionHash(working); err == nil {
		result.PHash = h.ToString()
	}
	result.Palette, result.Tone = paletteAndTone(working)

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

// scaleToFit downscales so the longest edge is at most maxDim, preserving
// aspect ratio. Returns nil when the source is already within budget, so we
// don't spend an S3 object storing a copy of the original.
func scaleToFit(img image.Image, maxDim uint) image.Image {
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
	return resize.Resize(targetW, targetH, img, resize.Lanczos3)
}

func encodeJPEG(img image.Image, quality int) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil
	}
	return buf.Bytes()
}

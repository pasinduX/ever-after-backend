package service

import (
	"image"
)

// Ported from VoyaLoom (internal/service/image_orientation.go).
//
// applyEXIFOrientation returns the image as it should be displayed. Phones
// often store pixels rotated and rely on the EXIF orientation tag; re-encoded
// derivatives drop that tag, so the rotation must be baked into the pixels.
func applyEXIFOrientation(src image.Image, orientation int) image.Image {
	if orientation < 2 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := swapsAxes(orientation)
	dw, dh := w, h
	if swap {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch orientation {
			case 2: // mirror horizontal
				dx, dy = w-1-x, y
			case 3: // rotate 180
				dx, dy = w-1-x, h-1-y
			case 4: // mirror vertical
				dx, dy = x, h-1-y
			case 5: // transpose
				dx, dy = y, x
			case 6: // rotate 90 clockwise
				dx, dy = h-1-y, x
			case 7: // transverse
				dx, dy = h-1-y, w-1-x
			case 8: // rotate 90 counter-clockwise
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

// swapsAxes reports whether an EXIF orientation turns the image on its side,
// so the displayed width is the stored height.
func swapsAxes(orientation int) bool {
	return orientation >= 5 && orientation <= 8
}

package bitmap

import (
	"image"
)

func Encode(src image.Image) []byte {
	b := src.Bounds()
	dst := NewRGB565(src.Bounds())

	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			dst.Set(x, y, src.At(x, y))
		}
	}

	return dst.pixels
}

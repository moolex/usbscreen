package bitmap

import (
	"image"
)

func Encode(src image.Image) []byte {
	b := src.Bounds()
	d := NewRGB565(b)

	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			d.Set(x, y, src.At(x, y))
		}
	}

	return d.pixels
}

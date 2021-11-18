package bitmap

import (
	"image"
	"image/color"
)

// https://github.com/gonutz/framebuffer/blob/master/fb.go

func NewRGB565(r image.Rectangle) *RGB565 {
	return &RGB565{
		pixels:     make([]byte, pixelBufferLength(2, r, "RGB565")),
		stride:     2 * r.Dx(),
		bounds:     r,
		colorModel: rgb565Color{},
	}
}

// RGB565 represents the frame buffer. It implements the draw.Image interface.
type RGB565 struct {
	pixels     []byte
	stride     int
	bounds     image.Rectangle
	colorModel color.Model
}

// Bounds implements the image.Image (and draw.Image) interface.
func (d *RGB565) Bounds() image.Rectangle {
	return d.bounds
}

// ColorModel implements the image.Image (and draw.Image) interface.
func (d *RGB565) ColorModel() color.Model {
	return d.colorModel
}

// At implements the image.Image (and draw.Image) interface.
func (d *RGB565) At(x, y int) color.Color {
	if x < d.bounds.Min.X || x >= d.bounds.Max.X ||
		y < d.bounds.Min.Y || y >= d.bounds.Max.Y {
		return rgb565(0)
	}
	i := y*d.stride + 2*x
	return rgb565(d.pixels[i+1])<<8 | rgb565(d.pixels[i])
}

// Set implements the draw.Image interface.
func (d *RGB565) Set(x, y int, c color.Color) {
	// the min bounds are at 0,0 (see Open)
	if x >= 0 && x < d.bounds.Max.X &&
		y >= 0 && y < d.bounds.Max.Y {
		r, g, b, a := c.RGBA()
		if a > 0 {
			rgb := toRGB565(r, g, b)
			i := y*d.stride + 2*x
			// This assumes a little endian system which is the default for
			// Raspbian. The d.pixels indices have to be swapped if the target
			// system is big endian.
			d.pixels[i+1] = byte(rgb >> 8)
			d.pixels[i] = byte(rgb & 0xFF)
		}
	}
}

// The default color model under the Raspberry Pi is RGB 565. Each pixel is
// represented by two bytes, with 5 bits for red, 6 bits for green and 5 bits
// for blue. There is no alpha channel, so alpha is assumed to always be 100%
// opaque.
// This shows the memory layout of a pixel:
//
//    bit 76543210  76543210
//        RRRRRGGG  GGGBBBBB
//       high byte  low byte
type rgb565Color struct{}

func (rgb565Color) Convert(c color.Color) color.Color {
	r, g, b, _ := c.RGBA()
	return toRGB565(r, g, b)
}

// toRGB565 helps convert a color.Color to rgb565. In a color.Color each
// channel is represented by the lower 16 bits in a uint32 so the maximum value
// is 0xFFFF. This function simply uses the highest 5 or 6 bits of each channel
// as the RGB values.
func toRGB565(r, g, b uint32) rgb565 {
	// RRRRRGGGGGGBBBBB
	return rgb565((r & 0xF800) +
		((g & 0xFC00) >> 5) +
		((b & 0xF800) >> 11))
}

// rgb565 implements the color.Color interface.
type rgb565 uint16

// RGBA implements the color.Color interface.
func (c rgb565) RGBA() (r, g, b, a uint32) {
	// To convert a color channel from 5 or 6 bits back to 16 bits, the short
	// bit pattern is duplicated to fill all 16 bits.
	// For example the green channel in rgb565 is the middle 6 bits:
	//     00000GGGGGG00000
	//
	// To create a 16 bit channel, these bits are or-ed together starting at the
	// highest bit:
	//     GGGGGG0000000000 shifted << 5
	//     000000GGGGGG0000 shifted >> 1
	//     000000000000GGGG shifted >> 7
	//
	// These patterns map the minimum (all bits 0) and maximum (all bits 1)
	// 5 and 6 bit channel values to the minimum and maximum 16 bit channel
	// values.
	//
	// Alpha is always 100% opaque since this model does not support
	// transparency.
	rBits := uint32(c & 0xF800) // RRRRR00000000000
	gBits := uint32(c & 0x7E0)  // 00000GGGGGG00000
	bBits := uint32(c & 0x1F)   // 00000000000BBBBB
	r = uint32(rBits | rBits>>5 | rBits>>10 | rBits>>15)
	g = uint32(gBits<<5 | gBits>>1 | gBits>>7)
	b = uint32(bBits<<11 | bBits<<6 | bBits<<1 | bBits>>4)
	a = 0xFFFF
	return
}

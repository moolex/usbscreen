package mixer

import (
	"image"

	"github.com/samber/lo"

	"usbscreen/pkg/proto"
)

func NewDrawer(dst proto.Control, opts ...Option) *Drawer {
	d := &Drawer{
		dev: dst,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

type Drawer struct {
	dev  proto.Control
	effs []Effect
}

func (d *Drawer) Canvas(img image.Image) error {
	eff := lo.Sample(d.effs)
	if eff != nil {
		w, err := eff.Process(img.(Image))
		if err != nil {
			return err
		}

		for w2 := range w {
			if err := d.dev.DrawBitmap(uint16(w2.At.X), uint16(w2.At.Y), w2.Img); err != nil {
				return err
			}
		}
		return nil
	}

	return d.dev.DrawBitmap(0, 0, img)
}

package album

import (
	"errors"
	"fmt"

	"github.com/disintegration/imaging"
	"github.com/moolex/wallhaven-go/api"
	"github.com/moolex/wallhaven-go/utils"

	"usbscreen/pkg/proto"
)

func NewDrawer(dev proto.Control, album *Album) *Drawer {
	return &Drawer{dev: dev, album: album}
}

type Drawer struct {
	dev   proto.Control
	album *Album
}

func (d *Drawer) Drawing() error {
	wp, err := d.album.GetResult().Pick(api.PickLoop, api.PickRand)
	if err != nil {
		if errors.Is(err, api.ErrNoMoreItems) {
			d.album.UpdateQuery(func(q *api.QueryCond) { q.Page = 1 })
		}
		return fmt.Errorf("get wallpaper failed: %w", err)
	}

	d.album.SetWallpaper(wp)

	img, err := utils.GetThumbImage(wp, api.ThumbOriginal)
	if err != nil {
		return fmt.Errorf("get thumb image failed: %w", err)
	}

	img2 := imaging.Fill(img, d.album.width, d.album.height, imaging.Center, imaging.Lanczos)
	if err := d.dev.DrawBitmap(0, 0, img2); err != nil {
		return fmt.Errorf("drag bitmap failed: %w", err)
	}

	return nil
}

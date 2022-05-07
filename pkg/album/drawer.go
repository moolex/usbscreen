package album

import (
	"bytes"
	"fmt"
	"image"

	"github.com/disintegration/imaging"
	"github.com/moolex/wallhaven-go/api"

	"usbscreen/pkg/proto"
)

func NewDrawer(dev proto.Control, params *Params, cache *Cache, history *History) *Drawer {
	return &Drawer{
		dev:     dev,
		params:  params,
		cache:   cache,
		history: history,
	}
}

type Drawer struct {
	dev     proto.Control
	params  *Params
	cache   *Cache
	history *History
}

type wpFetcher func(wp *api.Wallpaper) (origin []byte, thumb bool, err error)
type cacheHandler func(wp *api.Wallpaper, thumb image.Image) error
type postHandler func(wp *api.Wallpaper, thumb bool, origin []byte) error

func (d *Drawer) Filled(wp *api.Wallpaper, f wpFetcher, c cacheHandler, h postHandler) (image.Image, error) {
	exists, cacheImg, err := d.cache.LoadImage(wp, d.params.width, d.params.height)
	if err != nil {
		return nil, fmt.Errorf("load cache failed: %w", err)
	}

	if exists {
		if c != nil {
			if err := c(wp, cacheImg); err != nil {
				return nil, fmt.Errorf("cache handler failed: %w", err)
			}
		}
		d.history.Add(wp, cacheImg, false, nil)
		return cacheImg, nil
	}

	origin, thumb, err := f(wp)
	if err != nil {
		return nil, fmt.Errorf("download image failed: %w", err)
	}

	if h != nil {
		if err := h(wp, thumb, origin); err != nil {
			return nil, fmt.Errorf("post handler failed: %w", err)
		}
	}

	img, _, err := image.Decode(bytes.NewBuffer(origin))
	if err != nil {
		return nil, fmt.Errorf("image decode failed: %w", err)
	}

	filled := imaging.Fill(img, d.params.width, d.params.height, imaging.Center, imaging.Lanczos)
	if !thumb {
		if err := d.cache.SaveImage(wp, filled); err != nil {
			return filled, fmt.Errorf("save cache failed: %w", err)
		}
	}

	d.history.Add(wp, filled, thumb, origin)
	return filled, nil
}

func (d *Drawer) Canvas(img image.Image) error {
	return d.dev.DrawBitmap(0, 0, img)
}

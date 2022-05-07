package album

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/moolex/wallhaven-go/api"
	"github.com/spf13/afero"
)

func NewCache(dir string) (*Cache, error) {
	c := &Cache{}

	if dir == "" {
		return c, nil
	}

	if fs, err := newFs(dir); err != nil {
		return nil, fmt.Errorf("create cache failed: %w", err)
	} else {
		c.fs = fs
	}

	return c, nil
}

type Cache struct {
	fs afero.Fs
}

func (c *Cache) dirname(wp *api.Wallpaper, w, h int) string {
	return fmt.Sprintf("%s-%s-%dx%d", wp.Category, wp.Purity, w, h)
}

func (c *Cache) filename(wp *api.Wallpaper, w, h int) string {
	return fmt.Sprintf("%s/%s.png", c.dirname(wp, w, h), wp.Id)
}

func (c *Cache) LoadImage(wp *api.Wallpaper, w, h int) (bool, image.Image, error) {
	if c.fs == nil {
		return false, nil, nil
	}

	bs, err := afero.ReadFile(c.fs, c.filename(wp, w, h))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		} else {
			return false, nil, err
		}
	}

	buf := bytes.NewBuffer(bs)
	img, err := png.Decode(buf)
	if err != nil {
		return false, nil, err
	}

	return true, img, nil
}

func (c *Cache) SaveImage(wp *api.Wallpaper, img image.Image) error {
	if c.fs == nil {
		return nil
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	dir := c.dirname(wp, w, h)
	file := c.filename(wp, w, h)

	if exists, err := afero.DirExists(c.fs, dir); err != nil {
		return err
	} else if !exists {
		if err2 := c.fs.MkdirAll(dir, 0755); err2 != nil {
			return err2
		}
	}

	return afero.WriteFile(c.fs, file, buf.Bytes(), 0644)
}

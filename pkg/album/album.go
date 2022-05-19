package album

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/moolex/wallhaven-go/api"
)

func New(params *Params, dl *Downloader, d *Drawer, opts ...Option) *Album {
	a := &Album{
		params: params,
		dl:     dl,
		d:      d,
		// options
		maxPage:  -1,
		maxSize:  -1,
		autoSave: nil,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

type Album struct {
	params *Params
	dl     *Downloader
	d      *Drawer
	// options
	maxPage  int
	maxSize  int
	autoSave *autoSave
}

func (a *Album) pickImage() (*api.Wallpaper, image.Image, error) {
	wp, err := a.params.GetResult().Pick(api.PickLoop)
	if err != nil {
		if errors.Is(err, api.ErrNoMoreItems) {
			a.params.UpdateQuery(func(q *api.QueryCond) { q.Page = 1 })
		}
		return nil, nil, fmt.Errorf("get wallpaper failed: %w", err)
	}

	if a.maxPage > 0 && a.params.GetQuery().Page >= a.maxPage && a.params.GetResult().Index() == 1 {
		a.params.UpdateQuery(func(q *api.QueryCond) { q.Page = 0 })
	}

	filled, err2 := a.d.Filled(
		wp,
		func(wp *api.Wallpaper) (*VFile, bool, error) {
			thumb := a.maxSize > 0 && wp.FileSize > a.maxSize
			vf, err := a.dl.Get(wp, thumb)
			return vf, thumb, err
		},
		func(wp *api.Wallpaper, thumb image.Image) error {
			if a.autoSave != nil && a.autoSave.Check(wp) {
				if exists, err := a.dl.Exists(wp); err == nil && !exists {
					// force save file
					_ = a.dl.Save(wp, nil)
				}
			}
			return nil
		},
		func(wp *api.Wallpaper, thumb bool, origin *VFile) error {
			if !thumb && a.autoSave != nil && a.autoSave.Check(wp) {
				if err := a.dl.Save(wp, origin); err != nil {
					return fmt.Errorf("auto save failed: %w", err)
				}
			}
			return nil
		},
	)

	return wp, filled, err2
}

func (a *Album) Drawing() error {
	if !a.d.TryLock() {
		return errors.New("drawer busying")
	}
	defer a.d.Unlock()

	_, filled, err := a.pickImage()

	if filled != nil {
		if err := a.d.Canvas(filled); err != nil {
			return fmt.Errorf("draw bitmap failed: %w", err)
		}
	}

	if err != nil {
		return fmt.Errorf("pick image failed: %w", err)
	}

	return nil
}

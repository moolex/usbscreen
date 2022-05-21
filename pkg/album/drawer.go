package album

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/moolex/wallhaven-go/api"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"usbscreen/pkg/mixer"
)

func NewDrawer(mixer *mixer.Drawer, params *Params, tmp *TmpFs, cache *Cache, history *History, logger *zap.Logger) *Drawer {
	return &Drawer{
		mixer:   mixer,
		params:  params,
		tmpfs:   tmp,
		cache:   cache,
		history: history,
		logger:  logger,
	}
}

type Drawer struct {
	sync.Mutex
	mixer   *mixer.Drawer
	params  *Params
	tmpfs   *TmpFs
	cache   *Cache
	history *History
	logger  *zap.Logger
}

type wpFetcher func(wp *api.Wallpaper) (origin *VFile, thumb bool, err error)
type preCache func(wp *api.Wallpaper, thumb image.Image) error
type postFetch func(wp *api.Wallpaper, thumb bool, origin *VFile) error

func (d *Drawer) Filled(wp *api.Wallpaper, fetcher wpFetcher, preCache preCache, postFetch postFetch) (image.Image, error) {
	exists, cacheImg, errL := d.cache.LoadImage(wp, d.params.width, d.params.height)
	if errL != nil {
		return nil, fmt.Errorf("load cache failed: %w", errL)
	}

	if exists {
		if preCache != nil {
			if err := preCache(wp, cacheImg); err != nil {
				return nil, fmt.Errorf("cache handler failed: %w", err)
			}
		}
		d.history.Add(wp, cacheImg, false, nil)
		return cacheImg, nil
	}

	origin, thumb, errG := fetcher(wp)
	if errG != nil {
		return nil, fmt.Errorf("download image failed: %w", errG)
	}

	if postFetch != nil {
		if err := postFetch(wp, thumb, origin); err != nil {
			return nil, fmt.Errorf("post handler failed: %w", err)
		}
	}

	filled, errF := lo.Ternary(origin.IsFile(), d.byIMagick, d.byLocal)(origin)
	if errF != nil {
		return nil, errF
	}

	if !thumb {
		if err := d.cache.SaveImage(wp, filled); err != nil {
			return filled, fmt.Errorf("save cache failed: %w", err)
		}
	}

	d.history.Add(wp, filled, thumb, origin)
	return filled, nil
}

func (d *Drawer) byLocal(vf *VFile) (image.Image, error) {
	bs, err := vf.Bytes()
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewBuffer(bs))
	if err != nil {
		return nil, fmt.Errorf("image decode failed: %w", err)
	}

	return imaging.Fill(img, d.params.width, d.params.height, imaging.Center, imaging.Lanczos), nil
}

func (d *Drawer) byIMagick(vf *VFile) (image.Image, error) {
	tmp := d.tmpfs.NewFile()
	if tmp == "" {
		return nil, errors.New("no tmpfs supported")
	}
	tmp += ".png"

	wh := fmt.Sprintf("%dx%d", d.params.width, d.params.height)
	src := vf.Filepath()

	cmd := exec.Command(
		"convert",
		src,
		"-filter", "lanczos",
		"-resize", fmt.Sprintf("%s^", wh),
		"-gravity", "Center",
		"-extent", wh,
		tmp,
	)
	if bs, err := cmd.CombinedOutput(); err != nil {
		d.logger.With(zap.String("exec", cmd.String()), zap.Error(err)).Info("failed")
		fmt.Println(string(bs))
		return nil, err
	}

	d.logger.With(zap.String("by", "imagick"), zap.String("src", src), zap.String("dst", tmp)).Debug("converted")

	f, err := os.Open(tmp)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}()

	return png.Decode(f)
}

func (d *Drawer) Canvas(img image.Image) error {
	return d.mixer.Canvas(img)
}

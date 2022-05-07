package album

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"

	"github.com/go-resty/resty/v2"
	"github.com/moolex/wallhaven-go/api"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/afero"
	"go.uber.org/zap"
)

func NewDownloader(dir string, logger *zap.Logger) (*Downloader, error) {
	d := &Downloader{
		cli: resty.New().SetDoNotParseResponse(true),
		log: logger,
	}

	if dir == "" {
		return d, nil
	}

	if fs, err := newFs(dir); err != nil {
		return nil, fmt.Errorf("create downloader failed: %w", err)
	} else {
		d.fs = fs
	}

	return d, nil
}

type Downloader struct {
	fs  afero.Fs
	cli *resty.Client
	log *zap.Logger
}

func (d *Downloader) folder(wp *api.Wallpaper) string {
	return fmt.Sprintf("%s-%s", wp.Category, wp.Purity)
}

func (d *Downloader) filename(wp *api.Wallpaper) string {
	u, _ := url.Parse(wp.Path)
	return fmt.Sprintf("%s/%s", d.folder(wp), path.Base(u.Path))
}

func (d *Downloader) Exists(wp *api.Wallpaper) (bool, error) {
	return afero.Exists(d.fs, d.filename(wp))
}

func (d *Downloader) Get(wp *api.Wallpaper, thumb bool) ([]byte, error) {
	if d.fs != nil {
		file := d.filename(wp)
		if exists, err := afero.Exists(d.fs, file); err != nil {
			return nil, err
		} else if exists {
			return afero.ReadFile(d.fs, file)
		}
	}

	resp, err := d.cli.R().Get(lo.Ternary(thumb, wp.Thumbs.Original, wp.Path))
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.RawBody().Close()
	}()

	bar := progressbar.DefaultBytes(resp.RawResponse.ContentLength, fmt.Sprintf("Downloading %s", wp.Url))

	var buf bytes.Buffer
	if _, err := io.Copy(io.MultiWriter(&buf, bar), resp.RawBody()); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (d *Downloader) Save(wp *api.Wallpaper, bs []byte) error {
	dir := d.folder(wp)
	file := d.filename(wp)

	if exists, err := afero.Exists(d.fs, file); err != nil {
		return err
	} else if exists {
		return errors.New("already saved")
	}

	if exists, err := afero.DirExists(d.fs, dir); err != nil {
		return err
	} else if !exists {
		if err2 := d.fs.MkdirAll(dir, 0755); err2 != nil {
			return err2
		}
	}

	if len(bs) == 0 {
		var err error
		bs, err = d.Get(wp, false)
		if err != nil {
			return fmt.Errorf("re-download failed: %w", err)
		}
	}

	if err := afero.WriteFile(d.fs, file, bs, 0644); err != nil {
		return err
	}

	d.log.With(zap.String("url", wp.Url)).Debug("wallpaper saved")
	return nil
}

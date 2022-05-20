package album

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"path"

	"github.com/go-resty/resty/v2"
	"github.com/moolex/wallhaven-go/api"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/afero"
	"go.uber.org/zap"
)

func NewDownloader(dir string, tmp *TmpFs, logger *zap.Logger) (*Downloader, error) {
	d := &Downloader{
		tmp: tmp,
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
	tmp *TmpFs
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

func (d *Downloader) Get(wp *api.Wallpaper, thumb bool) (*VFile, error) {
	if d.fs != nil {
		file := d.filename(wp)
		if exists, err := afero.Exists(d.fs, file); err != nil {
			return nil, err
		} else if exists {
			return newFile(file, d.fs), nil
		}
	}

	tmp := d.tmp.NewFile()
	if !thumb && tmp != "" {
		return d.curlGet(wp.Path, tmp)
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

	return newBytes(buf.Bytes()), nil
}

func (d *Downloader) curlGet(url string, save string) (*VFile, error) {
	cmd := exec.Command("curl", "-s", url, "-o", save)
	if bs, err := cmd.CombinedOutput(); err != nil {
		d.log.With(zap.String("exec", cmd.String()), zap.Error(err)).Info("failed")
		fmt.Println(string(bs))
		return nil, err
	}
	d.log.With(zap.String("by", "curl"), zap.String("file", save)).Debug("downloaded")
	return newFile(save, nil), nil
}

func (d *Downloader) Save(wp *api.Wallpaper, vf *VFile) error {
	dir := d.folder(wp)
	file := d.filename(wp)

	if exists, err := afero.Exists(d.fs, file); err != nil {
		return err
	} else if exists {
		return nil
	}

	if exists, err := afero.DirExists(d.fs, dir); err != nil {
		return err
	} else if !exists {
		if err2 := d.fs.MkdirAll(dir, 0755); err2 != nil {
			return err2
		}
	}

	var dat []byte
	var err error

	if vf != nil {
		dat, err = vf.Bytes()
	}
	if err != nil {
		return err
	}

	if len(dat) == 0 {
		var errR error
		if vf, errR = d.Get(wp, false); errR != nil {
			return fmt.Errorf("re-download failed: %w", errR)
		} else if dat, errR = vf.Bytes(); errR != nil {
			return errR
		}
	}

	if err := afero.WriteFile(d.fs, file, dat, 0644); err != nil {
		return err
	}

	d.log.With(zap.String("url", wp.Url)).Debug("wallpaper saved")
	return nil
}

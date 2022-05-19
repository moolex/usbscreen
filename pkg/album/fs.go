package album

import (
	"errors"
	"fmt"

	"github.com/rs/xid"
	"github.com/spf13/afero"
)

func newFs(path string) (afero.Fs, error) {
	fs := afero.NewOsFs()
	if exists, err := afero.DirExists(fs, path); err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.New("dir not exists")
	}
	return afero.NewBasePathFs(fs, path), nil
}

func NewTmpFs(dir string) (*TmpFs, error) {
	t := &TmpFs{}

	if dir == "" {
		return t, nil
	}

	if fs, err := newFs(dir); err != nil {
		return nil, fmt.Errorf("create tmpdir failed: %w", err)
	} else {
		t.fs = fs
	}

	return t, nil
}

type TmpFs struct {
	fs afero.Fs
}

func (t *TmpFs) NewFile() string {
	if t.fs != nil {
		p, _ := t.fs.(*afero.BasePathFs).RealPath(xid.New().String())
		return p
	}
	return ""
}

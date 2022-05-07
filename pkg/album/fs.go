package album

import (
	"errors"

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

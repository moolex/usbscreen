package album

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/spf13/afero"
)

func newFile(path string, fs afero.Fs) *VFile {
	return &VFile{fs: fs, path: path}
}

func newBytes(bs []byte) *VFile {
	return &VFile{bytes: bs}
}

type VFile struct {
	sync.Mutex
	fs    afero.Fs
	path  string
	bytes []byte
	freed bool
}

func (v *VFile) IsFile() bool {
	return v.fs != nil || v.path != ""
}

func (v *VFile) Filepath() string {
	if v.fs != nil {
		p, _ := v.fs.(*afero.BasePathFs).RealPath(v.path)
		return p
	}
	return v.path
}

func (v *VFile) Free() error {
	v.Lock()
	defer v.Unlock()

	if v.freed {
		return nil
	}

	var err error
	if v.fs != nil {
		err = v.fs.Remove(v.path)
	} else if v.path != "" {
		err = os.Remove(v.path)
	}

	if err == nil {
		v.freed = true
	}

	return err
}

func (v *VFile) Bytes() ([]byte, error) {
	if len(v.bytes) > 0 {
		return v.bytes, nil
	}

	var bs []byte
	var err error

	if v.fs != nil {
		bs, err = afero.ReadFile(v.fs, v.path)
	} else if v.path != "" {
		bs, err = ioutil.ReadFile(v.path)
	} else {
		err = errors.New("no file to read")
	}

	if err != nil {
		err = fmt.Errorf("vfile read failed: %w", err)
	}

	return bs, err
}

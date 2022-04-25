package inch35

import (
	"bytes"
	"encoding/binary"
	"image"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"usbscreen/pkg/bitmap"
	"usbscreen/pkg/proto"
)

const (
	Restart    = 101
	Shutdown   = 108
	Startup    = 109
	SetLight   = 110
	SetRotate  = 121
	SetMirror  = 122
	DrawPixels = 195
	DrawBitmap = 197
	TESTING    = 255
)

func New(serial *proto.Serial, logger *zap.Logger) (proto.Control, error) {
	dev := &Inch35{
		serial: serial,
		logger: logger,
		width:  320,
		height: 480,
	}
	return dev, serial.Open(&proto.Options{
		DTR:         true,
		RTS:         true,
		BaudRate:    115200,
		ReadTimeout: time.Millisecond,
	})
}

type Inch35 struct {
	serial *proto.Serial
	logger *zap.Logger
	width  int
	height int
}

func (i *Inch35) Startup() error {
	return i.sendCMD(Startup)
}

func (i *Inch35) Shutdown() error {
	return i.sendCMD(Shutdown)
}

func (i *Inch35) Restart() error {
	return i.sendCMD(Restart)
}

func (i *Inch35) SetLight(light uint8) error {
	return i.sendCMD(SetLight, int(light))
}

func (i *Inch35) SetRotate(landscape bool, invert bool) error {
	ow := i.width
	oh := i.height

	ov := 100
	if landscape {
		ov++
		i.width = oh
		i.height = ow
		if invert {
			ov++
		}
	} else if invert {
		ov++
	}

	var bs bytes.Buffer
	bs.WriteByte(uint8(ov))
	_ = binary.Write(&bs, binary.BigEndian, uint16(i.width))
	_ = binary.Write(&bs, binary.BigEndian, uint16(i.height))

	return i.sendOpt(SetRotate, 16, bs.Bytes())
}

func (i *Inch35) SetMirror(mirror bool) error {
	var b byte
	if mirror {
		b = 1
	}

	return i.sendOpt(SetMirror, 16, []byte{b})
}

func (i *Inch35) DrawBitmap(posX uint16, posY uint16, image image.Image) error {
	rect := image.Bounds().Size()
	imgW := rect.X
	imgH := rect.Y

	if imgW+int(posX) > i.width {
		return errors.New("width overflow")
	} else if imgH+int(posY) > i.height {
		return errors.New("height overflow")
	}

	if err := i.sendCMD(DrawBitmap, int(posX), int(posY), int(posX)+imgW-1, int(posY)+imgH-1); err != nil {
		return err
	}

	bmp := bitmap.Encode(image)
	if err := i.sendBytes(bmp); err != nil {
		return err
	}

	return nil
}

package device

import (
	"bytes"
	"encoding/binary"
	"image"
	"time"

	"github.com/pkg/errors"

	"usbscreen/internal/bitmap"
	"usbscreen/internal/proto"
)

const (
	RESTART     = 101
	SHUTDOWN    = 108
	STARTUP     = 109
	SET_LIGHT   = 110
	SET_ROTATE  = 121
	SET_MIRROR  = 122
	DRAW_PIXELS = 195
	DRAW_BITMAP = 197
	TESTING     = 255
)

func NewInch35(serial *proto.Serial) (proto.Control, error) {
	dev := &Inch35{
		serial: serial,
		width:  320,
		height: 480,
	}
	return dev, dev.init()
}

type Inch35 struct {
	serial *proto.Serial
	width  int
	height int
}

func (i *Inch35) init() error {
	port, err := i.serial.Open(&proto.Options{BaudRate: 115200})
	if err != nil {
		return err
	}

	if err := port.SetDTR(true); err != nil {
		return err
	}

	if err := port.SetRTS(true); err != nil {
		return err
	}

	if err := port.SetReadTimeout(10 * time.Millisecond); err != nil {
		return err
	}

	return nil
}

func (i *Inch35) Startup() error {
	return i.sendCMD(STARTUP)
}

func (i *Inch35) Shutdown() error {
	return i.sendCMD(SHUTDOWN)
}

func (i *Inch35) Restart() error {
	return i.sendCMD(RESTART)
}

func (i *Inch35) SetLight(light uint8) error {
	return i.sendCMD(SET_LIGHT, int(light))
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

	return i.sendOpt(SET_ROTATE, 16, bs.Bytes())
}

func (i *Inch35) SetMirror(mirror bool) error {
	var b byte
	if mirror {
		b = 1
	}

	return i.sendOpt(SET_MIRROR, 16, []byte{b})
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

	if err := i.sendCMD(DRAW_BITMAP, int(posX), int(posY), int(posX)+imgW-1, int(posY)+imgH-1); err != nil {
		return err
	}

	bmp := bitmap.Encode(image)
	if err := i.sendBytes(bmp); err != nil {
		return err
	}

	return nil
}

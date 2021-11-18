package proto

import (
	"image"
)

type Control interface {
	Startup() error
	Shutdown() error
	Restart() error

	SetLight(light uint8) error
	SetMirror(mirror bool) error
	SetRotate(landscape bool, invert bool) error

	DrawBitmap(posX uint16, posY uint16, image image.Image) error
	// DrawPixels(offsetX uint16, offsetY uint16, color color.Color, coordinates []uint8) error
}

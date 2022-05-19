package virtual

import (
	"image"

	"go.uber.org/zap"

	"usbscreen/pkg/proto"
)

func Mock(logger *zap.Logger) proto.Control {
	return &Mocker{logger}
}

type Mocker struct {
	l *zap.Logger
}

func (m *Mocker) Startup() error {
	m.l.Info("startup")
	return nil
}

func (m *Mocker) Shutdown() error {
	m.l.Info("shutdown")
	return nil
}

func (m *Mocker) Restart() error {
	m.l.Info("restart")
	return nil
}

func (m *Mocker) SetLight(light uint8) error {
	m.l.With(zap.Uint8("light", light)).Info("set-light")
	return nil
}

func (m *Mocker) SetMirror(mirror bool) error {
	m.l.With(zap.Bool("mirror", mirror)).Info("set-mirror")
	return nil
}

func (m *Mocker) SetRotate(landscape bool, invert bool) error {
	m.l.With(zap.Bool("landscape", landscape), zap.Bool("invert", invert)).Info("set-rotate")
	return nil
}

func (m *Mocker) DrawBitmap(posX uint16, posY uint16, image image.Image) error {
	m.l.With(
		zap.Uint16("x", posX),
		zap.Uint16("y", posY),
		zap.Int("w", image.Bounds().Dx()),
		zap.Int("h", image.Bounds().Dy()),
	).Info("draw-bitmap")
	return nil
}

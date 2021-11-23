package remote

import (
	"bytes"
	"context"
	"image/png"
	"log"
	"net/http"
	"net/rpc"

	"github.com/pkg/errors"
	"go.uber.org/fx"

	"usbscreen/pkg/proto"
)

func Proxy(dev proto.Control, srv *http.Server, lifecycle fx.Lifecycle) error {
	svc := &Service{dev: dev}
	if err := rpc.Register(svc); err != nil {
		return err
	}

	rpc.HandleHTTP()

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != http.ErrServerClosed {
					log.Fatal(err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})

	return nil
}

type Service struct {
	dev proto.Control
}

func (s *Service) Command(name string, _ *EmptyResponse) error {
	switch name {
	case "startup":
		return s.dev.Startup()
	case "shutdown":
		return s.dev.Shutdown()
	case "restart":
		return s.dev.Restart()
	}

	return errors.New("unknown command")
}

func (s *Service) SetLight(light uint8, _ *EmptyResponse) error {
	return s.dev.SetLight(light)
}

func (s *Service) SetMirror(mirror bool, _ *EmptyResponse) error {
	return s.dev.SetMirror(mirror)
}

func (s *Service) SetRotate(req SetRotateRequest, _ *EmptyResponse) error {
	return s.dev.SetRotate(req.Landscape, req.Invert)
}

func (s *Service) DrawBitmap(req *DrawBitmapRequest, _ *EmptyResponse) error {
	img, err := png.Decode(bytes.NewBuffer(req.Image))
	if err != nil {
		return err
	}

	return s.dev.DrawBitmap(req.PosX, req.PosY, img)
}

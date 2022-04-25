package main

import (
	"net/http"

	flag "github.com/spf13/pflag"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"usbscreen/pkg/device/inch35"
	"usbscreen/pkg/device/remote"
	"usbscreen/pkg/proto"
)

var serial = flag.String("serial", "ttyACM0", "serial name")
var listen = flag.String("listen", ":9123", "listen addr")

func main() {
	flag.Parse()

	fx.New(
		fx.Provide(
			func() *proto.Serial {
				return proto.NewSerial(*serial)
			},
			func() *http.Server {
				return &http.Server{Addr: *listen}
			},
			func() *zap.Logger {
				l, _ := zap.NewDevelopment()
				return l
			},
			inch35.New,
		),
		fx.Invoke(
			remote.Proxy,
		),
	).Run()
}

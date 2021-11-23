package main

import (
	"net/http"

	flag "github.com/spf13/pflag"
	"go.uber.org/fx"

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
			func() (*proto.Serial, *http.Server) {
				return proto.NewSerial(*serial),
					&http.Server{Addr: *listen}
			},
			inch35.New,
		),
		fx.Invoke(
			remote.Proxy,
		),
	).Run()
}

package main

import (
	"usbscreen/internal/device"
	"usbscreen/internal/proto"
	"usbscreen/internal/utils"
)

func main() {
	serial := proto.NewSerial("usbmodemUSB35INCHIPSV21")
	dev, err := device.NewInch35(serial)
	if err != nil {
		panic(err)
	}

	if err := dev.Startup(); err != nil {
		panic(err)
	}

	if err := dev.SetMirror(false); err != nil {
		panic(err)
	}

	if err := dev.SetRotate(false, true); err != nil {
		panic(err)
	}

	if err := dev.SetLight(100); err != nil {
		panic(err)
	}

	img, err := utils.OpenImage("/Users/moyo/Downloads/35inchNEW/3.5/back0.png")
	if err != nil {
		panic(err)
	}

	if err := dev.DrawBitmap(0, 0, img); err != nil {
		panic(err)
	}
}

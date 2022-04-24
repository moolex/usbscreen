package main

import (
	"log"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/moolex/wallhaven-go/api"
	"github.com/moolex/wallhaven-go/utils"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	"usbscreen/pkg/device/inch35"
	"usbscreen/pkg/device/remote"
	"usbscreen/pkg/proto"
)

var serial = flag.String("serial", "ttyACM0", "serial name or remote addr")
var light = flag.Uint8("light", 100, "set light")
var landscape = flag.Bool("landscape", false, "set landscape")
var invert = flag.Bool("invert", false, "set invert")
var interval = flag.String("interval", "5m", "draw interval")
var debug = flag.Bool("debug", false, "set debug")
var whKey = flag.String("wh-key", "", "wallhaven api key")
var whQuery = flag.String("wh-query", "", "wallhaven query string")
var whCategory = flag.String("wh-category", "", "wallhaven category names")
var whPurity = flag.String("wh-purity", "", "wallhaven purity levels")
var whSorting = flag.String("wh-sorting", "", "wallhaven sorting type")
var whToplist = flag.String("wh-toplist", "1M", "wallhaven toplist range")
var whRatio = flag.String("wh-ratio", "", "wallhaven ratio filter")

func main() {
	flag.Parse()

	errorWait := 3 * time.Second
	var changeWait time.Duration
	if d, err := time.ParseDuration(*interval); err != nil {
		log.Fatal(err)
	} else {
		changeWait = d
	}

	var dev proto.Control
	var err error

	if strings.Contains(*serial, ":") {
		dev, err = remote.New(*serial)
	} else {
		dev, err = inch35.New(proto.NewSerial(*serial))
	}

	if err != nil {
		log.Fatal(err)
	}

	if err := dev.Startup(); err != nil {
		log.Fatal(err)
	}

	if err := dev.SetLight(*light); err != nil {
		log.Fatal(err)
	}

	if err := dev.SetRotate(*landscape, *invert); err != nil {
		log.Fatal(err)
	}

	width := 320
	height := 480
	if *landscape {
		width = 480
		height = 320
	}

	logger, _ := zap.NewDevelopment()

	wh := api.New(*whKey)
	wh.SetLogger(logger)
	if *debug {
		wh.SetDebug()
	}

	q := api.NewQuery(*whQuery)
	if whCategory != nil {
		q.SetCategory(strings.Split(*whCategory, ",")...)
	}
	if whPurity != nil {
		q.SetPurity(strings.Split(*whPurity, ",")...)
	}
	if whRatio != nil {
		q.SetRatio(*whRatio)
	}
	if whSorting != nil && *whSorting != "" {
		q.SortBy(*whSorting)
	} else if whToplist != nil {
		q.SortBy(api.SortTopList)
		q.TopRange = *whToplist
	}

	ret, err := wh.Query(q)
	if err != nil {
		log.Fatal(err)
	}

	var errors uint

	for {
		if errors > 100 {
			q.Page = 1
			if retN, err := wh.Query(q); err != nil {
				time.Sleep(errorWait)
				continue
			} else {
				errors = 0
				ret = retN
			}
		}

		wp, err := ret.Pick()
		if err != nil {
			logger.With(zap.Error(err)).Info("get wallpaper failed")
			time.Sleep(errorWait)
			errors++
			continue
		}

		img, err := utils.GetThumbImage(wp, api.ThumbOriginal)
		if err != nil {
			logger.With(zap.Error(err)).Info("get thumb image failed")
			time.Sleep(errorWait)
			errors++
			continue
		}

		img2 := imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)

		if err := dev.DrawBitmap(0, 0, img2); err != nil {
			logger.With(zap.Error(err)).Info("drag bitmap failed")
			time.Sleep(errorWait)
			errors++
			continue
		}

		errors = 0
		time.Sleep(changeWait)
	}
}

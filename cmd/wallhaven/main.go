package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/moolex/wallhaven-go/api"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	"usbscreen/pkg/album"
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
var whRandom = flag.Bool("wh-random", false, "wallhaven random sort")
var whSorting = flag.String("wh-sorting", "", "wallhaven sorting type")
var whToplist = flag.String("wh-toplist", "1M", "wallhaven toplist range")
var whRatio = flag.String("wh-ratio", "", "wallhaven ratio filter")
var tgToken = flag.String("tg-token", "", "telegram bot token")

func main() {
	flag.Parse()

	p := album.NewAlbum(320, 480)

	if d, err := time.ParseDuration(*interval); err != nil {
		log.Fatal(err)
	} else {
		p.ChangeWait = d
	}

	logger, _ := zap.NewDevelopment()

	var dev proto.Control
	var devErr error

	if strings.Contains(*serial, ":") {
		dev, devErr = remote.New(*serial)
	} else {
		dev, devErr = inch35.New(proto.NewSerial(*serial), logger)
	}

	if devErr != nil {
		log.Fatal(devErr)
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

	if *landscape {
		p.SwapRatio()
	}

	wh := api.New(*whKey)
	wh.SetLogger(logger)
	if *debug {
		wh.SetDebug()
	}
	p.SetAPI(wh)

	{
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
		if *whRandom {
			q.Random()
		} else if whSorting != nil && *whSorting != "" {
			q.SortBy(*whSorting)
		} else if whToplist != nil {
			q.SortBy(api.SortTopList)
			q.TopRange = *whToplist
		}
		p.SetQuery(q)
	}

	var bot *album.Bot
	if *tgToken != "" {
		var botErr error
		bot, botErr = album.NewBot(*tgToken, dev, p)
		if botErr != nil {
			log.Fatal(botErr)
		}
		bot.Start()
	}

	if ret, err := wh.Query(p.GetQuery()); err != nil {
		log.Fatal(err)
	} else {
		p.SetResult(ret)
	}

	shutdown := make(chan struct{})
	exited := make(chan struct{})

	go func() {
		timer := time.NewTimer(time.Nanosecond)

		defer func() {
			timer.Stop()
			if bot != nil {
				bot.Stop()
			}
			if err := dev.Shutdown(); err != nil {
				logger.With(zap.Error(err)).Info("shutdown failed")
			}
			exited <- struct{}{}
		}()

		drawer := album.NewDrawer(dev, p)
		wakeupChan := p.WakeupChan()

		for {
			select {
			case <-shutdown:
				return
			case <-wakeupChan:
				timer.Reset(time.Millisecond)
				continue
			case <-timer.C:
				if p.Paused() {
					logger.Info("switch paused, skip...")
					continue
				}
				if err := drawer.Drawing(); err != nil {
					logger.With(zap.Error(err)).Info("drawing failed")
					timer.Reset(p.ErrorWait)
				} else {
					timer.Reset(p.ChangeWait)
				}
			}
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	<-signals
	logger.Info("shutting down")
	shutdown <- struct{}{}
	<-exited
	logger.Info("exited")
}

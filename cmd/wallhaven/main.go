package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/inhies/go-bytesize"
	"github.com/moolex/wallhaven-go/api"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	"usbscreen/pkg/album"
	"usbscreen/pkg/device/inch35"
	"usbscreen/pkg/device/remote"
	"usbscreen/pkg/proto"
)

var serial = flag.String("serial", "ttyACM0", "serial name or remote addr")
var light = flag.Uint8("light", 50, "set light")
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
var cacheDir = flag.String("cache-dir", "", "caching thumb files")
var saveDir = flag.String("save-dir", "", "wallpaper save dir")
var maxSize = flag.String("max-size", "2MB", "max size to fetch origin")
var maxPage = flag.Int("max-page", -1, "max page to fetch")
var autoSaveViews = flag.Int("auto-save-views", -1, "auto save if views than")
var autoSaveFavorites = flag.Int("auto-save-favorites", -1, "auto save if favorites than")

func main() {
	flag.Parse()

	p := album.NewParams(320, 480)
	p.ScreenLight = *light

	if d, err := time.ParseDuration(*interval); err != nil {
		log.Fatal(err)
	} else {
		p.ChangeWait = d
	}

	bSize, bErr := bytesize.Parse(*maxSize)
	if bErr != nil {
		log.Fatal(bErr)
	}

	logger, _ := zap.NewDevelopment()

	cache, cErr := album.NewCache(*cacheDir)
	if cErr != nil {
		log.Fatal(cErr)
	}

	downloader, dErr := album.NewDownloader(*saveDir, logger)
	if dErr != nil {
		log.Fatal(dErr)
	}

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

	if err := dev.SetLight(p.GetLight()); err != nil {
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

	history := album.NewHistory()
	drawer := album.NewDrawer(dev, p, cache, history)

	var bot *album.Bot
	if *tgToken != "" {
		var botErr error
		bot, botErr = album.NewBot(*tgToken, dev, p, downloader, drawer, history)
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
		wakeupChan := p.WakeupChan()
		resetChan := p.ResetChan()

		ab := album.New(p, downloader, drawer,
			album.WithMaxPage(*maxPage),
			album.WithMaxSize(int(bSize)),
			album.WithAutoSave(*autoSaveViews, *autoSaveFavorites),
		)

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

		for {
			select {
			case <-shutdown:
				return
			case <-wakeupChan:
				timer.Reset(time.Millisecond)
				continue
			case dur := <-resetChan:
				timer.Reset(dur)
				continue
			case <-timer.C:
				if p.Paused() {
					logger.Info("switch paused, skip...")
					continue
				}
				if err := ab.Drawing(); err != nil {
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

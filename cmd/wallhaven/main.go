package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/disintegration/imaging"
	"github.com/moolex/wallhaven-go/api"
	"github.com/moolex/wallhaven-go/utils"
	"github.com/samber/lo"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"

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

	p := &params{
		wakeup:    make(chan struct{}),
		errorWait: 3 * time.Second,
		width:     320,
		height:    480,
	}

	if d, err := time.ParseDuration(*interval); err != nil {
		log.Fatal(err)
	} else {
		p.changeWait = d
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
		p.SwapWH()
	}

	p.api = api.New(*whKey)
	p.api.SetLogger(logger)
	if *debug {
		p.api.SetDebug()
	}

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

	var stopBot func()
	if *tgToken != "" {
		var botErr error
		stopBot, botErr = startBot(*tgToken, dev, p, logger)
		if botErr != nil {
			log.Fatal(botErr)
		}
	}

	{
		r, err := p.api.Query(p.GetQuery())
		if err != nil {
			log.Fatal(err)
		}
		p.SetResult(r)
	}

	shutdown := make(chan struct{})
	exited := make(chan struct{})

	go func() {
		timer := time.NewTimer(time.Nanosecond)

		defer func() {
			timer.Stop()
			if stopBot != nil {
				stopBot()
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
			case <-p.wakeup:
				timer.Reset(time.Millisecond)
				continue
			case <-timer.C:
				if p.paused {
					logger.Info("switch paused, skip...")
					continue
				}
				if wp, err := drawing(dev, p, logger); err != nil {
					timer.Reset(p.errorWait)
				} else {
					p.SetWP(wp)
					timer.Reset(p.changeWait)
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

type params struct {
	sync.RWMutex
	wakeup     chan struct{}
	errorWait  time.Duration
	changeWait time.Duration
	width      int
	height     int
	paused     bool
	api        *api.API
	wp         *api.Wallpaper
	q          *api.QueryCond
	r          *api.QueryResult
}

func (p *params) Pause() {
	p.paused = true
}

func (p *params) Wakeup() {
	p.paused = false
	p.wakeup <- struct{}{}
}

func (p *params) SwapWH() {
	p.width, p.height = p.height, p.width
}

func (p *params) SetWP(wp *api.Wallpaper) {
	p.Lock()
	defer p.Unlock()
	p.wp = wp
}

func (p *params) GetWP() *api.Wallpaper {
	p.RLock()
	defer p.RUnlock()
	return p.wp
}

func (p *params) GetQuery() *api.QueryCond {
	p.RLock()
	defer p.RUnlock()
	return p.q
}

func (p *params) GetResult() *api.QueryResult {
	p.RLock()
	defer p.RUnlock()
	return p.r
}

func (p *params) SetQuery(q *api.QueryCond) {
	p.Lock()
	defer p.Unlock()
	p.q = q
}

func (p *params) SetResult(r *api.QueryResult) {
	p.Lock()
	defer p.Unlock()
	p.r = r
}

func (p *params) UpdateQuery(fn func(q *api.QueryCond)) {
	p.Lock()
	defer p.Unlock()
	fn(p.q)
}

func (p *params) UpdateResult(fn func(r *api.QueryResult)) {
	p.Lock()
	defer p.Unlock()
	fn(p.r)
}

func drawing(dev proto.Control, p *params, logger *zap.Logger) (*api.Wallpaper, error) {
	wp, err := p.GetResult().Pick(api.PickLoop, api.PickRand)
	if err != nil {
		if errors.Is(err, api.ErrNoMoreItems) {
			p.UpdateQuery(func(q *api.QueryCond) { q.Page = 1 })
		}
		logger.With(zap.Error(err)).Info("get wallpaper failed")
		return nil, err
	}

	img, err := utils.GetThumbImage(wp, api.ThumbOriginal)
	if err != nil {
		logger.With(zap.Error(err)).Info("get thumb image failed")
		return wp, err
	}

	img2 := imaging.Fill(img, p.width, p.height, imaging.Center, imaging.Lanczos)

	if err := dev.DrawBitmap(0, 0, img2); err != nil {
		logger.With(zap.Error(err)).Info("drag bitmap failed")
		return wp, err
	}

	return wp, nil
}

func reQuery(p *params) error {
	result, err := p.api.Query(p.GetQuery())
	if err != nil {
		return err
	}
	p.SetResult(result)
	p.Wakeup()
	return nil
}

func startBot(token string, dev proto.Control, p *params, logger *zap.Logger) (func(), error) {
	pref := tele.Settings{
		Token: token,
		Poller: &tele.LongPoller{
			Timeout: 10 * time.Second,
		},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	b.Handle("/open", func(context tele.Context) error {
		err := dev.Startup()
		if err != nil {
			logger.With(zap.Error(err)).Info("open failed")
		}
		p.Wakeup()
		return err
	})

	b.Handle("/close", func(context tele.Context) error {
		err := dev.Shutdown()
		if err != nil {
			logger.With(zap.Error(err)).Info("close failed")
		}
		p.Pause()
		return err
	})

	b.Handle("/pause", func(context tele.Context) error {
		logger.With(zap.String("bot-cmd", "pause")).Info("invoking")
		p.Pause()
		return nil
	})

	b.Handle("/resume", func(context tele.Context) error {
		logger.With(zap.String("bot-cmd", "resume")).Info("invoking")
		p.Wakeup()
		return nil
	})

	b.Handle("/interval", func(context tele.Context) error {
		if d, err := time.ParseDuration(context.Message().Payload); err == nil {
			p.changeWait = d
			return nil
		} else {
			return err
		}
	})

	b.Handle("/info", func(context tele.Context) error {
		wp := p.GetWP()
		if wp != nil {
			return context.Reply(wp.Url)
		} else {
			return context.Reply("Current no wallpaper")
		}
	})

	b.Handle("/query", func(context tele.Context) error {
		p.UpdateQuery(func(q *api.QueryCond) {
			q.Query = context.Message().Payload
			q.SortBy(api.SortViews)
		})
		return reQuery(p)
	})

	b.Handle("/toplist", func(context tele.Context) error {
		p.UpdateQuery(func(q *api.QueryCond) {
			q.SortBy(api.SortTopList)
			q.TopRange = context.Message().Payload
		})
		return reQuery(p)
	})

	b.Handle("/sorting", func(context tele.Context) error {
		p.UpdateQuery(func(q *api.QueryCond) {
			q.SortBy(context.Message().Payload)
		})
		return reQuery(p)
	})

	b.Handle("/category", func(context tele.Context) error {
		p.UpdateQuery(func(q *api.QueryCond) {
			q.SetCategory(strings.Split(context.Message().Payload, ",")...)
		})
		return reQuery(p)
	})

	b.Handle("/purity", func(context tele.Context) error {
		p.UpdateQuery(func(q *api.QueryCond) {
			q.SetPurity(strings.Split(context.Message().Payload, ",")...)
		})
		return reQuery(p)
	})

	b.Handle("/page", func(context tele.Context) error {
		in := context.Message().Payload
		if in != "" {
			p.UpdateQuery(func(q *api.QueryCond) {
				p, e := strconv.Atoi(in)
				q.Page = lo.Ternary(e == nil, p, 1)
			})
			return reQuery(p)
		} else {
			r := p.GetResult()
			return context.Reply(fmt.Sprintf(
				"Total items: %d, pagination: %d/%d",
				r.Meta.Total,
				r.Meta.CurrentPage,
				r.Meta.LastPage,
			))
		}
	})

	b.Handle("/params", func(context tele.Context) error {
		query, err := p.GetQuery().ToMap()
		if err != nil {
			return context.Reply(fmt.Sprintf("query error: %s", err))
		}

		var qs []string
		for k, v := range query {
			qs = append(qs, fmt.Sprintf("%s=%s", k, v))
		}

		return context.Reply(strings.Join(qs, "\n"))
	})

	go b.Start()
	return func() { b.Stop() }, nil
}

package album

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/moolex/wallhaven-go/api"
	"github.com/samber/lo"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"

	"usbscreen/pkg/proto"
)

func NewBot(token string, dev proto.Control, album *Album, logger *zap.Logger) (*Bot, error) {
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

	return &Bot{
		b:      b,
		dev:    dev,
		album:  album,
		logger: logger,
	}, nil
}

type Bot struct {
	b      *tele.Bot
	dev    proto.Control
	album  *Album
	logger *zap.Logger
}

func (b *Bot) registerHandles() {
	b.b.Handle("/open", func(context tele.Context) error {
		err := b.dev.Startup()
		if err != nil {
			b.logger.With(zap.Error(err)).Info("open failed")
		}
		b.album.Wakeup()
		return err
	})

	b.b.Handle("/close", func(context tele.Context) error {
		err := b.dev.Shutdown()
		if err != nil {
			b.logger.With(zap.Error(err)).Info("close failed")
		}
		b.album.Pause()
		return err
	})

	b.b.Handle("/pause", func(context tele.Context) error {
		b.logger.With(zap.String("bot-cmd", "pause")).Info("invoking")
		b.album.Pause()
		return nil
	})

	b.b.Handle("/resume", func(context tele.Context) error {
		b.logger.With(zap.String("bot-cmd", "resume")).Info("invoking")
		b.album.Wakeup()
		return nil
	})

	b.b.Handle("/interval", func(context tele.Context) error {
		if d, err := time.ParseDuration(context.Message().Payload); err == nil {
			b.album.ChangeWait = d
			return nil
		} else {
			return err
		}
	})

	b.b.Handle("/info", func(context tele.Context) error {
		wp := b.album.GetWallpaper()
		if wp != nil {
			return context.Reply(wp.Url)
		} else {
			return context.Reply("Current no wallpaper")
		}
	})

	b.b.Handle("/query", func(context tele.Context) error {
		b.album.UpdateQuery(func(q *api.QueryCond) {
			q.Query = context.Message().Payload
			q.SortBy(api.SortViews)
		})
		return b.album.Querying()
	})

	b.b.Handle("/toplist", func(context tele.Context) error {
		b.album.UpdateQuery(func(q *api.QueryCond) {
			q.SortBy(api.SortTopList)
			q.TopRange = context.Message().Payload
		})
		return b.album.Querying()
	})

	b.b.Handle("/sorting", func(context tele.Context) error {
		b.album.UpdateQuery(func(q *api.QueryCond) {
			q.SortBy(context.Message().Payload)
		})
		return b.album.Querying()
	})

	b.b.Handle("/category", func(context tele.Context) error {
		b.album.UpdateQuery(func(q *api.QueryCond) {
			q.SetCategory(strings.Split(context.Message().Payload, ",")...)
		})
		return b.album.Querying()
	})

	b.b.Handle("/purity", func(context tele.Context) error {
		b.album.UpdateQuery(func(q *api.QueryCond) {
			q.SetPurity(strings.Split(context.Message().Payload, ",")...)
		})
		return b.album.Querying()
	})

	b.b.Handle("/page", func(context tele.Context) error {
		in := context.Message().Payload
		if in != "" {
			b.album.UpdateQuery(func(q *api.QueryCond) {
				p, e := strconv.Atoi(in)
				q.Page = lo.Ternary(e == nil, p, 1)
			})
			return b.album.Querying()
		} else {
			r := b.album.GetResult()
			return context.Reply(fmt.Sprintf(
				"Total items: %d, pagination: %d/%d",
				r.Meta.Total,
				r.Meta.CurrentPage,
				r.Meta.LastPage,
			))
		}
	})

	b.b.Handle("/params", func(context tele.Context) error {
		query, err := b.album.GetQuery().ToMap()
		if err != nil {
			return context.Reply(fmt.Sprintf("query error: %s", err))
		}

		var qs []string
		for k, v := range query {
			qs = append(qs, fmt.Sprintf("%s=%s", k, v))
		}

		return context.Reply(strings.Join(qs, "\n"))
	})
}

func (b *Bot) Start() {
	b.registerHandles()
	b.b.Start()
}

func (b *Bot) Stop() {
	b.b.Stop()
}

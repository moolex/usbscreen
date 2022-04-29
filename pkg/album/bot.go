package album

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/moolex/wallhaven-go/api"
	"github.com/samber/lo"
	tele "gopkg.in/telebot.v3"

	"usbscreen/pkg/proto"
)

func NewBot(token string, dev proto.Control, album *Album) (*Bot, error) {
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

	return &Bot{b: b, dev: dev, album: album}, nil
}

type Bot struct {
	b     *tele.Bot
	dev   proto.Control
	album *Album
}

func (b *Bot) handleBase() {
	b.b.Handle("/open", func(context tele.Context) error {
		if err := b.dev.Startup(); err != nil {
			return context.Reply(fmt.Sprintf("open failed: %s", err))
		}

		b.album.Wakeup()
		return context.Reply("OK")
	})

	b.b.Handle("/close", func(context tele.Context) error {
		if err := b.dev.Shutdown(); err != nil {
			return context.Reply(fmt.Sprintf("close failed: %s", err))
		}

		b.album.Pause()
		return context.Reply("OK")
	})

	b.b.Handle("/pause", func(context tele.Context) error {
		b.album.Pause()
		return context.Reply("OK")
	})

	b.b.Handle("/resume", func(context tele.Context) error {
		b.album.Wakeup()
		return context.Reply("OK")
	})
}

func (b *Bot) handleConfig() {
	b.b.Handle("/interval", func(context tele.Context) error {
		in := context.Message().Payload
		if in == "" {
			return context.Reply(b.album.ChangeWait.String())
		}

		duration, err := time.ParseDuration(in)
		if err != nil {
			return context.Reply(fmt.Sprintf("change failed: %s", err))
		}

		b.album.ChangeWait = duration
		b.album.Wakeup()
		return context.Reply("OK")
	})
}

func (b *Bot) handleQuery() {
	b.b.Handle("/info", func(context tele.Context) error {
		wp := b.album.GetWallpaper()
		if wp != nil {
			return context.Reply(wp.Url)
		} else {
			return context.Reply("Current no wallpaper")
		}
	})

	getPageInfo := func() string {
		r := b.album.GetResult()
		return fmt.Sprintf("items: %d, page: %d/%d", r.Meta.Total, r.Meta.CurrentPage, r.Meta.LastPage)
	}

	updateQuery := func(up func(q *api.QueryCond), ctx tele.Context) error {
		b.album.UpdateQuery(func(q *api.QueryCond) {
			q.Page = 1
			up(q)
		})

		if err := b.album.Querying(); err != nil {
			return ctx.Reply(fmt.Sprintf("update failed: %s", err))
		}

		return ctx.Reply(fmt.Sprintf("Updated, %s", getPageInfo()))
	}

	b.b.Handle("/query", func(context tele.Context) error {
		return updateQuery(func(q *api.QueryCond) {
			q.Query = context.Message().Payload
			q.SortBy(api.SortViews)
		}, context)
	})

	b.b.Handle("/toplist", func(context tele.Context) error {
		return updateQuery(func(q *api.QueryCond) {
			q.SortBy(api.SortTopList)
			q.TopRange = context.Message().Payload
		}, context)
	})

	b.b.Handle("/sorting", func(context tele.Context) error {
		return updateQuery(func(q *api.QueryCond) {
			q.SortBy(context.Message().Payload)
		}, context)
	})

	b.b.Handle("/category", func(context tele.Context) error {
		return updateQuery(func(q *api.QueryCond) {
			q.SetCategory(strings.Split(context.Message().Payload, ",")...)
		}, context)
	})

	b.b.Handle("/purity", func(context tele.Context) error {
		return updateQuery(func(q *api.QueryCond) {
			q.SetPurity(strings.Split(context.Message().Payload, ",")...)
		}, context)
	})

	b.b.Handle("/page", func(context tele.Context) error {
		in := context.Message().Payload
		if in == "" {
			return context.Reply(fmt.Sprintf("Total %s", getPageInfo()))
		}

		return updateQuery(func(q *api.QueryCond) {
			p, e := strconv.Atoi(in)
			q.Page = lo.Ternary(e == nil, p, 1)
		}, context)
	})

	b.b.Handle("/preview", func(context tele.Context) error {
		query, err := b.album.GetQuery().ToMap()
		if err != nil {
			return context.Reply(fmt.Sprintf("preview error: %s", err))
		}

		values := make(url.Values)
		for k, v := range query {
			values.Set(k, v)
		}

		return context.Send(fmt.Sprintf("https://wallhaven.cc/search?%s", values.Encode()))
	})
}

func (b *Bot) Start() {
	b.handleBase()
	b.handleConfig()
	b.handleQuery()
	go b.b.Start()
}

func (b *Bot) Stop() {
	// TODO telebot stop will freezes for next response
	go b.b.Stop()
}

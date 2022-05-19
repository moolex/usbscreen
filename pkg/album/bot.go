package album

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/inhies/go-bytesize"
	"github.com/moolex/wallhaven-go/api"
	"github.com/samber/lo"
	tele "gopkg.in/telebot.v3"

	"usbscreen/pkg/proto"
)

func NewBot(token string, dev proto.Control, params *Params, dl *Downloader, d *Drawer, h *History) (*Bot, error) {
	pref := tele.Settings{
		Token: token,
		Poller: &tele.LongPoller{
			Timeout: 30 * time.Second,
		},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return &Bot{
		b:      b,
		dev:    dev,
		params: params,
		dl:     dl,
		d:      d,
		h:      h,
	}, nil
}

type Bot struct {
	b      *tele.Bot
	dev    proto.Control
	params *Params
	dl     *Downloader
	d      *Drawer
	h      *History
}

func (b *Bot) handleBase() {
	b.b.Handle("/open", func(context tele.Context) error {
		if err := b.dev.Startup(); err != nil {
			return context.Reply(fmt.Sprintf("open failed: %s", err))
		}

		b.params.Wakeup()
		return context.Reply("OK")
	})

	b.b.Handle("/close", func(context tele.Context) error {
		if err := b.dev.Shutdown(); err != nil {
			return context.Reply(fmt.Sprintf("close failed: %s", err))
		}

		b.params.Pause()
		return context.Reply("OK")
	})

	b.b.Handle("/pause", func(context tele.Context) error {
		b.params.Pause()
		return context.Reply("OK")
	})

	b.b.Handle("/resume", func(context tele.Context) error {
		b.params.Wakeup()
		return context.Reply("OK")
	})
}

func (b *Bot) handleConfig() {
	b.b.Handle("/interval", func(context tele.Context) error {
		in := context.Message().Payload
		if in == "" {
			return context.Reply(b.params.ChangeWait.String())
		}

		duration, err := time.ParseDuration(in)
		if err != nil {
			return context.Reply(fmt.Sprintf("change failed: %s", err))
		}

		b.params.ChangeWait = duration
		b.params.Wakeup()
		return context.Reply("OK")
	})

	b.b.Handle("/light", func(context tele.Context) error {
		in := context.Message().Payload
		if in == "" {
			return context.Reply(strconv.Itoa(int(b.params.ScreenLight)))
		}

		if parsed, err := strconv.ParseUint(in, 10, 8); err == nil {
			b.params.ScreenLight = uint8(parsed)
		}

		if err := b.dev.SetLight(b.params.GetLight()); err != nil {
			return context.Reply(fmt.Sprintf("change failed: %s", err))
		}

		return context.Reply("OK")
	})
}

func (b *Bot) handleQuery() {
	getPageInfo := func() string {
		r := b.params.GetResult()
		return fmt.Sprintf("items: %d, page: %d/%d", r.Meta.Total, r.Meta.CurrentPage, r.Meta.LastPage)
	}

	updateQuery := func(up func(q *api.QueryCond), ctx tele.Context) error {
		b.params.UpdateQuery(func(q *api.QueryCond) {
			q.Page = 1
			up(q)
		})

		if err := b.params.Querying(); err != nil {
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
		query, err := b.params.GetQuery().ToMap()
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

func (b *Bot) handleAction() {
	b.b.Handle("/info", func(context tele.Context) error {
		log := b.h.Curr()
		if log == nil {
			return context.Reply("Current no wallpaper")
		}

		wp := log.wp
		lines := []string{
			fmt.Sprintf("Category: %s", wp.Category),
			fmt.Sprintf("Purity: %s", wp.Purity),
			fmt.Sprintf("Views: %d", wp.Views),
			fmt.Sprintf("Favorites: %d", wp.Favorites),
			fmt.Sprintf("Resolution: %s", wp.Resolution),
			fmt.Sprintf("File size: %s", bytesize.New(float64(wp.FileSize)).String()),
			fmt.Sprintf("Created at: %s", wp.CreatedAt),
			fmt.Sprintf("URL: %s", wp.Url),
		}

		return context.Reply(strings.Join(lines, "\n"))
	})

	b.b.Handle("/logs", func(context tele.Context) error {
		var lines []string
		for _, log := range b.h.Logs() {
			lines = append(lines, log.wp.Url)
		}

		return context.Reply(strings.Join(lines, "\n"))
	})

	b.b.Handle("/prev", func(context tele.Context) error {
		log := b.h.Prev()
		if log == nil {
			return context.Reply("Previous no item")
		}

		if err := b.d.Canvas(log.filled); err != nil {
			return context.Reply(fmt.Sprintf("draw canvas failed: %s", err))
		}

		b.params.Reset(b.params.ChangeWait)
		b.h.Push(log)

		return context.Reply("OK")
	})

	b.b.Handle("/full", func(context tele.Context) error {
		log := b.h.Curr()
		if log == nil {
			return context.Reply("Current no item")
		}
		if !log.thumb {
			return context.Reply("Current is full size")
		}

		b.d.Lock()
		defer b.d.Unlock()

		filled, err := b.d.Filled(
			log.wp,
			func(wp *api.Wallpaper) (*VFile, bool, error) {
				vf, err := b.dl.Get(wp, false)
				return vf, false, err
			},
			nil,
			func(wp *api.Wallpaper, thumb bool, origin *VFile) error {
				return b.dl.Save(wp, origin)
			},
		)
		if err != nil {
			return context.Reply(fmt.Sprintf("get thumb failed: %s", err))
		}

		if err := b.d.Canvas(filled); err != nil {
			return context.Reply(fmt.Sprintf("draw canvas failed: %s", err))
		}

		b.params.Reset(b.params.ChangeWait)

		return context.Reply("OK")
	})

	b.b.Handle("/save", func(context tele.Context) error {
		log := b.h.Curr()
		if log == nil {
			return context.Reply("Current no item")
		}

		if err := b.dl.Save(log.wp, lo.Ternary(log.thumb, nil, log.origin)); err != nil {
			return context.Reply(fmt.Sprintf("save failed: %s", err))
		}

		return context.Reply(fmt.Sprintf("Saved of %s", log.wp.Url))
	})
}

func (b *Bot) Start() {
	b.handleBase()
	b.handleConfig()
	b.handleQuery()
	b.handleAction()
	go b.b.Start()
}

func (b *Bot) Stop() {
	// TODO telebot stop will freezes for next response
	go b.b.Stop()
}

package album

import (
	"image"

	"github.com/moolex/wallhaven-go/api"
	"github.com/samber/lo"
)

func NewHistory() *History {
	return &History{max: 3}
}

type History struct {
	max   int
	items []*HistoryLog
}

type HistoryLog struct {
	wp     *api.Wallpaper
	filled image.Image
	thumb  bool
	origin []byte
}

func (h *History) push(item *HistoryLog) {
	h.items = append(h.items, item)
	if len(h.items) > h.max {
		h.items = h.items[1:]
	}
}

func (h *History) Logs() []*HistoryLog {
	return h.items
}

func (h *History) Add(wp *api.Wallpaper, filled image.Image, thumb bool, origin []byte) {
	h.push(&HistoryLog{wp: wp, filled: filled, thumb: thumb, origin: origin})
}

func (h *History) Push(item *HistoryLog) {
	h.push(item)
}

func (h *History) Curr() *HistoryLog {
	log, _ := lo.Last(h.items)
	return log
}

func (h *History) Prev() *HistoryLog {
	log, _ := lo.Nth(h.items, -2)
	return log
}

func (h *History) Preload() error {
	return nil
}

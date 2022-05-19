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
	origin *VFile
}

func (h *History) push(item *HistoryLog) {
	h.items = append(h.items, item)
	if len(h.items) > h.max {
		item := h.items[0]
		if item.origin != nil {
			_ = item.origin.Free()
		}
		h.items = h.items[1:]
	}
}

func (h *History) Clean() {
	for _, i := range h.items {
		if i.origin != nil {
			_ = i.origin.Free()
		}
	}
}

func (h *History) Logs() []*HistoryLog {
	return h.items
}

func (h *History) Add(wp *api.Wallpaper, filled image.Image, thumb bool, origin *VFile) {
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

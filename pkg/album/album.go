package album

import (
	"sync"
	"time"

	"github.com/moolex/wallhaven-go/api"
)

func NewAlbum(width, height int) *Album {
	p := &Album{
		ErrorWait:  3 * time.Second,
		ChangeWait: 30 * time.Second,
		wakeup:     make(chan struct{}, 1),
		width:      width,
		height:     height,
	}
	return p
}

type Album struct {
	l sync.RWMutex

	ErrorWait  time.Duration
	ChangeWait time.Duration

	wakeup chan struct{}
	paused bool
	width  int
	height int
	api    *api.API
	wp     *api.Wallpaper
	q      *api.QueryCond
	r      *api.QueryResult
}

func (p *Album) Paused() bool {
	return p.paused
}

func (p *Album) WakeupChan() <-chan struct{} {
	return p.wakeup
}

func (p *Album) Pause() {
	p.paused = true
}

func (p *Album) Wakeup() {
	p.paused = false
	p.wakeup <- struct{}{}
}

func (p *Album) SetAPI(api *api.API) {
	p.api = api
}

func (p *Album) SwapRatio() {
	p.width, p.height = p.height, p.width
}

func (p *Album) SetWallpaper(wp *api.Wallpaper) {
	p.l.Lock()
	defer p.l.Unlock()
	p.wp = wp
}

func (p *Album) GetWallpaper() *api.Wallpaper {
	p.l.RLock()
	defer p.l.RUnlock()
	return p.wp
}

func (p *Album) GetQuery() *api.QueryCond {
	p.l.RLock()
	defer p.l.RUnlock()
	return p.q
}

func (p *Album) GetResult() *api.QueryResult {
	p.l.RLock()
	defer p.l.RUnlock()
	return p.r
}

func (p *Album) SetQuery(q *api.QueryCond) {
	p.l.Lock()
	defer p.l.Unlock()
	p.q = q
}

func (p *Album) SetResult(r *api.QueryResult) {
	p.l.Lock()
	defer p.l.Unlock()
	p.r = r
}

func (p *Album) UpdateQuery(fn func(q *api.QueryCond)) {
	p.l.Lock()
	defer p.l.Unlock()
	fn(p.q)
}

func (p *Album) UpdateResult(fn func(r *api.QueryResult)) {
	p.l.Lock()
	defer p.l.Unlock()
	fn(p.r)
}

func (p *Album) Querying() error {
	p.l.Lock()
	defer p.l.Unlock()

	ret, err := p.api.Query(p.q)
	if err != nil {
		return err
	}

	p.r = ret
	p.Wakeup()
	return nil
}

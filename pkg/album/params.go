package album

import (
	"sync"
	"time"

	"github.com/moolex/wallhaven-go/api"
)

func NewParams(width, height int) *Params {
	p := &Params{
		ErrorWait:   3 * time.Second,
		ChangeWait:  30 * time.Second,
		ScreenLight: 50,
		wakeup:      make(chan struct{}, 1),
		reset:       make(chan time.Duration, 1),
		width:       width,
		height:      height,
	}
	return p
}

type Params struct {
	l sync.RWMutex

	ErrorWait   time.Duration
	ChangeWait  time.Duration
	ScreenLight uint8

	wakeup chan struct{}
	reset  chan time.Duration
	paused bool
	width  int
	height int
	api    *api.API
	q      *api.QueryCond
	r      *api.QueryResult
}

func (p *Params) Paused() bool {
	return p.paused
}

func (p *Params) WakeupChan() <-chan struct{} {
	return p.wakeup
}

func (p *Params) ResetChan() <-chan time.Duration {
	return p.reset
}

func (p *Params) Pause() {
	p.paused = true
}

func (p *Params) Wakeup() {
	p.paused = false
	p.wakeup <- struct{}{}
}

func (p *Params) Reset(dur time.Duration) {
	p.reset <- dur
}

func (p *Params) SetAPI(api *api.API) {
	p.api = api
}

func (p *Params) SwapRatio() {
	p.width, p.height = p.height, p.width
}

func (p *Params) GetLight() uint8 {
	return uint8((1 - float64(p.ScreenLight)/100) * 255)
}

func (p *Params) GetQuery() *api.QueryCond {
	p.l.RLock()
	defer p.l.RUnlock()
	return p.q
}

func (p *Params) GetResult() *api.QueryResult {
	p.l.RLock()
	defer p.l.RUnlock()
	return p.r
}

func (p *Params) SetQuery(q *api.QueryCond) {
	p.l.Lock()
	defer p.l.Unlock()
	p.q = q
}

func (p *Params) SetResult(r *api.QueryResult) {
	p.l.Lock()
	defer p.l.Unlock()
	p.r = r
}

func (p *Params) UpdateQuery(fn func(q *api.QueryCond)) {
	p.l.Lock()
	defer p.l.Unlock()
	fn(p.q)
}

func (p *Params) UpdateResult(fn func(r *api.QueryResult)) {
	p.l.Lock()
	defer p.l.Unlock()
	fn(p.r)
}

func (p *Params) Querying() error {
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

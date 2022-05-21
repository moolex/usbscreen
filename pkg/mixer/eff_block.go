package mixer

import (
	"image"
	"math/rand"
	"time"

	"github.com/samber/lo"
)

func EffectBlock() Effect {
	return &block{
		size: 32,
		rand: true,
	}
}

type block struct {
	size int
	rand bool
}

func (e *block) Name() string {
	return "block"
}

func (e *block) Process(img Image) (<-chan Write, error) {
	wc := make(chan Write)

	go func() {
		r := img.Bounds()
		w, h := r.Dx(), r.Dy()

		size := e.size
		if e.rand {
			rand.Seed(time.Now().UnixNano())
			size = rand.Intn(32) + 8
		}

		var ws []Write
		for x := 0; x < w; x += size {
			for y := 0; y < h; y += size {
				ws = append(ws, Write{
					At:  image.Pt(x, y),
					Img: img.SubImage(image.Rect(x, y, x+size, y+size)),
				})
			}
		}

		if e.rand {
			lo.Shuffle(ws)
		}

		for _, w2 := range ws {
			wc <- w2
		}

		close(wc)
	}()

	return wc, nil
}

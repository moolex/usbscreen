package album

import (
	"time"

	"github.com/moolex/wallhaven-go/api"
	"go.uber.org/zap"
)

type Option func(a *Album)

func WithMaxPage(max int) Option {
	return func(a *Album) {
		a.maxPage = max
	}
}

func WithMaxSize(max int) Option {
	return func(a *Album) {
		a.maxSize = max
	}
}

func WithAutoSave(log *zap.Logger, views, favorites, fIncDaily int) Option {
	return func(a *Album) {
		a.autoSave = &autoSave{
			log:       log.With(zap.String("via", "autoSave-checker")),
			views:     views,
			favorites: favorites,
			fIncDaily: fIncDaily,
		}
	}
}

type autoSave struct {
	log       *zap.Logger
	views     int
	favorites int
	fIncDaily int // favorites increments daily
}

func (s *autoSave) Check(wp *api.Wallpaper) bool {
	log := s.log.With(zap.String("id", wp.Id))

	if s.views > 0 && wp.Views >= s.views {
		log.With(zap.String("by", "views"), zap.Int("views", wp.Views)).Debug("pass")
		return true
	}

	if s.favorites > 0 && wp.Favorites >= s.favorites {
		log.With(zap.String("by", "favorites"), zap.Int("favorites", wp.Favorites)).Debug("pass")
		return true
	}

	if s.fIncDaily > 0 {
		t, err := time.Parse("2006-01-02 15:04:05", wp.CreatedAt)
		if err != nil {
			log.With(zap.Error(err)).Info("parse date fail")
			return false
		}
		days := time.Since(t).Hours() / 24
		daily := int(float64(wp.Favorites) / days)
		if daily >= s.fIncDaily {
			log.With(
				zap.String("by", "fIncDaily"),
				zap.Int("daily", daily),
				zap.Int("total", wp.Favorites),
				zap.Int("days", int(days)),
			).Debug("passed")
			return true
		}
	}

	return false
}

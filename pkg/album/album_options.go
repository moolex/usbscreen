package album

import "github.com/moolex/wallhaven-go/api"

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

func WithAutoSave(views, favorites int) Option {
	return func(a *Album) {
		a.autoSave = &autoSave{
			views:     views,
			favorites: favorites,
		}
	}
}

type autoSave struct {
	views     int
	favorites int
}

func (s *autoSave) Check(wp *api.Wallpaper) bool {
	return (s.views > 0 && wp.Views >= s.views) ||
		(s.favorites > 0 && wp.Favorites >= s.favorites)
}

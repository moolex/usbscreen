package mixer

type Option func(d *Drawer)

func WithEffect(e ...Effect) Option {
	return func(d *Drawer) {
		d.effs = e
	}
}

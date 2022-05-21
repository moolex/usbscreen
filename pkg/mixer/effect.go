package mixer

import "image"

type Write struct {
	At  image.Point
	Img image.Image
}

type Image interface {
	image.Image
	SubImage(image.Rectangle) image.Image
}

type Effect interface {
	Name() string
	Process(img Image) (<-chan Write, error)
}

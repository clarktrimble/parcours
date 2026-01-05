package board

import "image"

type rectIter struct {
	template   []image.Rectangle
	rectantles []image.Rectangle
	offset     int
}

func newRectIter(layout []image.Rectangle) *rectIter {
	return &rectIter{
		template:   layout,
		rectantles: make([]image.Rectangle, len(layout)),
		offset:     0,
	}
}

func (iter *rectIter) next() []image.Rectangle {

	// render a slice of rectantles using x's from layout and bump y's
	for i, rectangle := range iter.template {
		iter.rectantles[i] = image.Rect(
			rectangle.Min.X,
			iter.offset,
			rectangle.Max.X,
			iter.offset+1,
		)
	}
	iter.offset++

	return iter.rectantles
}

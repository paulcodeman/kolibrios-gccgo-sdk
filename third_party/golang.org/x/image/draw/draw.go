// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package draw provides image composition functions.
//
// See "The Go image/draw package" for an introduction to this package:
// http://golang.org/doc/articles/image_draw.html
//
// This package is a superset of and a drop-in replacement for the image/draw
// package in the standard library.
package draw

// This file just contains the API exported by the image/draw package in the
// standard library. Other files in this package provide additional features.

import (
	"image"
	imagedraw "image/draw"
)

// Draw calls DrawMask with a nil mask.
func Draw(dst Image, r image.Rectangle, src image.Image, sp image.Point, op Op) {
	imagedraw.Draw(dst, r, src, sp, imagedraw.Op(op))
}

// DrawMask aligns r.Min in dst with sp in src and mp in mask and then
// replaces the rectangle r in dst with the result of a Porter-Duff
// composition. A nil mask is treated as opaque.
func DrawMask(dst Image, r image.Rectangle, src image.Image, sp image.Point, mask image.Image, mp image.Point, op Op) {
	imagedraw.DrawMask(dst, r, src, sp, mask, mp, imagedraw.Op(op))
}

// Drawer contains the Draw method.
type Drawer = imagedraw.Drawer

// FloydSteinberg is a Drawer that is the Src Op with Floyd-Steinberg error
// diffusion.
var FloydSteinberg Drawer = floydSteinberg{}

type floydSteinberg struct{}

func (floydSteinberg) Draw(dst Image, r image.Rectangle, src image.Image, sp image.Point) {
	imagedraw.FloydSteinberg.Draw(dst, r, src, sp)
}

// Image is an image.Image with a Set method to change a single pixel.
type Image = imagedraw.Image

// RGBA64Image extends both the Image and image.RGBA64Image interfaces with a
// SetRGBA64 method to change a single pixel. SetRGBA64 is equivalent to
// calling Set, but it can avoid allocations from converting concrete color
// types to the color.Color interface type.
type RGBA64Image = imagedraw.RGBA64Image

// Op is a Porter-Duff compositing operator.
type Op = imagedraw.Op

const (
	// Over specifies ``(src in mask) over dst''.
	Over Op = imagedraw.Over
	// Src specifies ``src in mask''.
	Src Op = imagedraw.Src
)

// Quantizer produces a palette for an image.
type Quantizer = imagedraw.Quantizer

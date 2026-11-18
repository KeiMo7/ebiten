// Copyright 2026 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bitdepth

import (
	"fmt"
)

// Image is an RGBA image whose samples are stored as uint16 values in
// [0, Depth().Max()]. Storing every depth in the same container type
// guarantees that no implicit truncation to 8 bits can happen inside
// the pipeline.
type Image struct {
	width  int
	height int
	format Format
	pix    []uint16
}

// NewImage returns a new RGBA image of the given format. The format
// must be FormatRGBA8 or FormatRGBA10; planar formats such as
// FormatYUV420P10LE are converted explicitly with
// YUV420P10LEToRGBA10 instead.
func NewImage(width, height int, format Format) *Image {
	return &Image{
		width:  width,
		height: height,
		format: format,
		pix:    make([]uint16, 4*width*height),
	}
}

// FromRGBA8 converts raw RGBA8 pixels, 4 bytes per pixel, into an
// Image. The sample values are preserved as-is in [0, 255]; no
// upconversion is applied.
func FromRGBA8(pix []byte, width, height int) (*Image, error) {
	if got, want := len(pix), 4*width*height; got != want {
		return nil, fmt.Errorf("bitdepth: invalid RGBA8 pixel length %d, want %d", got, want)
	}
	img := NewImage(width, height, FormatRGBA8)
	for i, b := range pix {
		img.pix[i] = uint16(b)
	}
	return img, nil
}

// Width returns the width of the image in pixels.
func (i *Image) Width() int {
	return i.width
}

// Height returns the height of the image in pixels.
func (i *Image) Height() int {
	return i.height
}

// Format returns the pixel format of the image.
func (i *Image) Format() Format {
	return i.format
}

// Depth returns the bit depth per channel of the image.
func (i *Image) Depth() Depth {
	return i.format.Depth()
}

// Pix returns the RGBA samples of the image, 4 values per pixel in
// [0, Depth().Max()].
func (i *Image) Pix() []uint16 {
	return i.pix
}

// Clone returns a deep copy of the image.
func (i *Image) Clone() *Image {
	pix := make([]uint16, len(i.pix))
	copy(pix, i.pix)
	return &Image{
		width:  i.width,
		height: i.height,
		format: i.format,
		pix:    pix,
	}
}

// Validate checks that every sample fits the declared bit depth of the
// image. It detects inputs whose declared depth does not match the
// actual content, e.g. samples beyond the 8-bit range in an image
// declared as 8-bit.
func Validate(i *Image) error {
	max := i.Depth().Max()
	for idx, v := range i.pix {
		if v > max {
			return fmt.Errorf("bitdepth: sample value %d at index %d exceeds the %s range [0, %d]", v, idx, i.Depth(), max)
		}
	}
	return nil
}

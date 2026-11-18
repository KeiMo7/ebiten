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

// Package bitdepth provides a pixel pipeline that preserves bit depths
// beyond 8 bits per channel.
//
// The rest of Ebitengine's rendering pipeline is fixed to 8 bits per
// channel. This package offers an explicit opt-in path for 10-bit
// content: 10-bit inputs pass through without an 8-bit intermediate
// format, 8-bit inputs can be upconverted with a configurable dither
// strategy, and mixed-depth batches are converted item by item instead
// of falling back to the lowest bit depth of the batch.
//
// Nothing in this package is enabled implicitly. Existing callers that
// do not reference this package observe no change in return values,
// error types, or log output.
//
// Migration example: an existing 8-bit-only flow
//
//	pix := make([]byte, 4*w*h) // RGBA8
//
// keeps working untouched. To route 10-bit content through the new
// path, build a bitdepth.Image and convert explicitly:
//
//	img := bitdepth.NewImage(w, h, bitdepth.FormatRGBA10)
//	// ... fill img.Pix with 10-bit samples in [0, 1023] ...
//	if err := bitdepth.CheckTarget(target, img.Format()); err != nil {
//		// The target cannot represent 10 bits. Handle the error
//		// explicitly; no silent fallback to 8 bits happens here.
//		return err
//	}
//	out, err := bitdepth.Convert(img, bitdepth.Depth10, bitdepth.DitherNone)
//
// All functions in this package are stateless and safe for concurrent
// use. DrawOver mutates its destination image; callers must synchronize
// concurrent access to the same destination, as with any image type.
package bitdepth

import (
	"errors"
	"fmt"
)

// ErrUnsupportedFormat is returned when a target cannot represent a pixel format.
var ErrUnsupportedFormat = errors.New("bitdepth: unsupported format")

// ErrDepthMismatch is returned when an operation requires matching bit depths.
var ErrDepthMismatch = errors.New("bitdepth: depth mismatch")

// Depth is a bit depth per channel.
type Depth int

const (
	// Depth8 is 8 bits per channel, the depth of the existing pipeline.
	Depth8 Depth = 8
	// Depth10 is 10 bits per channel.
	Depth10 Depth = 10
)

// Valid reports whether d is a supported depth.
func (d Depth) Valid() bool {
	return d == Depth8 || d == Depth10
}

// Max returns the maximum sample value for the depth.
func (d Depth) Max() uint16 {
	return 1<<uint(d) - 1
}

func (d Depth) String() string {
	switch d {
	case Depth8:
		return "8bit"
	case Depth10:
		return "10bit"
	}
	return fmt.Sprintf("Depth(%d)", int(d))
}

// Format is a pixel format.
type Format int

const (
	// FormatRGBA8 is RGBA with 8 bits per channel.
	FormatRGBA8 Format = iota
	// FormatRGBA10 is RGBA with 10 bits per channel.
	FormatRGBA10
	// FormatYUV420P10LE is planar YUV 4:2:0 with 10-bit samples stored
	// in little-endian 16-bit words, as produced by common decoders.
	FormatYUV420P10LE
)

// Depth returns the bit depth per channel of the format.
func (f Format) Depth() Depth {
	switch f {
	case FormatRGBA8:
		return Depth8
	case FormatRGBA10, FormatYUV420P10LE:
		return Depth10
	}
	return 0
}

func (f Format) String() string {
	switch f {
	case FormatRGBA8:
		return "RGBA8"
	case FormatRGBA10:
		return "RGBA10"
	case FormatYUV420P10LE:
		return "YUV420P10LE"
	}
	return fmt.Sprintf("Format(%d)", int(f))
}

// Dither is a dither strategy applied when converting between depths.
type Dither int

const (
	// DitherNone applies no dither. Conversions are exact where the
	// target depth can represent the source values. This is the zero
	// value and the default.
	DitherNone Dither = iota
	// DitherOrdered applies a 4x4 ordered (Bayer) dither. On
	// downconversion this hides quantization banding; on upconversion
	// this spreads source bands over adjacent output codes.
	DitherOrdered
)

func (d Dither) String() string {
	switch d {
	case DitherNone:
		return "none"
	case DitherOrdered:
		return "ordered"
	}
	return fmt.Sprintf("Dither(%d)", int(d))
}

// Target describes the formats an output target, such as an encoder or
// a container, can represent.
type Target struct {
	Formats []Format
}

// Supports reports whether the target can represent the format.
func (t Target) Supports(f Format) bool {
	for _, tf := range t.Formats {
		if tf == f {
			return true
		}
	}
	return false
}

// CheckTarget reports whether the target can represent the format. If
// not, the error lists the supported alternatives and wraps
// ErrUnsupportedFormat. CheckTarget never silently falls back to a
// lower bit depth.
func CheckTarget(t Target, f Format) error {
	logf("check-target: enter: format=%s", f)
	if t.Supports(f) {
		logf("check-target: exit: format=%s supported", f)
		return nil
	}
	err := fmt.Errorf("bitdepth: target does not support %s; supported formats: %v: %w", f, t.Formats, ErrUnsupportedFormat)
	logf("check-target: failure: %v", err)
	return err
}

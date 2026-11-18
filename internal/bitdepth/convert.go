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

// bayer4 is a 4x4 ordered dither matrix with values in [0, 15].
var bayer4 = [16]uint16{
	0, 8, 2, 10,
	12, 4, 14, 6,
	3, 11, 1, 9,
	15, 7, 13, 5,
}

// Convert returns a copy of img at the target depth.
//
// If img already has the target depth, the samples pass through
// unchanged: no 8-bit intermediate format is involved at any point.
// Upconversion from 8 to 10 bits uses bit replication, which is exact.
// Downconversion rounds to nearest, optionally with the given dither.
//
// Convert fails if img does not validate or the target depth is not
// supported. On failure img is left untouched and no partial result is
// returned.
func Convert(img *Image, target Depth, dither Dither) (*Image, error) {
	logf("convert: enter: depth=%s target=%s dither=%s size=%dx%d", img.Depth(), target, dither, img.width, img.height)
	out, err := convert(img, target, dither)
	if err != nil {
		logf("convert: failure: %v", err)
		return nil, err
	}
	logf("convert: exit: depth=%s", out.Depth())
	return out, nil
}

func convert(img *Image, target Depth, dither Dither) (*Image, error) {
	if !target.Valid() {
		return nil, fmt.Errorf("bitdepth: invalid target depth %s", target)
	}
	if err := Validate(img); err != nil {
		return nil, err
	}

	src := img.Depth()
	if src == target {
		return img.Clone(), nil
	}

	format := FormatRGBA8
	if target == Depth10 {
		format = FormatRGBA10
	}
	out := NewImage(img.width, img.height, format)
	for idx, v := range img.pix {
		x := (idx / 4) % img.width
		y := (idx / 4) / img.width
		out.pix[idx] = convertSample(v, src, target, dither, x, y)
	}
	return out, nil
}

func convertSample(v uint16, src, target Depth, dither Dither, x, y int) uint16 {
	switch {
	case src == Depth8 && target == Depth10:
		// Bit replication is the exact inverse of rounding division:
		// v*1023/255 == v<<2 | v>>6 for every v in [0, 255].
		up := v<<2 | v>>6
		if dither == DitherOrdered {
			// Spread 8-bit source bands over adjacent 10-bit codes.
			up = clampSample(int(up)+int(bayer4[y%4*4+x%4]%3)-1, Depth10.Max())
		}
		return up
	case src == Depth10 && target == Depth8:
		// Scale by 32 so that the ordered threshold, a fraction of one
		// output step, can be applied with integer math.
		n := int(v) * int(Depth8.Max()) * 32
		d := int(Depth10.Max()) * 32
		if dither == DitherOrdered {
			// Add an ordered threshold in [-0.5, 0.5) of one output
			// step before quantization.
			threshold := (int(bayer4[y%4*4+x%4])*2 - 15) * int(Depth10.Max())
			return clampSample((n+threshold)/d, Depth8.Max())
		}
		return uint16((n + d/2) / d)
	}
	return v
}

func clampSample(v int, max uint16) uint16 {
	if v < 0 {
		return 0
	}
	if v > int(max) {
		return max
	}
	return uint16(v)
}

// ConvertBatch converts each image to the target depth and returns the
// results in the same order. Every image is considered individually:
// images already at the target depth pass through unchanged, and only
// the others are converted. The batch is never converted as a whole to
// the lowest bit depth present.
//
// The conversion is atomic: if any image fails, ConvertBatch returns
// an error and no partial results, so a failure cannot leave behind a
// set of images with inconsistent depths.
func ConvertBatch(imgs []*Image, target Depth, dither Dither) ([]*Image, error) {
	logf("convert-batch: enter: count=%d target=%s dither=%s", len(imgs), target, dither)
	outs := make([]*Image, len(imgs))
	for idx, img := range imgs {
		out, err := convert(img, target, dither)
		if err != nil {
			err = fmt.Errorf("bitdepth: converting item %d of %d: %w", idx, len(imgs), err)
			logf("convert-batch: failure: %v", err)
			return nil, err
		}
		outs[idx] = out
	}
	logf("convert-batch: exit: count=%d depth=%s", len(outs), target)
	return outs, nil
}

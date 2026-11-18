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

// YUVColorSpace is the color matrix used to convert YUV to RGB. The
// caller must choose one explicitly; this package has no default, so
// conversions never silently depend on an encoder's assumption.
type YUVColorSpace int

const (
	// YUVColorSpaceBT601 is the ITU-R BT.601 matrix.
	YUVColorSpaceBT601 YUVColorSpace = iota
	// YUVColorSpaceBT709 is the ITU-R BT.709 matrix.
	YUVColorSpaceBT709
)

func (c YUVColorSpace) String() string {
	switch c {
	case YUVColorSpaceBT601:
		return "BT.601"
	case YUVColorSpaceBT709:
		return "BT.709"
	}
	return fmt.Sprintf("YUVColorSpace(%d)", int(c))
}

// YUV420P10LEToRGBA10 converts planar YUV 4:2:0 with 10-bit samples,
// stored in little-endian 16-bit words (yuv420p10le), into a 10-bit
// RGBA image. y, u, and v are the three planes; yStride and cStride
// are the strides of the luma and chroma planes in samples. The input
// is assumed to be limited range, as is standard for yuv420p10le.
//
// The conversion is explicit: the color space is a required argument
// and the output depth always matches the 10-bit input. Samples are
// clamped to the 10-bit range instead of being truncated.
func YUV420P10LEToRGBA10(y, u, v []byte, width, height, yStride, cStride int, cs YUVColorSpace) (*Image, error) {
	logf("yuv-to-rgba: enter: format=%s colorspace=%s size=%dx%d", FormatYUV420P10LE, cs, width, height)
	out, err := yuv420P10LEToRGBA10(y, u, v, width, height, yStride, cStride, cs)
	if err != nil {
		logf("yuv-to-rgba: failure: %v", err)
		return nil, err
	}
	logf("yuv-to-rgba: exit: depth=%s", out.Depth())
	return out, nil
}

func yuv420P10LEToRGBA10(y, u, v []byte, width, height, yStride, cStride int, cs YUVColorSpace) (*Image, error) {
	if cs != YUVColorSpaceBT601 && cs != YUVColorSpaceBT709 {
		return nil, fmt.Errorf("bitdepth: invalid YUV color space %s", cs)
	}
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("bitdepth: invalid size %dx%d", width, height)
	}
	if want := 2 * yStride * height; len(y) < want {
		return nil, fmt.Errorf("bitdepth: luma plane too short: %d bytes, want at least %d", len(y), want)
	}
	if want := 2 * cStride * ((height + 1) / 2); len(u) < want || len(v) < want {
		return nil, fmt.Errorf("bitdepth: chroma plane too short: %d and %d bytes, want at least %d each", len(u), len(v), want)
	}

	// Limited-range coefficients scaled by 1<<16.
	var krv, kgu, kgv, kbu int
	switch cs {
	case YUVColorSpaceBT601:
		krv, kgu, kgv, kbu = 104597, 25675, 53264, 132201 // 1.596, 0.392, 0.813, 2.017
	case YUVColorSpaceBT709:
		krv, kgu, kgv, kbu = 117489, 13964, 34943, 138439 // 1.793, 0.213, 0.533, 2.112
	}
	const ky = 76309 // 1.164 * 1<<16

	sample := func(p []byte, idx int) int {
		return int(uint16(p[2*idx]) | uint16(p[2*idx+1])<<8)
	}

	out := NewImage(width, height, FormatRGBA10)
	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			ly := sample(y, py*yStride+px) - 64
			cu := sample(u, (py/2)*cStride+px/2) - 512
			cv := sample(v, (py/2)*cStride+px/2) - 512
			d := 4 * (py*width + px)
			out.pix[d] = clampSample((ky*ly+krv*cv+(1<<15))>>16, Depth10.Max())
			out.pix[d+1] = clampSample((ky*ly-kgu*cu-kgv*cv+(1<<15))>>16, Depth10.Max())
			out.pix[d+2] = clampSample((ky*ly+kbu*cu+(1<<15))>>16, Depth10.Max())
			out.pix[d+3] = Depth10.Max()
		}
	}
	return out, nil
}

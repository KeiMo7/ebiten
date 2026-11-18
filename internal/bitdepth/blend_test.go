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

package bitdepth_test

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/bitdepth"
)

func TestDrawOver10bit(t *testing.T) {
	dst := bitdepth.NewImage(2, 1, bitdepth.FormatRGBA10)
	copy(dst.Pix(), []uint16{1000, 0, 0, 1023, 0, 0, 0, 0})
	src := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA10)
	copy(src.Pix(), []uint16{0, 0, 0, 512})

	if err := bitdepth.DrawOver(dst, src, 0, 0); err != nil {
		t.Fatal(err)
	}
	// out = (0*512 + 1000*511) / 1023, rounded.
	if got, want := dst.Pix()[0], uint16(500); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
	if got := dst.Pix()[3]; got != 1023 {
		t.Errorf("alpha: got %d, want 1023", got)
	}
	// The transparent pixel over a transparent pixel stays zero.
	if got := dst.Pix()[7]; got != 0 {
		t.Errorf("alpha: got %d, want 0", got)
	}
}

func TestDrawOverDepthMismatch(t *testing.T) {
	dst := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA10)
	dst.Pix()[0] = 100
	src := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA8)

	err := bitdepth.DrawOver(dst, src, 0, 0)
	if !errors.Is(err, bitdepth.ErrDepthMismatch) {
		t.Fatalf("got %v, want ErrDepthMismatch", err)
	}
	// A failed blend must not leave a half-blended destination.
	if dst.Pix()[0] != 100 {
		t.Error("dst was modified by a failed DrawOver")
	}
}

// TestDrawOverNoBanding blends a semi-transparent overlay over a smooth
// 10-bit gradient and verifies by adjacent-pixel differences and a
// histogram that no banding appears. The same computation routed
// through an 8-bit intermediate would show visible steps.
func TestDrawOverNoBanding(t *testing.T) {
	const width = 256
	dst := gradient10(width)
	src := bitdepth.NewImage(width, 1, bitdepth.FormatRGBA10)
	for x := 0; x < width; x++ {
		src.Pix()[4*x] = 512
		src.Pix()[4*x+3] = 512
	}
	if err := bitdepth.DrawOver(dst, src, 0, 0); err != nil {
		t.Fatal(err)
	}

	maxDelta := 0
	hist := map[uint16]int{}
	for x := 0; x < width; x++ {
		v := dst.Pix()[4*x]
		hist[v]++
		if x > 0 {
			d := int(v) - int(dst.Pix()[4*(x-1)])
			if d < 0 {
				t.Fatalf("gradient is not monotonic at x=%d", x)
			}
			if d > maxDelta {
				maxDelta = d
			}
		}
	}
	// The blend preserves the smooth ramp: adjacent pixels differ by at
	// most 3 out of 1023 (the ideal step here is 2.004, plus rounding).
	if maxDelta > 3 {
		t.Errorf("max adjacent delta: got %d, want <= 3", maxDelta)
	}
	// The histogram keeps most of the 256 input levels distinct.
	if got := len(hist); got < 200 {
		t.Errorf("distinct levels: got %d, want at least 200", got)
	}

	// For contrast, routing the same gradient through an 8-bit
	// intermediate produces steps of 4: the banding this package avoids.
	down, err := bitdepth.Convert(gradient10(width), bitdepth.Depth8, bitdepth.DitherNone)
	if err != nil {
		t.Fatal(err)
	}
	up, err := bitdepth.Convert(down, bitdepth.Depth10, bitdepth.DitherNone)
	if err != nil {
		t.Fatal(err)
	}
	for x := 1; x < width; x++ {
		if d := int(up.Pix()[4*x]) - int(up.Pix()[4*(x-1)]); d > 2 {
			if maxDelta >= d {
				t.Fatalf("10-bit path is not smoother than the 8-bit round trip: %d vs %d", maxDelta, d)
			}
			return
		}
	}
	t.Fatal("8-bit round trip showed no banding; the comparison is meaningless")
}

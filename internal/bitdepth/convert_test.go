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
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/bitdepth"
)

// gradient10 returns a widthx1 10-bit image whose red channel ramps
// from 0 to 1023. The other channels are opaque black.
func gradient10(width int) *bitdepth.Image {
	img := bitdepth.NewImage(width, 1, bitdepth.FormatRGBA10)
	for x := 0; x < width; x++ {
		img.Pix()[4*x] = uint16(x * 1023 / (width - 1))
		img.Pix()[4*x+3] = 1023
	}
	return img
}

func TestConvertPassthrough10bit(t *testing.T) {
	img := bitdepth.NewImage(2, 1, bitdepth.FormatRGBA10)
	copy(img.Pix(), []uint16{0, 1, 512, 1023, 1023, 513, 2, 0})

	out, err := bitdepth.Convert(img, bitdepth.Depth10, bitdepth.DitherNone)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out.Depth(), bitdepth.Depth10; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	// The samples must pass through bit-exactly. Any 8-bit intermediate
	// would quantize odd values such as 1 and 513 to multiples of 4.
	for i, v := range img.Pix() {
		if out.Pix()[i] != v {
			t.Errorf("Pix()[%d]: got %d, want %d", i, out.Pix()[i], v)
		}
	}
	// The result must be a copy, not an alias.
	out.Pix()[0] = 999
	if img.Pix()[0] != 0 {
		t.Error("Convert must not alias the source pixels")
	}
}

func TestConvertUpconvert8To10Exact(t *testing.T) {
	// Bit replication is the exact inverse of the 10-to-8 rounding
	// division. Guard this mapping with fixed values so a later change
	// cannot silently degrade the upconversion.
	cases := []struct{ in, want uint16 }{
		{0, 0},
		{1, 4},
		{64, 257},
		{128, 514},
		{255, 1023},
	}
	img := bitdepth.NewImage(len(cases), 1, bitdepth.FormatRGBA8)
	for i, c := range cases {
		img.Pix()[4*i] = c.in
		img.Pix()[4*i+3] = 255
	}
	out, err := bitdepth.Convert(img, bitdepth.Depth10, bitdepth.DitherNone)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		if got := out.Pix()[4*i]; got != c.want {
			t.Errorf("upconvert(%d): got %d, want %d", c.in, got, c.want)
		}
		if got := out.Pix()[4*i+3]; got != 1023 {
			t.Errorf("alpha: got %d, want 1023", got)
		}
	}
}

func TestConvertRoundtrip8To10To8(t *testing.T) {
	img := bitdepth.NewImage(256, 1, bitdepth.FormatRGBA8)
	for x := 0; x < 256; x++ {
		img.Pix()[4*x] = uint16(x)
		img.Pix()[4*x+3] = 255
	}
	up, err := bitdepth.Convert(img, bitdepth.Depth10, bitdepth.DitherNone)
	if err != nil {
		t.Fatal(err)
	}
	down, err := bitdepth.Convert(up, bitdepth.Depth8, bitdepth.DitherNone)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range img.Pix() {
		if down.Pix()[i] != v {
			t.Fatalf("roundtrip changed sample %d: got %d, want %d", i, down.Pix()[i], v)
		}
	}
}

func TestConvertDownconvertDither(t *testing.T) {
	// A constant 10-bit value between two 8-bit codes quantizes to a
	// single code without dither, and to a spatial mix with dither.
	const v = 130 // 130*255/1023 = 32.4
	img := bitdepth.NewImage(16, 16, bitdepth.FormatRGBA10)
	for i := 0; i < 256; i++ {
		img.Pix()[4*i] = v
		img.Pix()[4*i+3] = 1023
	}
	plain, err := bitdepth.Convert(img, bitdepth.Depth8, bitdepth.DitherNone)
	if err != nil {
		t.Fatal(err)
	}
	dithered, err := bitdepth.Convert(img, bitdepth.Depth8, bitdepth.DitherOrdered)
	if err != nil {
		t.Fatal(err)
	}
	plainLevels := map[uint16]bool{}
	ditheredLevels := map[uint16]bool{}
	for i := 0; i < 256; i++ {
		plainLevels[plain.Pix()[4*i]] = true
		ditheredLevels[dithered.Pix()[4*i]] = true
	}
	if len(plainLevels) != 1 {
		t.Errorf("DitherNone: got %d levels, want 1", len(plainLevels))
	}
	if len(ditheredLevels) < 2 {
		t.Errorf("DitherOrdered: got %d levels, want at least 2", len(ditheredLevels))
	}
	// The dithered average must stay close to the ideal value.
	var sum int
	for i := 0; i < 256; i++ {
		sum += int(dithered.Pix()[4*i])
	}
	avg := float64(sum) / 256
	if want := float64(v) * 255 / 1023; avg < want-1 || avg > want+1 {
		t.Errorf("dithered average: got %f, want %f±1", avg, want)
	}
}

func TestConvertBatchMixedDepths(t *testing.T) {
	eight := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA8)
	copy(eight.Pix(), []uint16{128, 64, 32, 255})
	ten := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA10)
	copy(ten.Pix(), []uint16{513, 1, 1023, 1023})

	outs, err := bitdepth.ConvertBatch([]*bitdepth.Image{eight, ten, eight}, bitdepth.Depth10, bitdepth.DitherNone)
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 3 {
		t.Fatalf("got %d results, want 3", len(outs))
	}
	// The 8-bit items are upconverted.
	if got := outs[0].Pix()[0]; got != 514 {
		t.Errorf("8-bit item: got %d, want 514", got)
	}
	// The 10-bit item passes through bit-exactly. If the batch were
	// processed at the lowest depth, 513 would come back as 512.
	for i, v := range ten.Pix() {
		if outs[1].Pix()[i] != v {
			t.Errorf("10-bit item Pix()[%d]: got %d, want %d", i, outs[1].Pix()[i], v)
		}
	}
	for i, out := range outs {
		if out.Depth() != bitdepth.Depth10 {
			t.Errorf("outs[%d]: got %v, want 10bit", i, out.Depth())
		}
	}
}

func TestConvertBatchAtomic(t *testing.T) {
	valid := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA8)
	copy(valid.Pix(), []uint16{1, 2, 3, 255})
	invalid := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA8)
	copy(invalid.Pix(), []uint16{1, 2, 300, 255}) // 300 exceeds the 8-bit range.

	outs, err := bitdepth.ConvertBatch([]*bitdepth.Image{valid, invalid}, bitdepth.Depth10, bitdepth.DitherNone)
	if err == nil {
		t.Fatal("expected an error")
	}
	// A failure must not leave behind a half-converted batch.
	if outs != nil {
		t.Errorf("got %d partial results, want nil", len(outs))
	}
	// The inputs must be untouched.
	if valid.Pix()[0] != 1 {
		t.Error("input was modified")
	}
}

func TestConvertInvalidTarget(t *testing.T) {
	img := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA8)
	if _, err := bitdepth.Convert(img, bitdepth.Depth(12), bitdepth.DitherNone); err == nil {
		t.Fatal("expected an error")
	}
}

func TestValidateDetectsDepthMismatch(t *testing.T) {
	img := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA8)
	img.Pix()[2] = 256 // Declared 8-bit, but the sample needs 9 bits.
	if err := bitdepth.Validate(img); err == nil {
		t.Fatal("expected an error")
	}
	img.Pix()[2] = 255
	if err := bitdepth.Validate(img); err != nil {
		t.Fatal(err)
	}
}

func TestCheckTarget(t *testing.T) {
	target := bitdepth.Target{Formats: []bitdepth.Format{bitdepth.FormatRGBA8}}
	if err := bitdepth.CheckTarget(target, bitdepth.FormatRGBA8); err != nil {
		t.Fatal(err)
	}
	err := bitdepth.CheckTarget(target, bitdepth.FormatRGBA10)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, bitdepth.ErrUnsupportedFormat) {
		t.Errorf("error must wrap ErrUnsupportedFormat: %v", err)
	}
	// The error must suggest the supported alternatives instead of
	// silently falling back to 8 bits.
	if got := err.Error(); !strings.Contains(got, "RGBA8") {
		t.Errorf("error must list alternatives, got %q", got)
	}
}

func TestImageMemoryFootprint(t *testing.T) {
	// A 10-bit image must cost exactly twice the bytes of its 8-bit
	// RGBA equivalent, not several times more.
	const w, h = 64, 64
	img := bitdepth.NewImage(w, h, bitdepth.FormatRGBA10)
	if got, want := 2*len(img.Pix()), 2*4*w*h; got != want {
		t.Errorf("got %d bytes, want %d", got, want)
	}
}

func TestConvertConcurrent(t *testing.T) {
	// Convert is stateless and must be safe for concurrent use.
	img := gradient10(64)
	done := make(chan struct{})
	for g := 0; g < 8; g++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for i := 0; i < 100; i++ {
				if _, err := bitdepth.Convert(img, bitdepth.Depth8, bitdepth.DitherOrdered); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	for g := 0; g < 8; g++ {
		<-done
	}
}

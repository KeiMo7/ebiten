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
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/bitdepth"
)

// plane10 returns a little-endian 16-bit plane with the given samples.
func plane10(samples ...uint16) []byte {
	p := make([]byte, 2*len(samples))
	for i, s := range samples {
		p[2*i] = byte(s)
		p[2*i+1] = byte(s >> 8)
	}
	return p
}

func TestYUV420P10LEToRGBA10(t *testing.T) {
	// 2x2 pixels, neutral gray at limited-range mid level.
	y := plane10(512, 512, 512, 512)
	u := plane10(512)
	v := plane10(512)
	img, err := bitdepth.YUV420P10LEToRGBA10(y, u, v, 2, 2, 2, 1, bitdepth.YUVColorSpaceBT709)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := img.Depth(), bitdepth.Depth10; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
	// 1.164*(512-64) = 521.5; neutral chroma keeps R == G == B.
	for i := 0; i < 4; i++ {
		r, g, b := img.Pix()[4*i], img.Pix()[4*i+1], img.Pix()[4*i+2]
		if r != g || g != b {
			t.Errorf("pixel %d: got (%d, %d, %d), want equal channels", i, r, g, b)
		}
		if r < 520 || r > 523 {
			t.Errorf("pixel %d: got %d, want about 522", i, r)
		}
		if got := img.Pix()[4*i+3]; got != 1023 {
			t.Errorf("alpha: got %d, want 1023", got)
		}
	}
}

func TestYUV420P10LEToRGBA10Chroma(t *testing.T) {
	y := plane10(512, 512, 512, 512)
	u := plane10(512)
	v := plane10(612) // +100 red-difference chroma.
	img, err := bitdepth.YUV420P10LEToRGBA10(y, u, v, 2, 2, 2, 1, bitdepth.YUVColorSpaceBT709)
	if err != nil {
		t.Fatal(err)
	}
	// BT.709: R += 1.793*100, G -= 0.533*100, B unchanged.
	r, g, b := img.Pix()[0], img.Pix()[1], img.Pix()[2]
	if r < 700 || r > 702 {
		t.Errorf("R: got %d, want about 701", r)
	}
	if g < 467 || g > 470 {
		t.Errorf("G: got %d, want about 469", g)
	}
	if b < 520 || b > 523 {
		t.Errorf("B: got %d, want about 522", b)
	}
}

func TestYUV420P10LEToRGBA10Errors(t *testing.T) {
	y := plane10(512, 512, 512, 512)
	c := plane10(512)
	if _, err := bitdepth.YUV420P10LEToRGBA10(y, c, c, 2, 2, 2, 1, bitdepth.YUVColorSpace(99)); err == nil {
		t.Error("expected an error for an invalid color space")
	}
	if _, err := bitdepth.YUV420P10LEToRGBA10(y[:2], c, c, 2, 2, 2, 1, bitdepth.YUVColorSpaceBT709); err == nil {
		t.Error("expected an error for a short luma plane")
	}
	if _, err := bitdepth.YUV420P10LEToRGBA10(y, c, c, 0, 2, 2, 1, bitdepth.YUVColorSpaceBT709); err == nil {
		t.Error("expected an error for an invalid size")
	}
}

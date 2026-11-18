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

// DrawOver draws src over dst at the position (dx, dy) using the
// source-over compositing rule. Both images must have the same bit
// depth; DrawOver never converts implicitly, so a 10-bit destination
// is never truncated to 8 bits while blending. All intermediate math
// is done at full precision, so semi-transparent overlays on 10-bit
// content do not introduce quantization steps.
//
// DrawOver validates both images before mutating dst, so a failure
// leaves dst untouched.
func DrawOver(dst, src *Image, dx, dy int) error {
	logf("draw-over: enter: dst=%s src=%s", dst.Depth(), src.Depth())
	if dst.Depth() != src.Depth() {
		err := fmt.Errorf("bitdepth: cannot blend %s over %s: %w", src.Depth(), dst.Depth(), ErrDepthMismatch)
		logf("draw-over: failure: %v", err)
		return err
	}
	if err := Validate(dst); err != nil {
		logf("draw-over: failure: %v", err)
		return err
	}
	if err := Validate(src); err != nil {
		logf("draw-over: failure: %v", err)
		return err
	}

	max := uint64(dst.Depth().Max())
	for sy := 0; sy < src.height; sy++ {
		ty := sy + dy
		if ty < 0 || ty >= dst.height {
			continue
		}
		for sx := 0; sx < src.width; sx++ {
			tx := sx + dx
			if tx < 0 || tx >= dst.width {
				continue
			}
			s := 4 * (sy*src.width + sx)
			d := 4 * (ty*dst.width + tx)
			sa := uint64(src.pix[s+3])
			da := uint64(dst.pix[d+3])
			outA := sa + (da*(max-sa)+max/2)/max
			if outA == 0 {
				for c := 0; c < 4; c++ {
					dst.pix[d+c] = 0
				}
				continue
			}
			for c := 0; c < 3; c++ {
				sc := uint64(src.pix[s+c])
				dc := uint64(dst.pix[d+c])
				num := sc*sa*max + dc*da*(max-sa)
				den := outA * max
				dst.pix[d+c] = uint16((num + den/2) / den)
			}
			dst.pix[d+3] = uint16(outA)
		}
	}
	logf("draw-over: exit: depth=%s", dst.Depth())
	return nil
}

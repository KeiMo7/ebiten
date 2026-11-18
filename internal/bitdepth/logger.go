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
	"io"
	"log"
)

// Logger receives the bit-depth decisions of this package: one line
// when a path is entered, one when it exits, and one when it fails.
// Log lines contain only depths, formats, dither modes, and image
// dimensions; they never contain file paths, credentials, or pixel
// data. By default the logs are discarded, matching the rest of
// Ebitengine, which is quiet unless debugging is enabled.
//
// Replacing Logger is not synchronized; set it once before use.
var Logger = log.New(io.Discard, "bitdepth: ", 0)

func logf(format string, args ...any) {
	Logger.Printf(format, args...)
}

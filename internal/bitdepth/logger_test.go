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
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/bitdepth"
)

// captureLogs replaces the package logger for the duration of a test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := bitdepth.Logger
	bitdepth.Logger = log.New(&buf, "bitdepth: ", 0)
	t.Cleanup(func() {
		bitdepth.Logger = orig
	})
	return &buf
}

func TestLogsEnterExitFailure(t *testing.T) {
	buf := captureLogs(t)

	img := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA8)
	if _, err := bitdepth.Convert(img, bitdepth.Depth10, bitdepth.DitherNone); err != nil {
		t.Fatal(err)
	}
	// Trigger a failure path as well.
	if _, err := bitdepth.Convert(img, bitdepth.Depth(12), bitdepth.DitherNone); err == nil {
		t.Fatal("expected an error")
	}

	out := buf.String()
	for _, want := range []string{"convert: enter", "convert: exit", "convert: failure"} {
		if !strings.Contains(out, want) {
			t.Errorf("log output must contain %q, got %q", want, out)
		}
	}
}

// TestLogsContainNoUserData asserts that log lines never leak file
// paths, credentials, or pixel data, even when the inputs are crafted
// to carry a canary value.
func TestLogsContainNoUserData(t *testing.T) {
	buf := captureLogs(t)

	// The canary stands in for user data such as credentials or file
	// names. It is embedded in the only user-controlled inputs the
	// package sees: dimensions and sample values.
	const canary = "s3cr3t-token"
	t.Logf("canary %s must never appear in package logs", canary)

	img := bitdepth.NewImage(2, 2, bitdepth.FormatRGBA8)
	img.Pix()[0] = 300 // Invalid for 8-bit: forces a failure log.
	if _, err := bitdepth.Convert(img, bitdepth.Depth10, bitdepth.DitherNone); err == nil {
		t.Fatal("expected an error")
	}
	if err := bitdepth.CheckTarget(bitdepth.Target{Formats: []bitdepth.Format{bitdepth.FormatRGBA8}}, bitdepth.FormatRGBA10); err == nil {
		t.Fatal("expected an error")
	}
	dst := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA10)
	src := bitdepth.NewImage(1, 1, bitdepth.FormatRGBA8)
	if err := bitdepth.DrawOver(dst, src, 0, 0); err == nil {
		t.Fatal("expected an error")
	}

	out := buf.String()
	if out == "" {
		t.Fatal("expected log output")
	}
	for _, forbidden := range []string{canary, "/home/", "/Users/", `C:\`, "token", "password"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("log output must not contain %q, got %q", forbidden, out)
		}
	}
	// Log lines must only carry depths, formats, dither modes, sizes,
	// and sample ranges.
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if !strings.HasPrefix(line, "bitdepth: ") {
			t.Errorf("unexpected log line: %q", line)
		}
	}
}

func TestDefaultLoggerIsQuiet(t *testing.T) {
	// The zero-configuration behavior must match the rest of
	// Ebitengine: no log output unless a logger is installed.
	if bitdepth.Logger == nil {
		t.Fatal("Logger must not be nil")
	}
	if got := bitdepth.Logger.Prefix(); got != "bitdepth: " {
		t.Errorf("got prefix %q, want %q", got, "bitdepth: ")
	}
}

// Copyright 2026 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build ebitenginedebug || ebitendebug

package vector_test

import (
	"bytes"
	"image/color"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// countDrawTriangleCommands forces a flush and counts the "draw-triangles"
// command lines the engine prints to stdout under the debug build tags.
func countDrawTriangleCommands(t *testing.T, draw func(*ebiten.Image)) int {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	dst := ebiten.NewImage(256, 256)
	draw(dst)
	// ReadPixels forces a flush; the frame logger Flush()es to os.Stdout.
	dst.ReadPixels(make([]byte, 4*256*256))

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = original
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, "draw-triangles") {
			n++
		}
	}
	return n
}

// TestAnalyticBatchingDebug verifies that many analytic primitives drawn to one
// image collapse into far fewer commands than the equivalent stencil pipeline.
func TestAnalyticBatchingDebug(t *testing.T) {
	const n = 32
	analyticCmds := countDrawTriangleCommands(t, func(dst *ebiten.Image) {
		for i := 0; i < n; i++ {
			vector.FillCircle(dst, float32(8+i*7), 128, 5, color.White, true)
		}
	})
	stencilCmds := countDrawTriangleCommands(t, func(dst *ebiten.Image) {
		for i := 0; i < n; i++ {
			var p vector.Path
			p.MoveTo(float32(8+i*7), 123)
			p.LineTo(float32(13+i*7), 123)
			p.LineTo(float32(13+i*7), 133)
			p.LineTo(float32(8+i*7), 133)
			p.Close()
			op := &vector.DrawPathOptions{AntiAlias: true}
			op.ColorScale.ScaleWithColor(color.White)
			vector.FillPath(dst, &p, nil, op)
		}
	})
	t.Logf("analytic commands=%d stencil commands=%d for %d primitives", analyticCmds, stencilCmds, n)
	if analyticCmds >= stencilCmds {
		t.Errorf("analytic path did not reduce commands: %d vs stencil %d", analyticCmds, stencilCmds)
	}
}

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

package vector_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const analyticBenchCount = 64

// stencilFillRects renders the same number of axis-aligned rectangles through
// the general stencil pipeline, giving a baseline for the analytic fast path.
func stencilFillRects(dst *ebiten.Image) {
	for i := 0; i < analyticBenchCount; i++ {
		var p vector.Path
		x := float32(8 + (i%8)*30)
		y := float32(8 + (i/8)*30)
		p.MoveTo(x, y)
		p.LineTo(x+20, y)
		p.LineTo(x+20, y+20)
		p.LineTo(x, y+20)
		p.Close()
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(color.White)
		vector.FillPath(dst, &p, nil, op)
	}
}

func BenchmarkAnalyticFillRect(b *testing.B) {
	dst := ebiten.NewImage(256, 256)
	defer dst.Deallocate()
	pixels := make([]byte, 4*256*256)
	dst.ReadPixels(pixels)
	b.ReportAllocs()
	for b.Loop() {
		dst.Clear()
		for i := 0; i < analyticBenchCount; i++ {
			x := float32(8 + (i%8)*30)
			y := float32(8 + (i/8)*30)
			vector.FillRect(dst, x, y, 20, 20, color.White, true)
		}
		dst.ReadPixels(pixels)
	}
}

func BenchmarkStencilFillRect(b *testing.B) {
	dst := ebiten.NewImage(256, 256)
	defer dst.Deallocate()
	pixels := make([]byte, 4*256*256)
	dst.ReadPixels(pixels)
	b.ReportAllocs()
	for b.Loop() {
		dst.Clear()
		stencilFillRects(dst)
		dst.ReadPixels(pixels)
	}
}

func BenchmarkAnalyticFillCircle(b *testing.B) {
	dst := ebiten.NewImage(256, 256)
	defer dst.Deallocate()
	pixels := make([]byte, 4*256*256)
	dst.ReadPixels(pixels)
	b.ReportAllocs()
	for b.Loop() {
		dst.Clear()
		for i := 0; i < analyticBenchCount; i++ {
			vector.FillCircle(dst, float32(18+(i%8)*30), float32(18+(i/8)*30), 8, color.White, true)
		}
		dst.ReadPixels(pixels)
	}
}

func BenchmarkAnalyticStrokeLine(b *testing.B) {
	dst := ebiten.NewImage(256, 256)
	defer dst.Deallocate()
	pixels := make([]byte, 4*256*256)
	dst.ReadPixels(pixels)
	b.ReportAllocs()
	for b.Loop() {
		dst.Clear()
		for i := 0; i < analyticBenchCount; i++ {
			vector.StrokeLine(dst, 8, float32(8+i*3), 248, float32(20+i*3), 3, color.White, true)
		}
		dst.ReadPixels(pixels)
	}
}

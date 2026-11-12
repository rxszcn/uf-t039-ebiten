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

package vector_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// maxPixelDiff returns the largest per-channel absolute difference between two
// RGBA buffers. Zero means the two renders are byte-identical.
func maxPixelDiff(a, b []byte) int {
	m := 0
	for i := range a {
		d := int(a[i]) - int(b[i])
		if d < 0 {
			d = -d
		}
		if d > m {
			m = d
		}
	}
	return m
}

func readback(t *testing.T, dst *ebiten.Image) []byte {
	t.Helper()
	b := dst.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	dst.ReadPixels(pix)
	return pix
}

// The analytic fast path must reproduce the stencil anti-aliasing pipeline
// byte-for-byte for the convex primitives it replaces. Both implementations
// quantize coverage to the same eight fixed sub-pixel samples, so these
// comparisons require exact equality rather than a tolerance.

func TestAnalyticVsStencilFillRect(t *testing.T) {
	cases := []struct {
		x, y, w, h float32
	}{
		{4, 4, 12, 8},
		{4.5, 4.5, 12, 8},
		{4.125, 3.375, 7.75, 9.25},
		{0, 0, 16, 16},
		{-2.5, 1.5, 10, 7},
		{2.0625, 2.0625, 11.875, 11.875},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			got := ebiten.NewImage(20, 20)
			want := ebiten.NewImage(20, 20)
			vector.AnalyticFillRectForTest(got, float64(c.x), float64(c.y), float64(c.w), float64(c.h), color.White)
			vector.FillRect(want, c.x, c.y, c.w, c.h, color.White, true)
			if m := maxPixelDiff(readback(t, got), readback(t, want)); m != 0 {
				t.Errorf("FillRect %+v: max diff %d, want 0", c, m)
			}
		})
	}
}

func TestAnalyticVsStencilStrokeLine(t *testing.T) {
	cases := []struct {
		x0, y0, x1, y1, width float32
	}{
		{2, 6, 18, 6, 4},
		{6, 2, 6, 18, 4},
		{2, 2, 18, 18, 4},
		{1.5, 3.375, 17.5, 11.625, 2},
		{3, 10, 15, 4, 3.5},
		{-2, -2, 20, 18, 5},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			got := ebiten.NewImage(20, 20)
			want := ebiten.NewImage(20, 20)
			vector.AnalyticStrokeLineForTest(got, float64(c.x0), float64(c.y0), float64(c.x1), float64(c.y1), float64(c.width), color.White)
			vector.StrokeLine(want, c.x0, c.y0, c.x1, c.y1, c.width, color.White, true)
			if m := maxPixelDiff(readback(t, got), readback(t, want)); m != 0 {
				t.Errorf("StrokeLine %+v: max diff %d, want 0", c, m)
			}
		})
	}
}

func TestAnalyticVsStencilCircle(t *testing.T) {
	t.Run("fill", func(t *testing.T) {
		// Integer and sub-pixel centers across small and large radii.
		for _, r := range []float32{0.5, 1, 2, 4, 8, 13.5, 30} {
			for _, off := range []float32{0, 0.375} {
				got := ebiten.NewImage(72, 72)
				want := ebiten.NewImage(72, 72)
				vector.AnalyticFillCircleForTest(got, 36+float64(off), 36, float64(r), color.White)
				vector.FillCircle(want, 36+off, 36, r, color.White, true)
				if m := maxPixelDiff(readback(t, got), readback(t, want)); m != 0 {
					t.Errorf("FillCircle r=%v off=%v: max diff %d, want 0", r, off, m)
				}
			}
		}
	})
	t.Run("stroke", func(t *testing.T) {
		for _, tc := range []struct{ r, w float32 }{
			{8, 4}, {10, 2}, {13, 6}, {5, 10}, {15, 3}, {2.5, 1},
		} {
			got := ebiten.NewImage(40, 40)
			want := ebiten.NewImage(40, 40)
			vector.AnalyticStrokeCircleForTest(got, 20, 20, float64(tc.r), float64(tc.w), color.White)
			vector.StrokeCircle(want, 20, 20, tc.r, tc.w, color.White, true)
			if m := maxPixelDiff(readback(t, got), readback(t, want)); m != 0 {
				t.Errorf("StrokeCircle %+v: max diff %d, want 0", tc, m)
			}
		}
	})
}

// TestAnalyticVsStencilSemiTransparent ensures premultiplied color scaling is
// applied identically to the stencil path.
func TestAnalyticVsStencilSemiTransparent(t *testing.T) {
	clr := color.RGBA{R: 0x60, G: 0x30, B: 0x18, A: 0x80}
	got := ebiten.NewImage(24, 24)
	want := ebiten.NewImage(24, 24)
	vector.AnalyticFillCircleForTest(got, 12, 12, 8, clr)
	vector.FillCircle(want, 12, 12, 8, clr, true)
	if m := maxPixelDiff(readback(t, got), readback(t, want)); m != 0 {
		t.Errorf("semi-transparent FillCircle: max diff %d, want 0", m)
	}
}

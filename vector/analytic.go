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

package vector

import (
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// analyticIndicesScratch holds the two triangles covering every primitive's
// bounding-box quad. Each primitive is one quad (four vertices, two triangles);
// the fragment shader decides coverage, so the quad only has to cover the
// primitive's bounding box.
var analyticIndicesScratch = []uint32{0, 1, 2, 1, 2, 3}

// drawAnalyticPrimitive enqueues one convex primitive rendered entirely by the
// analytic anti-aliasing shader. quadBounds and custom are expressed in the
// destination image's caller coordinate frame.
//
// The engine translates quad vertex positions by the destination origin itself,
// so the quad keeps caller coordinates. Inside the shader,
// dstPos - imageDstOrigin() yields coordinates relative to the destination's
// bounds origin; the geometry carried in custom therefore has that same origin
// subtracted here, while radius and direction-vector channels are untouched.
//
// Because the shader has no source image and reads no uniform, calls drawn to
// the same destination with the same blend mode are merged into one graphics
// command by the engine's command queue.
func drawAnalyticPrimitive(dst *ebiten.Image, quadBounds image.Rectangle, kind analyticPrimitiveKind, halfWidth float64, custom [4]float64, clr ebiten.ColorScale, blend ebiten.Blend) {
	shader, err := ensureAnalyticPrimitiveShader()
	if err != nil {
		panic(fmt.Sprintf("vector: failed to create analytic primitive shader: %v", err))
	}

	// The shader's local frame has its origin at the destination bounds origin.
	// Subtract it in float64 so a large bounds origin (up to 1<<24 in tests) does
	// not lose sub-pixel precision when the result is narrowed to float32.
	ox := float64(dst.Bounds().Min.X)
	oy := float64(dst.Bounds().Min.Y)
	localCustom := custom
	localCustom[0] -= ox
	localCustom[1] -= oy
	// The second position pair exists only for the rectangle's far corner.
	if kind == analyticKindRect {
		localCustom[2] -= ox
		localCustom[3] -= oy
	}

	// Quad vertices stay in caller coordinates; the engine applies the origin.
	minX := float32(quadBounds.Min.X)
	minY := float32(quadBounds.Min.Y)
	maxX := float32(quadBounds.Max.X)
	maxY := float32(quadBounds.Max.Y)

	v := func(x, y float32) ebiten.Vertex {
		return ebiten.Vertex{
			DstX:    x,
			DstY:    y,
			SrcX:    float32(kind),
			SrcY:    float32(halfWidth),
			ColorR:  clr.R(),
			ColorG:  clr.G(),
			ColorB:  clr.B(),
			ColorA:  clr.A(),
			Custom0: float32(localCustom[0]),
			Custom1: float32(localCustom[1]),
			Custom2: float32(localCustom[2]),
			Custom3: float32(localCustom[3]),
		}
	}
	vs := []ebiten.Vertex{
		v(minX, minY),
		v(maxX, minY),
		v(minX, maxY),
		v(maxX, maxY),
	}

	op := &ebiten.DrawTrianglesShaderOptions{}
	op.Blend = blend
	dst.DrawTrianglesShader32(vs, analyticIndicesScratch, shader, op)
}

// analyticFillRect renders an anti-aliased axis-aligned rectangle with the
// analytic shader. Coordinates and size use the same conventions as fillRect.
func analyticFillRect(dst *ebiten.Image, x, y, width, height float64, clr ebiten.ColorScale, blend ebiten.Blend) {
	// Normalize a negative width or height, mirroring fillRect's callers.
	if width < 0 {
		x += width
		width = -width
	}
	if height < 0 {
		y += height
		height = -height
	}

	x0 := snapToSubpixel(point{x: x, y: 0}).x
	y0 := snapToSubpixel(point{x: 0, y: y}).y
	x1 := snapToSubpixel(point{x: x + width, y: 0}).x
	y1 := snapToSubpixel(point{x: 0, y: y + height}).y

	bounds := boundsAround(x0, y0, x1, y1)
	drawAnalyticPrimitive(dst, bounds, analyticKindRect, 0,
		[4]float64{x0, y0, x1, y1}, clr, blend)
}

// analyticStrokeLine renders an anti-aliased butt-capped straight segment with
// the analytic shader. The endpoints are snapped to the sub-pixel grid so the
// coverage decision matches the stencil pipeline's rasterization.
func analyticStrokeLine(dst *ebiten.Image, x0, y0, x1, y1, strokeWidth float64, clr ebiten.ColorScale, blend ebiten.Blend) {
	a := snapToSubpixel(point{x: x0, y: y0})
	b := snapToSubpixel(point{x: x1, y: y1})
	halfWidth := strokeWidth / 2

	dx := b.x - a.x
	dy := b.y - a.y
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	// Unit normal to the segment.
	nx := -dy / length
	ny := dx / length

	qx0 := min(a.x-nx*halfWidth, b.x-nx*halfWidth, a.x+nx*halfWidth, b.x+nx*halfWidth)
	qx1 := max(a.x-nx*halfWidth, b.x-nx*halfWidth, a.x+nx*halfWidth, b.x+nx*halfWidth)
	qy0 := min(a.y-ny*halfWidth, b.y-ny*halfWidth, a.y+ny*halfWidth, b.y+ny*halfWidth)
	qy1 := max(a.y-ny*halfWidth, b.y-ny*halfWidth, a.y+ny*halfWidth, b.y+ny*halfWidth)

	bounds := boundsAround(qx0, qy0, qx1, qy1)
	drawAnalyticPrimitive(dst, bounds, analyticKindStrokeLine, halfWidth,
		[4]float64{a.x, a.y, dx, dy}, clr, blend)
}

// analyticFillCircle renders an anti-aliased filled disk with the analytic
// shader. A true circle is used rather than the Bézier approximation of the
// stencil pipeline; both produce the same k/8 coverage quantization.
func analyticFillCircle(dst *ebiten.Image, cx, cy, r float64, clr ebiten.ColorScale, blend ebiten.Blend) {
	c := snapToSubpixel(point{x: cx, y: cy})
	bounds := boundsAround(c.x-r, c.y-r, c.x+r, c.y+r)
	drawAnalyticPrimitive(dst, bounds, analyticKindCircle, 0,
		[4]float64{c.x, c.y, r, 0}, clr, blend)
}

// analyticStrokeCircle renders an anti-aliased circular ring with the analytic
// shader. strokeWidth >= 2r degenerates to a filled disk, matching strokeCircle.
func analyticStrokeCircle(dst *ebiten.Image, cx, cy, r, strokeWidth float64, clr ebiten.ColorScale, blend ebiten.Blend) {
	if strokeWidth >= 2*r {
		analyticFillCircle(dst, cx, cy, r+strokeWidth/2, clr, blend)
		return
	}
	c := snapToSubpixel(point{x: cx, y: cy})
	outer := r + strokeWidth/2
	inner := r - strokeWidth/2
	bounds := boundsAround(c.x-outer, c.y-outer, c.x+outer, c.y+outer)
	drawAnalyticPrimitive(dst, bounds, analyticKindStrokeCircle, 0,
		[4]float64{c.x, c.y, inner, outer}, clr, blend)
}

// boundsAround returns the integer pixel rectangle covering every fragment the
// primitive may touch. It is expanded by a whole pixel so partially covered
// edge pixels are included regardless of sub-pixel placement.
func boundsAround(x0, y0, x1, y1 float64) image.Rectangle {
	return image.Rect(
		int(math.Floor(x0))-1,
		int(math.Floor(y0))-1,
		int(math.Ceil(x1))+1,
		int(math.Ceil(y1))+1,
	)
}

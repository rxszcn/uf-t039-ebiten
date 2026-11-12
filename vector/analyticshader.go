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
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// analyticPrimitiveKind selects the analytic coverage test performed by the
// anti-aliased primitive shader. Every kind is a convex primitive, so coverage
// is the number of the fixed eight sub-pixel samples that fall inside it.
type analyticPrimitiveKind float32

const (
	analyticKindRect analyticPrimitiveKind = iota + 1
	analyticKindCircle
	analyticKindStrokeCircle
	analyticKindStrokeLine
)

var (
	analyticPrimitiveShader     *ebiten.Shader
	analyticPrimitiveShaderErr  error
	analyticPrimitiveShaderOnce sync.Once
)

func ensureAnalyticPrimitiveShader() (*ebiten.Shader, error) {
	analyticPrimitiveShaderOnce.Do(func() {
		analyticPrimitiveShader, analyticPrimitiveShaderErr = ebiten.NewShader([]byte(analyticPrimitiveShaderSrc))
	})
	return analyticPrimitiveShader, analyticPrimitiveShaderErr
}

// The shader samples no image and reads no uniform: every parameter travels in
// per-vertex attributes, so successive primitives that share the destination
// and blend mode merge into one graphics command.
//
// Attribute packing (identical on all four vertices of a primitive):
//   - color (ColorR..A): the premultiplied primitive color.
//   - src0Pos.x: the analyticPrimitiveKind discriminator.
//   - src0Pos.y: the half stroke width (used only by the stroke line).
//   - custom.xyzw: kind-specific geometry (see analyticInside).
//
// The kind is converted to an integer local before branching: comparing a
// varying float against a constant directly does not dispatch reliably in the
// generated shader, while an integer local variable does.
//
// The eight sample positions reproduce the MSAA offsets of the stencil pipeline
// (see offsetAndColorsAA). Counting how many samples lie inside the convex
// primitive yields the same discrete k/8 coverage the stencil resolve shaders
// compute.
//
//ebitengine:shadersource
const analyticPrimitiveShaderSrc = `//kage:unit pixels

package main

// sampleCount is the number of sub-pixel coverage samples.
const sampleCount = 8

func Fragment(dstPos vec4, src0Pos vec2, color vec4, custom vec4) vec4 {
	kind := int(src0Pos.x)
	halfWidth := src0Pos.y
	// dstPos is absolute on the destination texture, while the geometry in custom
	// is image-local. Subtract the destination region origin (constant for every
	// primitive drawn to the same image, so it does not split same-image batches).
	base := dstPos.xy - imageDstOrigin()

	var coverage float
	// The eight fixed MSAA sample offsets, matching offsetAndColorsAA.
	coverage += boolToFloat(analyticInside(kind, halfWidth, base+vec2(1.0/16.0, -3.0/16.0), custom))
	coverage += boolToFloat(analyticInside(kind, halfWidth, base+vec2(-1.0/16.0, 3.0/16.0), custom))
	coverage += boolToFloat(analyticInside(kind, halfWidth, base+vec2(5.0/16.0, 1.0/16.0), custom))
	coverage += boolToFloat(analyticInside(kind, halfWidth, base+vec2(-3.0/16.0, -5.0/16.0), custom))
	coverage += boolToFloat(analyticInside(kind, halfWidth, base+vec2(-5.0/16.0, 5.0/16.0), custom))
	coverage += boolToFloat(analyticInside(kind, halfWidth, base+vec2(-7.0/16.0, -1.0/16.0), custom))
	coverage += boolToFloat(analyticInside(kind, halfWidth, base+vec2(3.0/16.0, 7.0/16.0), custom))
	coverage += boolToFloat(analyticInside(kind, halfWidth, base+vec2(7.0/16.0, -7.0/16.0), custom))

	v := coverage / float(sampleCount)
	if v == 0.0 {
		discard()
	}
	return vec4(color.rgb*v, color.a*v)
}

func analyticInside(kind int, halfWidth float, p vec2, c vec4) bool {
	if kind == 1 {
		// Axis-aligned rectangle: [c.x, c.z) x [c.y, c.w).
		return p.x >= c.x && p.x < c.z && p.y >= c.y && p.y < c.w
	}
	if kind == 2 {
		// Filled disk centered at (c.x, c.y), radius c.z.
		d := p - c.xy
		return dot(d, d) <= c.z*c.z
	}
	if kind == 3 {
		// Ring centered at (c.x, c.y), inner radius c.z, outer radius c.w.
		d := p - c.xy
		l2 := dot(d, d)
		return l2 >= c.z*c.z && l2 <= c.w*c.w
	}
	// Butt-capped segment stroke. c.xy is the start point and c.zw the segment
	// vector; halfWidth is the half stroke width.
	ap := p - c.xy
	len2 := dot(c.zw, c.zw)
	t := dot(ap, c.zw) / len2
	if t < 0.0 || t > 1.0 {
		return false
	}
	// Squared perpendicular distance from p to the center line.
	perp := vec2(-c.w, c.z)
	cross2 := dot(ap, perp)
	d2 := cross2 * cross2 / len2
	return d2 <= halfWidth*halfWidth
}

func boolToFloat(b bool) float {
	if b {
		return 1.0
	}
	return 0.0
}
`

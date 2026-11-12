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

//go:build !ebitenginedebug && !ebitendebug

package vector_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Without the debug build tags the engine does not record per-command logs, so
// the command-count assertion is a no-op; the pixel tests cover correctness.
func TestAnalyticBatchingDebug(t *testing.T) {
	t.Skip("command counting requires -tags=ebitenginedebug")
}

func countDrawTriangleCommands(t *testing.T, draw func(*ebiten.Image)) int {
	t.Helper()
	t.Skip("command counting requires -tags=ebitenginedebug")
	return 0
}

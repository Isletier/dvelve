/*
The MIT License (MIT)

Copyright (c) 2026 Aliaksei Astrouski

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package dvap

import (
	"fmt"
	"strings"

	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/service/api"
)

// MaxGoroutines is the maximum number of goroutines included in a single
// DVAP broadcast, applied after filtering. At ~70 bytes per token this caps
// the goroutine portion of a broadcast at ~35 KB, keeping wire traffic
// predictable even in programs with thousands of goroutines.
const MaxGoroutines = 512

// FormatState converts the current goroutine and breakpoint state into the
// DVAP wire format — a single space-separated string of tokens:
//
//	selected:{goid}
//	thread:{goid}:{file}:{line}:{threadID}
//	bp:{id}:{file}:{line}:{funcName}:{nonconditional}:{enabled}
//
// selectedGoid is the goroutine currently focused by the debugger (-1 = none).
// threadID is 0 when the goroutine is not scheduled on any OS thread.
//
// Goroutines are filtered: only Grunnable, Grunning, and Gsyscall goroutines
// are emitted. Gwaiting (parked — channel, timer, mutex), Gdead, and Gidle
// goroutines carry no useful navigation information and are omitted. The
// selected goroutine is always included regardless of its status.
// At most MaxGoroutines goroutines are emitted after filtering.
//
// Internal breakpoints (ID < 0) are omitted.
func FormatState(goroutines []*api.Goroutine, breakpoints []*api.Breakpoint, selectedGoid int64) string {
	var sb strings.Builder

	if selectedGoid >= 0 {
		fmt.Fprintf(&sb, "selected:%d ", selectedGoid)
	}

	count := 0
	for _, g := range goroutines {
		if g.Unreadable != "" {
			continue
		}
		// Always include the selected goroutine so it is never omitted.
		// For all others, skip parked (Gwaiting), dead (Gdead), and idle (Gidle)
		// goroutines — these carry no useful navigation information.
		if g.ID != selectedGoid &&
			(g.Status == proc.Gwaiting || g.Status == proc.Gdead || g.Status == proc.Gidle) {
			continue
		}
		if count >= MaxGoroutines {
			break
		}
		fmt.Fprintf(&sb, "thread:%d:%s:%d:%d ",
			g.ID, g.CurrentLoc.File, g.CurrentLoc.Line, g.ThreadID)
		count++
	}

	for _, bp := range breakpoints {
		if bp.ID < 0 {
			continue
		}
		nonconditional := bp.Cond == "" && bp.HitCond == ""
		fmt.Fprintf(&sb, "bp:%d:%s:%d:%s:%t:%t ",
			bp.ID, bp.File, bp.Line, bp.FunctionName,
			nonconditional, !bp.Disabled)
	}

	return strings.TrimSpace(sb.String())
}

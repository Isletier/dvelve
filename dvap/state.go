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

const (
	fs = ";;" // field separator (within a record)
	rs = "||" // record separator (between records)
)

// Thread type labels used in the wire format.
const (
	threadTypeGoroutine = "goroutine"
	threadTypeThread    = "thread"
)

func formatBool(value bool) string {
	if !value {
		return "False"
	}
	return "True"
}

// FormatState converts the current debugger state into the DVAP wire format —
// a sequence of records separated by "||", fields within each record separated
// by ";;":
//
//	selected;;{goid};;goroutine||                              goroutine is focused
//	selected;;{threadID};;thread||                             OS thread fallback (no goroutine)
//	thread;;{goid};;goroutine;;{file};;{line};;{osThreadID}||
//	thread;;{osThreadID};;thread;;{file};;{line};;{goroutineID}||
//	bp;;{id};;{file};;{line};;{nonconditional};;{enabled}||
//
// Selection: goroutine wins over thread; thread is emitted only when
// selectedGoid < 0 and selectedThreadID > 0.
//
// The cross-reference field in each thread record links the two layers:
// for goroutine records it is the OS thread ID (0 = not scheduled);
// for OS thread records it is the goroutine ID (0 = idle thread).
//
// Goroutine filtering: only Grunnable, Grunning, and Gsyscall goroutines are
// emitted. Gwaiting, Gdead, and Gidle goroutines are omitted. The selected
// goroutine is always included regardless of its status.
// At most MaxGoroutines goroutines are emitted after filtering.
//
// All OS threads from the threads slice are emitted without filtering.
//
// Internal breakpoints (ID < 0) are omitted.

func FormatState(goroutines []*api.Goroutine, threads []*api.Thread, breakpoints []*api.Breakpoint, selectedGoid int64, selectedThreadID int) string {
	var sb strings.Builder

	// selected record: goroutine preferred, OS thread as fallback.
	if selectedGoid >= 0 {
		fmt.Fprintf(&sb, "selected%s%d%s%s%s", fs, selectedGoid, fs, threadTypeGoroutine,  rs)
	} else if selectedThreadID > 0 {
		fmt.Fprintf(&sb, "selected%s%d%s%s%s",  fs, selectedThreadID, fs, threadTypeThread, rs)
	}

	// Goroutine records.
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

		fmt.Fprintf(&sb, "thread%s%d%s%s%s%s%s%d%s%d%s",
			fs, g.ID, fs, threadTypeGoroutine, fs, g.CurrentLoc.File, fs, g.CurrentLoc.Line, fs, g.ThreadID, rs)
		count++
	}

	// OS thread records — all threads, no filtering.
	for _, th := range threads {
		fmt.Fprintf(&sb, "thread%s%d%s%s%s%s%s%d%s%d%s",
			fs, th.ID, fs, threadTypeThread, fs, th.File, fs, th.Line, fs, th.GoroutineID, rs)
	}

	for _, bp := range breakpoints {
		if bp.ID < 0 {
			continue
		}

		nonconditional := bp.Cond == "" && bp.HitCond == ""
		fmt.Fprintf(&sb, "bp%s%d%s%s%s%d%s%s%s%s%s",
			fs, bp.ID, fs, bp.File, fs, bp.Line, fs, formatBool(nonconditional), 
			fs, formatBool(!bp.Disabled), rs)
	}

	return sb.String()
}

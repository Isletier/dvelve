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
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package dvap

import (
	"strings"
	"testing"

	"github.com/go-delve/delve/service/api"
)

func goroutine(id int64, file string, line int, threadID int, status uint64) *api.Goroutine {
	return &api.Goroutine{
		ID:         id,
		CurrentLoc: api.Location{File: file, Line: line},
		ThreadID:   threadID,
		Status:     status,
	}
}

func breakpoint(id int, file string, line int, fn string, cond string, disabled bool) *api.Breakpoint {
	return &api.Breakpoint{
		ID:           id,
		File:         file,
		Line:         line,
		FunctionName: fn,
		Cond:         cond,
		Disabled:     disabled,
	}
}

func osthread(id int, file string, line int, goroutineID int64) *api.Thread {
	return &api.Thread{
		ID:          id,
		File:        file,
		Line:        line,
		GoroutineID: goroutineID,
	}
}

// --- selected record ---

func TestFormatState_EmptyInputs(t *testing.T) {
	got := FormatState(nil, nil, nil, -1, 0)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFormatState_SelectedGoroutinePrefix(t *testing.T) {
	got := FormatState(nil, nil, nil, 42, 0)
	if got != "selected;;42;;goroutine||" {
		t.Errorf("expected 'selected;;42;;goroutine||', got %q", got)
	}
}

func TestFormatState_SelectedFallbackToThread(t *testing.T) {
	// No goroutine selected; should fall back to the OS thread.
	got := FormatState(nil, nil, nil, -1, 5)
	if got != "selected;;5;;thread||" {
		t.Errorf("expected 'selected;;5;;thread||', got %q", got)
	}
}

func TestFormatState_SelectedPrefersGoroutine(t *testing.T) {
	// Both goroutine and thread available — goroutine wins.
	got := FormatState(nil, nil, nil, 7, 5)
	if !strings.Contains(got, "selected;;7;;goroutine||") {
		t.Errorf("expected goroutine selection, got %q", got)
	}
	if strings.Contains(got, "selected;;5;;thread||") {
		t.Errorf("unexpected thread selection when goroutine available, got %q", got)
	}
}

func TestFormatState_NoSelectionWhenBothAbsent(t *testing.T) {
	got := FormatState(nil, nil, nil, -1, 0)
	if strings.Contains(got, "selected;;") {
		t.Errorf("expected no selected record, got %q", got)
	}
}

// --- goroutine thread records ---

func TestFormatState_RunningGoroutineIncluded(t *testing.T) {
	// Grunning = 2
	gs := []*api.Goroutine{goroutine(1, "main.go", 10, 5, 2)}
	got := FormatState(gs, nil, nil, -1, 0)
	if !strings.Contains(got, "thread;;1;;goroutine;;main.go;;10;;5||") {
		t.Errorf("expected running goroutine in output, got %q", got)
	}
}

func TestFormatState_WaitingGoroutineExcluded(t *testing.T) {
	// Gwaiting = 4
	gs := []*api.Goroutine{goroutine(2, "runtime/proc.go", 463, 0, 4)}
	got := FormatState(gs, nil, nil, -1, 0)
	if strings.Contains(got, "thread;;") {
		t.Errorf("expected waiting goroutine excluded, got %q", got)
	}
}

func TestFormatState_DeadGoroutineExcluded(t *testing.T) {
	// Gdead = 6
	gs := []*api.Goroutine{goroutine(3, "main.go", 1, 0, 6)}
	got := FormatState(gs, nil, nil, -1, 0)
	if strings.Contains(got, "thread;;") {
		t.Errorf("expected dead goroutine excluded, got %q", got)
	}
}

func TestFormatState_IdleGoroutineExcluded(t *testing.T) {
	// Gidle = 0
	gs := []*api.Goroutine{goroutine(4, "main.go", 1, 0, 0)}
	got := FormatState(gs, nil, nil, -1, 0)
	if strings.Contains(got, "thread;;") {
		t.Errorf("expected idle goroutine excluded, got %q", got)
	}
}

func TestFormatState_SelectedGoroutineAlwaysIncluded(t *testing.T) {
	// Gwaiting but is the selected goroutine — must be included.
	gs := []*api.Goroutine{goroutine(7, "main.go", 20, 0, 4)}
	got := FormatState(gs, nil, nil, 7, 0)
	if !strings.Contains(got, "thread;;7;;goroutine;;main.go;;20;;0||") {
		t.Errorf("expected selected goroutine included despite waiting status, got %q", got)
	}
}

func TestFormatState_UnreadableGoroutineExcluded(t *testing.T) {
	g := goroutine(5, "", 0, 0, 2)
	g.Unreadable = "some error"
	got := FormatState([]*api.Goroutine{g}, nil, nil, -1, 0)
	if strings.Contains(got, "thread;;") {
		t.Errorf("expected unreadable goroutine excluded, got %q", got)
	}
}

func TestFormatState_MaxGoroutinesLimit(t *testing.T) {
	gs := make([]*api.Goroutine, MaxGoroutines+10)
	for i := range gs {
		gs[i] = goroutine(int64(i+1), "main.go", i, 0, 2) // Grunning
	}
	got := FormatState(gs, nil, nil, -1, 0)
	count := strings.Count(got, ";;goroutine;;")
	if count != MaxGoroutines {
		t.Errorf("expected %d goroutines, got %d", MaxGoroutines, count)
	}
}

// --- OS thread records ---

func TestFormatState_OSThreadIncluded(t *testing.T) {
	ths := []*api.Thread{osthread(5, "main.go", 10, 3)}
	got := FormatState(nil, ths, nil, -1, 0)
	if !strings.Contains(got, "thread;;5;;thread;;main.go;;10;;3||") {
		t.Errorf("expected OS thread in output, got %q", got)
	}
}

func TestFormatState_OSThreadWithNoGoroutine(t *testing.T) {
	// GoroutineID = 0 means thread is not running a goroutine.
	ths := []*api.Thread{osthread(3, "runtime/asm.s", 100, 0)}
	got := FormatState(nil, ths, nil, -1, 0)
	if !strings.Contains(got, "thread;;3;;thread;;runtime/asm.s;;100;;0||") {
		t.Errorf("expected OS thread with goroutineID=0 in output, got %q", got)
	}
}

func TestFormatState_GoroutineAndOSThreadCoexist(t *testing.T) {
	gs := []*api.Goroutine{goroutine(1, "main.go", 10, 5, 2)}
	ths := []*api.Thread{osthread(5, "main.go", 10, 1)}
	got := FormatState(gs, ths, nil, -1, 0)
	if !strings.Contains(got, ";;goroutine;;") {
		t.Errorf("expected goroutine record, got %q", got)
	}
	if !strings.Contains(got, ";;thread;;") {
		t.Errorf("expected OS thread record, got %q", got)
	}
}

// --- breakpoint records ---

func TestFormatState_BreakpointFormatted(t *testing.T) {
	bps := []*api.Breakpoint{breakpoint(1, "main.go", 10, "main.main", "", false)}
	got := FormatState(nil, nil, bps, -1, 0)
	if got != "bp;;1;;main.go;;10;;True;;True||" {
		t.Errorf("unexpected breakpoint format: %q", got)
	}
}

func TestFormatState_ConditionalBreakpoint(t *testing.T) {
	bps := []*api.Breakpoint{breakpoint(2, "main.go", 5, "main.foo", "x > 0", false)}
	got := FormatState(nil, nil, bps, -1, 0)
	if !strings.Contains(got, "bp;;2;;main.go;;5;;False;;True||") {
		t.Errorf("unexpected conditional breakpoint format: %q", got)
	}
}

func TestFormatState_DisabledBreakpoint(t *testing.T) {
	bps := []*api.Breakpoint{breakpoint(3, "main.go", 7, "main.bar", "", true)}
	got := FormatState(nil, nil, bps, -1, 0)
	if !strings.Contains(got, "bp;;3;;main.go;;7;;True;;False||") {
		t.Errorf("unexpected disabled breakpoint format: %q", got)
	}
}

func TestFormatState_InternalBreakpointExcluded(t *testing.T) {
	bps := []*api.Breakpoint{breakpoint(-1, "main.go", 1, "main.main", "", false)}
	got := FormatState(nil, nil, bps, -1, 0)
	if strings.Contains(got, "bp;;") {
		t.Errorf("expected internal breakpoint excluded, got %q", got)
	}
}

// --- structural ---

func TestFormatState_RecordTerminator(t *testing.T) {
	gs := []*api.Goroutine{goroutine(1, "main.go", 10, 5, 2)}
	got := FormatState(gs, nil, nil, -1, 0)
	if !strings.HasSuffix(got, "||") {
		t.Errorf("expected output to end with '||', got %q", got)
	}
}

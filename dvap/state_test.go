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

func TestFormatState_EmptyInputs(t *testing.T) {
	got := FormatState(nil, nil, -1)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFormatState_SelectedGoidPrefix(t *testing.T) {
	got := FormatState(nil, nil, 42)
	if got != "selected:42" {
		t.Errorf("expected 'selected:42', got %q", got)
	}
}

func TestFormatState_RunningGoroutineIncluded(t *testing.T) {
	// Grunning = 2
	gs := []*api.Goroutine{goroutine(1, "main.go", 10, 5, 2)}
	got := FormatState(gs, nil, -1)
	if !strings.Contains(got, "thread:1:main.go:10:5") {
		t.Errorf("expected running goroutine in output, got %q", got)
	}
}

func TestFormatState_WaitingGoroutineExcluded(t *testing.T) {
	// Gwaiting = 4
	gs := []*api.Goroutine{goroutine(2, "runtime/proc.go", 463, 0, 4)}
	got := FormatState(gs, nil, -1)
	if strings.Contains(got, "thread:") {
		t.Errorf("expected waiting goroutine excluded, got %q", got)
	}
}

func TestFormatState_DeadGoroutineExcluded(t *testing.T) {
	// Gdead = 6
	gs := []*api.Goroutine{goroutine(3, "main.go", 1, 0, 6)}
	got := FormatState(gs, nil, -1)
	if strings.Contains(got, "thread:") {
		t.Errorf("expected dead goroutine excluded, got %q", got)
	}
}

func TestFormatState_IdleGoroutineExcluded(t *testing.T) {
	// Gidle = 0
	gs := []*api.Goroutine{goroutine(4, "main.go", 1, 0, 0)}
	got := FormatState(gs, nil, -1)
	if strings.Contains(got, "thread:") {
		t.Errorf("expected idle goroutine excluded, got %q", got)
	}
}

func TestFormatState_SelectedGoroutineAlwaysIncluded(t *testing.T) {
	// Gwaiting but is the selected goroutine — must be included
	gs := []*api.Goroutine{goroutine(7, "main.go", 20, 0, 4)}
	got := FormatState(gs, nil, 7)
	if !strings.Contains(got, "thread:7:main.go:20:0") {
		t.Errorf("expected selected goroutine included despite waiting status, got %q", got)
	}
}

func TestFormatState_UnreadableGoroutineExcluded(t *testing.T) {
	g := goroutine(5, "", 0, 0, 2)
	g.Unreadable = "some error"
	got := FormatState([]*api.Goroutine{g}, nil, -1)
	if strings.Contains(got, "thread:") {
		t.Errorf("expected unreadable goroutine excluded, got %q", got)
	}
}

func TestFormatState_BreakpointFormatted(t *testing.T) {
	bps := []*api.Breakpoint{breakpoint(1, "main.go", 10, "main.main", "", false)}
	got := FormatState(nil, bps, -1)
	if got != "bp:1:main.go:10:main.main:true:true" {
		t.Errorf("unexpected breakpoint format: %q", got)
	}
}

func TestFormatState_ConditionalBreakpoint(t *testing.T) {
	bps := []*api.Breakpoint{breakpoint(2, "main.go", 5, "main.foo", "x > 0", false)}
	got := FormatState(nil, bps, -1)
	if !strings.Contains(got, "bp:2:main.go:5:main.foo:false:true") {
		t.Errorf("unexpected conditional breakpoint format: %q", got)
	}
}

func TestFormatState_DisabledBreakpoint(t *testing.T) {
	bps := []*api.Breakpoint{breakpoint(3, "main.go", 7, "main.bar", "", true)}
	got := FormatState(nil, bps, -1)
	if !strings.Contains(got, "bp:3:main.go:7:main.bar:true:false") {
		t.Errorf("unexpected disabled breakpoint format: %q", got)
	}
}

func TestFormatState_InternalBreakpointExcluded(t *testing.T) {
	bps := []*api.Breakpoint{breakpoint(-1, "main.go", 1, "main.main", "", false)}
	got := FormatState(nil, bps, -1)
	if strings.Contains(got, "bp:") {
		t.Errorf("expected internal breakpoint excluded, got %q", got)
	}
}

func TestFormatState_MaxGoroutinesLimit(t *testing.T) {
	gs := make([]*api.Goroutine, MaxGoroutines+10)
	for i := range gs {
		gs[i] = goroutine(int64(i+1), "main.go", i, 0, 2) // Grunning
	}
	got := FormatState(gs, nil, -1)
	count := strings.Count(got, "thread:")
	if count != MaxGoroutines {
		t.Errorf("expected %d goroutines, got %d", MaxGoroutines, count)
	}
}

func TestFormatState_NoTrailingSpace(t *testing.T) {
	gs := []*api.Goroutine{goroutine(1, "main.go", 10, 5, 2)}
	got := FormatState(gs, nil, -1)
	if strings.HasSuffix(got, " ") {
		t.Errorf("unexpected trailing space in %q", got)
	}
}

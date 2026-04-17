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
	"bufio"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-delve/delve/service/api"
)

// mockClient is a minimal stub that satisfies service.Client for testing.
// Only the methods exercised by DVAPClient are implemented.
type mockClient struct {
	goroutines  []*api.Goroutine
	breakpoints []*api.Breakpoint
	state       *api.DebuggerState
	running     bool

	nextState *api.DebuggerState
	nextErr   error

	stepState *api.DebuggerState
	stepErr   error

	bpResult *api.Breakpoint
	bpErr    error
}

func (m *mockClient) ListGoroutines(start, count int) ([]*api.Goroutine, int, error) {
	return m.goroutines, len(m.goroutines), nil
}
func (m *mockClient) ListBreakpoints(all bool) ([]*api.Breakpoint, error) {
	return m.breakpoints, nil
}
func (m *mockClient) GetStateNonBlocking() (*api.DebuggerState, error) {
	return m.state, nil
}
func (m *mockClient) Next() (*api.DebuggerState, error) { return m.nextState, m.nextErr }
func (m *mockClient) Step() (*api.DebuggerState, error) { return m.stepState, m.stepErr }
func (m *mockClient) CreateBreakpoint(bp *api.Breakpoint) (*api.Breakpoint, error) {
	return m.bpResult, m.bpErr
}

// Remaining service.Client methods — stubs that panic if called unexpectedly.
func (m *mockClient) GetVersion() *api.GetVersionOut                   { panic("not implemented") }
func (m *mockClient) ProcessPid() int                                  { return 0 }
func (m *mockClient) BuildID() string                                  { return "" }
func (m *mockClient) LastModified() time.Time                          { return time.Time{} }
func (m *mockClient) Detach(bool) error                                { return nil }
func (m *mockClient) Restart(bool) ([]api.DiscardedBreakpoint, error)  { return nil, nil }
func (m *mockClient) RestartFrom(bool, string, bool, []string, [3]string, bool) ([]api.DiscardedBreakpoint, error) {
	return nil, nil
}
func (m *mockClient) GetState() (*api.DebuggerState, error)            { return m.state, nil }
func (m *mockClient) Continue() <-chan *api.DebuggerState              { panic("not implemented") }
func (m *mockClient) Rewind() <-chan *api.DebuggerState                { panic("not implemented") }
func (m *mockClient) DirectionCongruentContinue() <-chan *api.DebuggerState {
	panic("not implemented")
}
func (m *mockClient) ReverseNext() (*api.DebuggerState, error)        { return nil, nil }
func (m *mockClient) ReverseStep() (*api.DebuggerState, error)        { return nil, nil }
func (m *mockClient) StepOut() (*api.DebuggerState, error)            { return nil, nil }
func (m *mockClient) ReverseStepOut() (*api.DebuggerState, error)     { return nil, nil }
func (m *mockClient) Call(int64, string, bool) (*api.DebuggerState, error) { return nil, nil }
func (m *mockClient) StepInstruction(bool) (*api.DebuggerState, error)     { return nil, nil }
func (m *mockClient) ReverseStepInstruction(bool) (*api.DebuggerState, error) { return nil, nil }
func (m *mockClient) SwitchThread(int) (*api.DebuggerState, error)    { return nil, nil }
func (m *mockClient) SwitchGoroutine(int64) (*api.DebuggerState, error) { return nil, nil }
func (m *mockClient) Halt() (*api.DebuggerState, error)               { return nil, nil }
func (m *mockClient) GetBreakpoint(int) (*api.Breakpoint, error)      { return nil, nil }
func (m *mockClient) GetBreakpointByName(string) (*api.Breakpoint, error) { return nil, nil }
func (m *mockClient) CreateBreakpointWithExpr(*api.Breakpoint, string, [][2]string, bool) (*api.Breakpoint, error) {
	return m.bpResult, m.bpErr
}
func (m *mockClient) CreateWatchpoint(api.EvalScope, string, api.WatchType) (*api.Breakpoint, error) {
	return m.bpResult, m.bpErr
}
func (m *mockClient) ClearBreakpoint(int) (*api.Breakpoint, error)        { return nil, nil }
func (m *mockClient) ClearBreakpointByName(string) (*api.Breakpoint, error) { return nil, nil }
func (m *mockClient) ToggleBreakpoint(int) (*api.Breakpoint, error)       { return nil, nil }
func (m *mockClient) ToggleBreakpointByName(string) (*api.Breakpoint, error) { return nil, nil }
func (m *mockClient) AmendBreakpoint(*api.Breakpoint) error               { return nil }
func (m *mockClient) CancelNext() error                                    { return nil }
func (m *mockClient) ListThreads() ([]*api.Thread, error)                  { return nil, nil }
func (m *mockClient) GetThread(int) (*api.Thread, error)                   { return nil, nil }
func (m *mockClient) ListPackageVariables(string, api.LoadConfig) ([]api.Variable, error) {
	return nil, nil
}
func (m *mockClient) EvalVariable(api.EvalScope, string, api.LoadConfig) (*api.Variable, error) {
	return nil, nil
}
func (m *mockClient) SetVariable(api.EvalScope, string, string) error { return nil }
func (m *mockClient) ListSources(string) ([]string, error)             { return nil, nil }
func (m *mockClient) ListFunctions(string, int) ([]string, error)      { return nil, nil }
func (m *mockClient) ListTypes(string) ([]string, error)               { return nil, nil }
func (m *mockClient) ListPackagesBuildInfo(string, bool) ([]api.PackageBuildInfo, error) {
	return nil, nil
}
func (m *mockClient) ListLocalVariables(api.EvalScope, api.LoadConfig) ([]api.Variable, error) {
	return nil, nil
}
func (m *mockClient) ListFunctionArgs(api.EvalScope, api.LoadConfig) ([]api.Variable, error) {
	return nil, nil
}
func (m *mockClient) ListThreadRegisters(int, bool) (api.Registers, error)    { return nil, nil }
func (m *mockClient) ListScopeRegisters(api.EvalScope, bool) (api.Registers, error) {
	return nil, nil
}
func (m *mockClient) ListGoroutinesWithFilter(int, int, []api.ListGoroutinesFilter, *api.GoroutineGroupingOptions, *api.EvalScope) ([]*api.Goroutine, []api.GoroutineGroup, int, bool, error) {
	return nil, nil, 0, false, nil
}
func (m *mockClient) Stacktrace(int64, int, api.StacktraceOptions, *api.LoadConfig) ([]api.Stackframe, error) {
	return nil, nil
}
func (m *mockClient) Ancestors(int64, int, int) ([]api.Ancestor, error) { return nil, nil }
func (m *mockClient) AttachedToExistingProcess() bool                   { return false }
func (m *mockClient) FindLocation(api.EvalScope, string, bool, [][2]string) ([]api.Location, string, error) {
	return nil, "", nil
}
func (m *mockClient) DisassembleRange(api.EvalScope, uint64, uint64, api.AssemblyFlavour) (api.AsmInstructions, error) {
	return nil, nil
}
func (m *mockClient) DisassemblePC(api.EvalScope, uint64, api.AssemblyFlavour) (api.AsmInstructions, error) {
	return nil, nil
}
func (m *mockClient) Recorded() bool                             { return false }
func (m *mockClient) TraceDirectory() (string, error)            { return "", nil }
func (m *mockClient) Checkpoint(string) (int, error)             { return 0, nil }
func (m *mockClient) ListCheckpoints() ([]api.Checkpoint, error) { return nil, nil }
func (m *mockClient) ClearCheckpoint(int) error                  { return nil }
func (m *mockClient) SetReturnValuesLoadConfig(*api.LoadConfig)  {}
func (m *mockClient) SetEventsFn(func(*api.Event))               {}
func (m *mockClient) IsMulticlient() bool                        { return false }
func (m *mockClient) ListDynamicLibraries() ([]api.Image, bool, error) { return nil, false, nil }
func (m *mockClient) ExamineMemory(uint64, int) ([]byte, bool, error)  { return nil, false, nil }
func (m *mockClient) StopRecording() error                             { return nil }
func (m *mockClient) CoreDumpStart(string) (api.DumpState, error)      { return api.DumpState{}, nil }
func (m *mockClient) CoreDumpWait(int) api.DumpState                   { return api.DumpState{} }
func (m *mockClient) CoreDumpCancel() error                            { return nil }
func (m *mockClient) ListTargets() ([]api.Target, error)               { return nil, nil }
func (m *mockClient) FollowExec(bool, string) error                    { return nil }
func (m *mockClient) FollowExecEnabled() bool                          { return false }
func (m *mockClient) Disconnect(bool) error                            { return nil }
func (m *mockClient) SetDebugInfoDirectories([]string) error           { return nil }
func (m *mockClient) GetDebugInfoDirectories() ([]string, error)       { return nil, nil }
func (m *mockClient) GuessSubstitutePath() ([][2]string, error)        { return nil, nil }
func (m *mockClient) CancelDownloads() error                           { return nil }
func (m *mockClient) DownloadLibraryDebugInfo(int) error               { return nil }
func (m *mockClient) CallAPI(string, any, any) error                   { return nil }

// --- tests ---

func readEvent(t *testing.T, resp *http.Response, timeout time.Duration) string {
	t.Helper()
	done := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "data: ") {
				done <- strings.TrimPrefix(line, "data: ")
				return
			}
		}
		done <- ""
	}()
	select {
	case msg := <-done:
		return msg
	case <-time.After(timeout):
		t.Fatal("timeout waiting for SSE event")
		return ""
	}
}

func TestDVAPClient_Next_Broadcasts(t *testing.T) {
	srv := New()
	if err := srv.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	inner := &mockClient{
		nextState: &api.DebuggerState{
			SelectedGoroutine: &api.Goroutine{ID: 3},
		},
		goroutines: []*api.Goroutine{
			{ID: 3, CurrentLoc: api.Location{File: "main.go", Line: 10}, ThreadID: 1, Status: 2},
		},
		state: &api.DebuggerState{Running: false, SelectedGoroutine: &api.Goroutine{ID: 3}},
	}
	client := NewClient(inner, srv)

	resp, err := http.Get("http://" + srv.Addr() + "/events")
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	time.Sleep(10 * time.Millisecond)

	st, err := client.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if st.SelectedGoroutine.ID != 3 {
		t.Errorf("unexpected goroutine ID: %d", st.SelectedGoroutine.ID)
	}

	got := readEvent(t, resp, time.Second)
	if !strings.Contains(got, "selected;;3;;goroutine||") {
		t.Errorf("expected selected;;3;;goroutine in broadcast, got %q", got)
	}
	if !strings.Contains(got, "thread;;3;;goroutine;;main.go;;10;;1||") {
		t.Errorf("expected goroutine token in broadcast, got %q", got)
	}
}

func TestDVAPClient_CreateBreakpoint_Broadcasts(t *testing.T) {
	srv := New()
	if err := srv.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	inner := &mockClient{
		bpResult: &api.Breakpoint{ID: 1, File: "main.go", Line: 5, FunctionName: "main.main"},
		breakpoints: []*api.Breakpoint{
			{ID: 1, File: "main.go", Line: 5, FunctionName: "main.main"},
		},
		state: &api.DebuggerState{Running: false},
	}
	client := NewClient(inner, srv)

	resp, err := http.Get("http://" + srv.Addr() + "/events")
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	time.Sleep(10 * time.Millisecond)

	_, err = client.CreateBreakpoint(&api.Breakpoint{File: "main.go", Line: 5})
	if err != nil {
		t.Fatalf("CreateBreakpoint: %v", err)
	}

	got := readEvent(t, resp, time.Second)
	if !strings.Contains(got, "bp;;1;;main.go;;5;;") {
		t.Errorf("expected breakpoint token in broadcast, got %q", got)
	}
}

func TestDVAPClient_BreakpointBroadcast_SkippedWhenRunning(t *testing.T) {
	srv := New()
	if err := srv.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	inner := &mockClient{
		bpResult: &api.Breakpoint{ID: 1},
		state:    &api.DebuggerState{Running: true},
	}
	client := NewClient(inner, srv)

	// Set a known last event so we can verify no new event is sent.
	srv.Broadcast("sentinel")

	resp, err := http.Get("http://" + srv.Addr() + "/events")
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	// Consume the replayed sentinel.
	got := readEvent(t, resp, time.Second)
	if got != "sentinel" {
		t.Fatalf("expected sentinel, got %q", got)
	}

	_, _ = client.CreateBreakpoint(&api.Breakpoint{})

	// No further event should arrive within 100ms.
	select {
	case <-time.After(100 * time.Millisecond):
		// expected: no broadcast when process is running
	}
}

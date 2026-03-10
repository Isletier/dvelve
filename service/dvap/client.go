package dvap

import (
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/api"
)

// Client wraps a service.Client and broadcasts the current DVAP state after
// each operation that changes goroutine positions or breakpoints.
type Client struct {
	service.Client
	srv *Server
}

// NewClient wraps inner with DVAP broadcasting. srv must already be started.
func NewClient(inner service.Client, srv *Server) *Client {
	return &Client{Client: inner, srv: srv}
}

// goroutineFetchLimit is the maximum number of goroutines fetched per broadcast.
// Programs with more goroutines than this are paginated: the first
// goroutineFetchLimit goroutines are fetched, and the selected goroutine is
// always appended if it falls outside that window. This bounds RPC payload
// size while ensuring the focused goroutine is never silently dropped.
// FormatState then applies its own status-based filter and MaxGoroutines cap
// on the client side, so the broadcast size is always predictable.
const goroutineFetchLimit = MaxGoroutines * 8 // 4096

// broadcastState fetches current goroutines and breakpoints then broadcasts.
// st carries the DebuggerState returned by a just-completed operation; when
// nil the state is fetched with GetStateNonBlocking and the broadcast is
// skipped if the process is still running.
func (c *Client) broadcastState(st *api.DebuggerState) {
	if st == nil {
		var err error
		st, err = c.Client.GetStateNonBlocking()
		if err != nil || st.Running {
			return
		}
	}
	selectedGoid := int64(-1)
	if st.SelectedGoroutine != nil {
		selectedGoid = st.SelectedGoroutine.ID
	}
	goroutines, _, _ := c.Client.ListGoroutines(0, goroutineFetchLimit)
	// Ensure the selected goroutine is always present even when it falls
	// outside the fetch window (e.g. it is waiting/parked and goroutineFetchLimit
	// was reached before reaching it in the server's ordering).
	if st.SelectedGoroutine != nil {
		found := false
		for _, g := range goroutines {
			if g.ID == selectedGoid {
				found = true
				break
			}
		}
		if !found {
			goroutines = append(goroutines, st.SelectedGoroutine)
		}
	}
	breakpoints, _ := c.Client.ListBreakpoints(false)
	c.srv.Broadcast(FormatState(goroutines, breakpoints, selectedGoid))
}

// wrapContinueChan forwards all values from inner, broadcasting after each
// final (non-running) state arrives.
func (c *Client) wrapContinueChan(inner <-chan *api.DebuggerState) <-chan *api.DebuggerState {
	out := make(chan *api.DebuggerState, 1)
	go func() {
		defer close(out)
		for st := range inner {
			out <- st
			if st != nil && !st.Running {
				c.broadcastState(st)
			}
		}
	}()
	return out
}

// --- async execution methods ---

func (c *Client) Continue() <-chan *api.DebuggerState {
	return c.wrapContinueChan(c.Client.Continue())
}

func (c *Client) Rewind() <-chan *api.DebuggerState {
	return c.wrapContinueChan(c.Client.Rewind())
}

func (c *Client) DirectionCongruentContinue() <-chan *api.DebuggerState {
	return c.wrapContinueChan(c.Client.DirectionCongruentContinue())
}

// --- sync execution methods ---

func (c *Client) Next() (*api.DebuggerState, error) {
	st, err := c.Client.Next()
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) ReverseNext() (*api.DebuggerState, error) {
	st, err := c.Client.ReverseNext()
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) Step() (*api.DebuggerState, error) {
	st, err := c.Client.Step()
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) ReverseStep() (*api.DebuggerState, error) {
	st, err := c.Client.ReverseStep()
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) StepOut() (*api.DebuggerState, error) {
	st, err := c.Client.StepOut()
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) ReverseStepOut() (*api.DebuggerState, error) {
	st, err := c.Client.ReverseStepOut()
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) Call(goroutineID int64, expr string, unsafe bool) (*api.DebuggerState, error) {
	st, err := c.Client.Call(goroutineID, expr, unsafe)
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) StepInstruction(skipCalls bool) (*api.DebuggerState, error) {
	st, err := c.Client.StepInstruction(skipCalls)
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) ReverseStepInstruction(skipCalls bool) (*api.DebuggerState, error) {
	st, err := c.Client.ReverseStepInstruction(skipCalls)
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) SwitchThread(threadID int) (*api.DebuggerState, error) {
	st, err := c.Client.SwitchThread(threadID)
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) SwitchGoroutine(goroutineID int64) (*api.DebuggerState, error) {
	st, err := c.Client.SwitchGoroutine(goroutineID)
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

func (c *Client) Halt() (*api.DebuggerState, error) {
	st, err := c.Client.Halt()
	if err == nil {
		c.broadcastState(st)
	}
	return st, err
}

// --- breakpoint mutation methods ---

func (c *Client) CreateBreakpoint(bp *api.Breakpoint) (*api.Breakpoint, error) {
	result, err := c.Client.CreateBreakpoint(bp)
	if err == nil {
		c.broadcastState(nil)
	}
	return result, err
}

func (c *Client) CreateBreakpointWithExpr(bp *api.Breakpoint, expr string, substitutePathRules [][2]string, suspended bool) (*api.Breakpoint, error) {
	result, err := c.Client.CreateBreakpointWithExpr(bp, expr, substitutePathRules, suspended)
	if err == nil {
		c.broadcastState(nil)
	}
	return result, err
}

func (c *Client) CreateWatchpoint(scope api.EvalScope, expr string, wtype api.WatchType) (*api.Breakpoint, error) {
	result, err := c.Client.CreateWatchpoint(scope, expr, wtype)
	if err == nil {
		c.broadcastState(nil)
	}
	return result, err
}

func (c *Client) ClearBreakpoint(id int) (*api.Breakpoint, error) {
	result, err := c.Client.ClearBreakpoint(id)
	if err == nil {
		c.broadcastState(nil)
	}
	return result, err
}

func (c *Client) ClearBreakpointByName(name string) (*api.Breakpoint, error) {
	result, err := c.Client.ClearBreakpointByName(name)
	if err == nil {
		c.broadcastState(nil)
	}
	return result, err
}

func (c *Client) ToggleBreakpoint(id int) (*api.Breakpoint, error) {
	result, err := c.Client.ToggleBreakpoint(id)
	if err == nil {
		c.broadcastState(nil)
	}
	return result, err
}

func (c *Client) ToggleBreakpointByName(name string) (*api.Breakpoint, error) {
	result, err := c.Client.ToggleBreakpointByName(name)
	if err == nil {
		c.broadcastState(nil)
	}
	return result, err
}

func (c *Client) AmendBreakpoint(bp *api.Breakpoint) error {
	err := c.Client.AmendBreakpoint(bp)
	if err == nil {
		c.broadcastState(nil)
	}
	return err
}

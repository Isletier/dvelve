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
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServer_StartStop(t *testing.T) {
	s := New()
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if s.Addr() == "" {
		t.Fatal("Addr() empty after Start")
	}
	s.Stop()
}

func TestServer_BroadcastReceived(t *testing.T) {
	s := New()
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	url := "http://" + s.Addr() + "/events"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	// Give the server time to register the client before broadcasting.
	time.Sleep(10 * time.Millisecond)

	s.Broadcast("hello world")

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			got := strings.TrimPrefix(line, "data: ")
			if got != "hello world" {
				t.Errorf("unexpected data: %q", got)
			}
			return
		}
	}
	t.Error("did not receive broadcast")
}

func TestServer_LastEventReplayedToNewClient(t *testing.T) {
	s := New()
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	// Broadcast before any client connects.
	s.Broadcast("initial state")

	url := "http://" + s.Addr() + "/events"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	// The client should immediately receive the cached last event.
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			got := strings.TrimPrefix(line, "data: ")
			if got != "initial state" {
				t.Errorf("unexpected replay data: %q", got)
			}
			return
		}
	}
	t.Error("did not receive replayed last event")
}

func TestServer_StopClosesConnections(t *testing.T) {
	s := New()
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	url := "http://" + s.Addr() + "/events"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	time.Sleep(10 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
	}()

	s.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("Stop() did not close SSE connection within 2s")
	}
}

func TestServer_LocalhostOnly(t *testing.T) {
	s := New()
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	// Simulate a request with a non-loopback Host header.
	port := s.Addr()[strings.LastIndex(s.Addr(), ":"):]
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1%s/events", port), nil)
	req.Host = "evil.example.com"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

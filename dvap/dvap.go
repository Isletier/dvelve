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

Package dvap implements a Debug View Adapter Protocol server.
It serves the current debugger state (goroutines and breakpoints)
as a plain-text SSE stream at GET /events, broadcasting on every
state-changing debugger operation.
*/

package dvap

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
)

// dispatcher manages the set of connected SSE client queues.
type dispatcher struct {
	mu      sync.Mutex
	clients []chan []byte
	last    []byte // last broadcast message, replayed to new subscribers
}

func (d *dispatcher) subscribe() chan []byte {
	ch := make(chan []byte, 100)
	d.mu.Lock()
	d.clients = append(d.clients, ch)
	if d.last != nil {
		ch <- d.last
	}
	d.mu.Unlock()
	return ch
}

func (d *dispatcher) unsubscribe(ch chan []byte) {
	d.mu.Lock()
	for i, c := range d.clients {
		if c == ch {
			d.clients = append(d.clients[:i], d.clients[i+1:]...)
			break
		}
	}
	d.mu.Unlock()
}

// closeAll closes every client channel, causing all handleSSE goroutines to
// return so that http.Server.Shutdown has no active handlers left to wait on.
func (d *dispatcher) closeAll() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, ch := range d.clients {
		close(ch)
	}
	d.clients = nil
}

// broadcast sends data to every connected client and caches it for replay.
// Clients whose queue is full are silently dropped.
func (d *dispatcher) broadcast(data string) {
	msg := []byte("data: " + data + "\n\n")
	d.mu.Lock()
	defer d.mu.Unlock()
	d.last = msg
	for i := len(d.clients) - 1; i >= 0; i-- {
		select {
		case d.clients[i] <- msg:
		default:
			close(d.clients[i])
			d.clients = append(d.clients[:i], d.clients[i+1:]...)
		}
	}
}

// Server is the DVAP SSE HTTP server. Its lifetime must be managed by the
// caller: Start before the terminal client runs, Stop when the debug session
// ends.
type Server struct {
	d        *dispatcher
	http     *http.Server
	listener net.Listener
}

// New creates a Server. Call Start to begin serving.
func New() *Server {
	s := &Server{d: &dispatcher{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.handleSSE)
	s.http = &http.Server{Handler: mux}
	return s
}

// Start binds to addr and begins serving SSE in a background goroutine.
func (s *Server) Start(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = ln
	go s.http.Serve(ln) //nolint:errcheck
	return nil
}

// Addr returns the address the server is listening on, or "" if not started.
func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Stop closes all open SSE connections and shuts down the HTTP server.
// Blocks until shutdown completes.
func (s *Server) Stop() {
	// Close all client channels first so every handleSSE goroutine returns
	// immediately. Without this, http.Server.Shutdown would block waiting
	// for the long-lived SSE connections to drain on their own.
	s.d.closeAll()
	s.http.Shutdown(context.Background()) //nolint:errcheck
}

// Broadcast sends a pre-formatted DVAP state string to all connected clients.
func (s *Server) Broadcast(data string) {
	s.d.broadcast(data)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Restrict to loopback connections.
	host := r.Host
	if !strings.HasPrefix(host, "127.0.0.1") && !strings.HasPrefix(host, "localhost") {
		http.Error(w, "Access Denied: loopback only", http.StatusForbidden)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher.Flush()

	ch := s.d.subscribe()
	defer s.d.unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.Write(msg) //nolint:errcheck
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

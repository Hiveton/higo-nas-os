// Package relay is the cloud data plane. Each NAS agent dials in and holds one
// persistent WebSocket; the cloud multiplexes App HTTP requests onto it and reads
// the responses back, so an App off the LAN reaches the NAS even though the NAS
// has no public address.
//
// Framing is JSON request/response envelopes correlated by id. Bodies are
// buffered (base64) — enough for the JSON control API. Streaming endpoints (SSE,
// the AI message stream, the Docker PTY WebSocket) are out of scope for this
// first cut and are documented as a follow-up.
package relay

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ErrOffline is returned when no agent is connected for a device.
var ErrOffline = errors.New("device offline")

// ErrTimeout is returned when the NAS does not answer in time.
var ErrTimeout = errors.New("relay request timed out")

const (
	requestTimeout = 30 * time.Second
	writeWait      = 10 * time.Second
	pongWait       = 90 * time.Second
	pingPeriod     = 60 * time.Second
)

// frame is the wire envelope in both directions.
type frame struct {
	Type    string            `json:"type"` // request | response | ping
	ID      string            `json:"id,omitempty"`
	Method  string            `json:"method,omitempty"`
	Path    string            `json:"path,omitempty"`
	Status  int               `json:"status,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"` // base64
}

// Response is the materialized NAS reply.
type Response struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

// agentConn wraps a single NAS WebSocket and its in-flight requests.
type agentConn struct {
	ws       *websocket.Conn
	writeMu  sync.Mutex
	mu       sync.Mutex
	pending  map[string]chan frame
	closed   bool
}

// Hub is the device->connection registry.
type Hub struct {
	mu     sync.RWMutex
	agents map[string]*agentConn
}

// NewHub returns an empty hub.
func NewHub() *Hub { return &Hub{agents: map[string]*agentConn{}} }

// Online reports whether a device currently holds a relay connection.
func (h *Hub) Online(deviceID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.agents[deviceID]
	return ok
}

// Serve takes an upgraded WebSocket for an authenticated device and runs its read
// loop until the connection drops. It blocks, so callers run it from the agent
// HTTP handler goroutine.
func (h *Hub) Serve(deviceID string, ws *websocket.Conn) {
	conn := &agentConn{ws: ws, pending: map[string]chan frame{}}
	h.mu.Lock()
	if old := h.agents[deviceID]; old != nil {
		old.close()
	}
	h.agents[deviceID] = conn
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		if h.agents[deviceID] == conn {
			delete(h.agents, deviceID)
		}
		h.mu.Unlock()
		conn.close()
	}()

	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go conn.pingLoop()

	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			return
		}
		var f frame
		if json.Unmarshal(data, &f) != nil {
			continue
		}
		switch f.Type {
		case "response", "response-head", "response-chunk", "response-end":
			conn.deliver(f)
		}
	}
}

// Forward sends an HTTP request to a device over its relay connection and streams
// the NAS response straight to w. It is the engine behind /d/{deviceID}/...
//
// Responses arrive as either a single buffered "response" frame or a streamed
// "response-head" → "response-chunk"* → "response-end" sequence; the latter lets
// SSE / chunked endpoints (events stream, AI message stream) flow through the
// tunnel without buffering. The first-frame deadline bounds a hung NAS; once
// streaming has begun the request context governs the lifetime.
func (h *Hub) Forward(ctx context.Context, deviceID string, r *http.Request, upstreamPath string, w http.ResponseWriter) error {
	h.mu.RLock()
	conn := h.agents[deviceID]
	h.mu.RUnlock()
	if conn == nil {
		return ErrOffline
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 16<<20))
	if err != nil {
		return err
	}
	id := randID()
	req := frame{
		Type:    "request",
		ID:      id,
		Method:  r.Method,
		Path:    upstreamPath,
		Headers: flattenHeaders(r.Header),
		Body:    base64.StdEncoding.EncodeToString(body),
	}

	ch := make(chan frame, 64)
	conn.register(id, ch)
	defer conn.unregister(id)

	if err := conn.write(req); err != nil {
		return err
	}

	flusher, _ := w.(http.Flusher)
	headWritten := false
	writeHead := func(status int, headers map[string]string) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		headWritten = true
	}

	first := time.NewTimer(requestTimeout)
	defer first.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-first.C:
			if !headWritten {
				return ErrTimeout
			}
		case f := <-ch:
			first.Stop()
			switch f.Type {
			case "response": // buffered
				decoded, _ := base64.StdEncoding.DecodeString(f.Body)
				writeHead(f.Status, f.Headers)
				_, _ = w.Write(decoded)
				return nil
			case "response-head":
				writeHead(f.Status, f.Headers)
				if flusher != nil {
					flusher.Flush()
				}
			case "response-chunk":
				if !headWritten {
					writeHead(http.StatusOK, nil)
				}
				decoded, _ := base64.StdEncoding.DecodeString(f.Body)
				_, _ = w.Write(decoded)
				if flusher != nil {
					flusher.Flush()
				}
			case "response-end":
				return nil
			}
		}
	}
}

// Call forwards a body-defined request (no *http.Request needed) to a device. It
// backs cloud-internal calls such as account provisioning over the tunnel.
func (h *Hub) Call(ctx context.Context, deviceID, method, path string, headers map[string]string, body []byte) (Response, error) {
	h.mu.RLock()
	conn := h.agents[deviceID]
	h.mu.RUnlock()
	if conn == nil {
		return Response{}, ErrOffline
	}
	id := randID()
	if headers == nil {
		headers = map[string]string{}
	}
	if body != nil {
		headers["Content-Type"] = "application/json"
	}
	req := frame{
		Type:    "request",
		ID:      id,
		Method:  method,
		Path:    path,
		Headers: headers,
		Body:    base64.StdEncoding.EncodeToString(body),
	}
	ch := make(chan frame, 64)
	conn.register(id, ch)
	defer conn.unregister(id)
	if err := conn.write(req); err != nil {
		return Response{}, err
	}
	timeout := time.NewTimer(requestTimeout)
	defer timeout.Stop()
	var acc Response
	for {
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		case <-timeout.C:
			return Response{}, ErrTimeout
		case f := <-ch:
			switch f.Type {
			case "response":
				decoded, _ := base64.StdEncoding.DecodeString(f.Body)
				return Response{Status: f.Status, Headers: f.Headers, Body: decoded}, nil
			case "response-head":
				acc.Status = f.Status
				acc.Headers = f.Headers
			case "response-chunk":
				decoded, _ := base64.StdEncoding.DecodeString(f.Body)
				acc.Body = append(acc.Body, decoded...)
			case "response-end":
				return acc, nil
			}
		}
	}
}

// --- agentConn helpers -------------------------------------------------------

func (c *agentConn) register(id string, ch chan frame) {
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
}

func (c *agentConn) unregister(id string) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *agentConn) deliver(f frame) {
	c.mu.Lock()
	ch := c.pending[f.ID]
	c.mu.Unlock()
	// Keep the registration alive across a multi-frame stream; the requester
	// unregisters when it sees the terminal frame (response / response-end).
	if ch != nil {
		ch <- f
	}
}

func (c *agentConn) write(f frame) error {
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(websocket.TextMessage, data)
}

func (c *agentConn) pingLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for range ticker.C {
		c.writeMu.Lock()
		c.ws.SetWriteDeadline(time.Now().Add(writeWait))
		err := c.ws.WriteMessage(websocket.PingMessage, nil)
		c.writeMu.Unlock()
		if err != nil {
			return
		}
	}
}

func (c *agentConn) close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()
	_ = c.ws.Close()
}

func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k := range h {
		// Hop-by-hop headers must not be forwarded across the tunnel.
		switch k {
		case "Connection", "Upgrade", "Keep-Alive", "Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding":
			continue
		}
		out[k] = h.Get(k)
	}
	return out
}

func randID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}

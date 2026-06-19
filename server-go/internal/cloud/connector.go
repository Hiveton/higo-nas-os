// Package cloud is the NAS-side agent for the HiGoOS cloud control plane
// (server-cloud). It registers this device with the cloud, persists the issued
// secret and the cloud's public key, then holds a single outbound WebSocket open
// so the cloud can forward App requests to the local API even when the NAS sits
// behind NAT.
//
// It is the counterpart to server-cloud/internal/relay: the wire framing
// (JSON request/response envelopes correlated by id, base64 bodies) matches that
// package exactly. Enable it with HIGO_CLOUD_ENABLED=true and HIGO_CLOUD_BASE.
package cloud

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"higoos/server-go/internal/state"
)

// Persisted is the durable cloud binding state for this device.
type Persisted struct {
	DeviceID       string `json:"deviceId"`
	Serial         string `json:"serial"`
	DeviceSecret   string `json:"deviceSecret"`
	CloudPublicKey string `json:"cloudPublicKey"`
	RelayPath      string `json:"relayPath"`
}

// Connector manages registration + the persistent relay tunnel.
type Connector struct {
	baseURL   string
	deviceID  string
	model     string
	version   string
	statePath string
	handler   http.Handler // the local API router; forwarded requests dispatch here
	logger    *slog.Logger

	state Persisted
}

// NewConnector builds a connector. handler is the local API mux that forwarded
// requests are dispatched into.
func NewConnector(baseURL, deviceID, model, version, stateDir string, handler http.Handler, logger *slog.Logger) *Connector {
	c := &Connector{
		baseURL:  strings.TrimRight(baseURL, "/"),
		deviceID: deviceID,
		model:    model,
		version:  version,
		handler:  handler,
		logger:   logger,
	}
	if stateDir != "" {
		c.statePath = filepath.Join(stateDir, "cloud.json")
	}
	return c
}

// CloudPublicKey returns the Ed25519 public key (hex) the cloud uses to sign
// device access tokens. The session guard verifies App-presented tokens with it.
// Empty until the device has registered.
func (c *Connector) CloudPublicKey() string { return c.state.CloudPublicKey }

// Run registers if needed, then maintains the relay connection with backoff until
// ctx is cancelled. Intended to be launched in a goroutine at boot.
func (c *Connector) Run(ctx context.Context) {
	_ = c.load()
	if c.state.DeviceSecret == "" {
		if err := c.register(ctx); err != nil {
			c.logger.Error("cloud register failed", slog.Any("error", err))
			// Retry registration on the reconnect loop below.
		}
	}

	backoff := time.Second
	for ctx.Err() == nil {
		if c.state.DeviceSecret == "" {
			if err := c.register(ctx); err != nil {
				sleep(ctx, backoff)
				backoff = nextBackoff(backoff)
				continue
			}
		}
		if err := c.serve(ctx); err != nil && ctx.Err() == nil {
			c.logger.Warn("cloud relay disconnected", slog.Any("error", err))
		}
		sleep(ctx, backoff)
		backoff = nextBackoff(backoff)
	}
}

func (c *Connector) register(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{
		"deviceId": c.deviceID,
		"model":    c.model,
		"version":  c.version,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/devices/register", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var env struct {
		OK   bool      `json:"ok"`
		Data Persisted `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return err
	}
	c.state = env.Data
	c.state.DeviceID = c.deviceID
	c.logger.Info("registered with cloud", slog.String("serial", c.state.Serial))
	return c.save()
}

func (c *Connector) serve(ctx context.Context) error {
	wsURL, err := relayURL(c.baseURL, c.state.RelayPath)
	if err != nil {
		return err
	}
	header := http.Header{
		"X-Device-Id":     {c.deviceID},
		"X-Device-Secret": {c.state.DeviceSecret},
	}
	ws, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return err
	}
	defer ws.Close()
	c.logger.Info("cloud relay connected", slog.String("url", wsURL))

	wc := &wsConn{ws: ws}
	go func() {
		<-ctx.Done()
		_ = ws.Close()
	}()

	for {
		var f frame
		if err := ws.ReadJSON(&f); err != nil {
			return err
		}
		if f.Type != "request" {
			continue
		}
		go c.dispatch(wc, f)
	}
}

// dispatch replays a forwarded request into the local API and streams the response
// back over the tunnel as response-head → response-chunk* → response-end frames,
// so SSE / chunked endpoints flow without buffering.
func (c *Connector) dispatch(wc *wsConn, f frame) {
	bodyBytes, _ := base64.StdEncoding.DecodeString(f.Body)
	req := httptest.NewRequest(f.Method, "http://nas.local"+f.Path, bytes.NewReader(bodyBytes))
	for k, v := range f.Headers {
		req.Header.Set(k, v)
	}
	// Mark the request as relay-originated so cloud-internal endpoints trust it;
	// requests arriving on the LAN never carry this flag.
	req = req.WithContext(WithRelayOrigin(req.Context()))

	sw := &streamWriter{conn: wc, id: f.ID, header: make(http.Header)}
	c.handler.ServeHTTP(sw, req)
	sw.finish()
}

// wsConn serializes writes from concurrent dispatch goroutines onto one socket.
type wsConn struct {
	ws *websocket.Conn
	mu sync.Mutex
}

func (c *wsConn) writeFrame(f frame) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.ws.WriteJSON(f)
}

// streamWriter is the http.ResponseWriter the local API writes into; each write is
// forwarded as a frame so a Flush on the NAS side reaches the App promptly.
type streamWriter struct {
	conn     *wsConn
	id       string
	header   http.Header
	status   int
	headSent bool
}

func (s *streamWriter) Header() http.Header { return s.header }

func (s *streamWriter) WriteHeader(code int) { s.status = code }

func (s *streamWriter) Write(p []byte) (int, error) {
	if !s.headSent {
		s.sendHead()
	}
	_ = s.conn.writeFrame(frame{Type: "response-chunk", ID: s.id, Body: base64.StdEncoding.EncodeToString(p)})
	return len(p), nil
}

// Flush is a no-op marker that lets server-go's SSE handlers detect a flushable
// writer; each Write already emits its own frame.
func (s *streamWriter) Flush() {}

func (s *streamWriter) sendHead() {
	status := s.status
	if status == 0 {
		status = http.StatusOK
	}
	headers := map[string]string{}
	for k := range s.header {
		headers[k] = s.header.Get(k)
	}
	_ = s.conn.writeFrame(frame{Type: "response-head", ID: s.id, Status: status, Headers: headers})
	s.headSent = true
}

func (s *streamWriter) finish() {
	if !s.headSent {
		s.sendHead()
	}
	_ = s.conn.writeFrame(frame{Type: "response-end", ID: s.id})
}

func (c *Connector) load() error {
	if c.statePath == "" {
		return nil
	}
	return state.LoadJSON(c.statePath, &c.state)
}

func (c *Connector) save() error {
	if c.statePath == "" {
		return nil
	}
	return state.SaveJSON(c.statePath, c.state)
}

type frame struct {
	Type    string            `json:"type"`
	ID      string            `json:"id,omitempty"`
	Method  string            `json:"method,omitempty"`
	Path    string            `json:"path,omitempty"`
	Status  int               `json:"status,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

func relayURL(baseURL, relayPath string) (string, error) {
	if relayPath == "" {
		relayPath = "/v1/agent/connect"
	}
	u, err := url.Parse(baseURL + relayPath)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	return u.String(), nil
}

func nextBackoff(d time.Duration) time.Duration {
	d *= 2
	if d > 30*time.Second {
		return 30 * time.Second
	}
	return d
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

var _ = io.Discard // reserved

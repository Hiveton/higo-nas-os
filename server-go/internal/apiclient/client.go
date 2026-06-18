// Package apiclient is a typed Go client for the HiGoOS /api/v1 REST surface.
//
// It is the single transport used by every MCP tool. Two flavors exist:
//
//   - Remote: a normal *http.Client pointed at a base URL (used by the
//     standalone higo-mcp stdio binary, which talks to higo-api over the wire).
//   - In-process: a RoundTripper that dispatches directly into the API mux of
//     the running higo-api process (used by the embedded /mcp endpoint), so no
//     TCP loopback or listen-address guessing is needed.
//
// Both flavors decode the standard response envelope
// ({ok,data,error,requestId}) defined in internal/platform.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Auth carries the caller identity forwarded onto every request. For the
// embedded endpoint these are lifted from the incoming MCP HTTP request so
// loopback calls run as the real actor; for the stdio binary they come from
// configuration (HIGO_MCP_API_TOKEN).
type Auth struct {
	// Authorization sets the Authorization header verbatim (e.g. "Bearer xyz").
	Authorization string
	// SessionCookie sets the higo_session cookie value when non-empty.
	SessionCookie string
}

// Client issues envelope-aware requests against the HiGoOS API.
type Client struct {
	base string
	http *http.Client
	auth Auth
}

// NewRemote builds a Client that talks to a HiGoOS API over HTTP at baseURL
// (e.g. "http://127.0.0.1:8080"). A nil httpClient uses http.DefaultClient.
func NewRemote(baseURL string, auth Auth, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{base: strings.TrimRight(baseURL, "/"), http: httpClient, auth: auth}
}

// NewInProcess builds a Client that dispatches directly into handler (the API
// mux) without going over the network. The dummy base host is never dialed;
// only the request path/query matter.
func NewInProcess(handler http.Handler, auth Auth) *Client {
	return &Client{
		base: "http://higoos.local",
		http: &http.Client{Transport: handlerRoundTripper{handler: handler}},
		auth: auth,
	}
}

// WithAuth returns a shallow copy of the client that uses the given auth. The
// underlying http.Client/transport is shared, so this is cheap to call
// per-request (e.g. with auth forwarded from an incoming MCP call).
func (c *Client) WithAuth(auth Auth) *Client {
	clone := *c
	clone.auth = auth
	return &clone
}

// Error is a typed API error carrying the envelope error code and HTTP status.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("api error %d %s: %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("api error %d: %s", e.Status, e.Message)
}

// Do issues a request and unwraps the envelope. The raw JSON of the envelope's
// "data" field is returned on success; a *Error is returned otherwise.
//
//   - method: HTTP method.
//   - path: API path beginning with "/" (e.g. "/api/v1/system/info").
//   - query: optional query parameters (may be nil).
//   - body: optional JSON request body (may be nil).
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	endpoint := c.base + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.auth.Authorization != "" {
		req.Header.Set("Authorization", c.auth.Authorization)
	}
	if c.auth.SessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "higo_session", Value: c.auth.SessionCookie})
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Non-JSON or empty bodies (e.g. binary streams routed here by mistake).
	var env struct {
		OK    bool            `json:"ok"`
		Data  json.RawMessage `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &env); err != nil {
			if resp.StatusCode >= 400 {
				return nil, &Error{Status: resp.StatusCode, Code: "non_json_error", Message: strings.TrimSpace(string(raw))}
			}
			return nil, fmt.Errorf("decode envelope (%d): %w", resp.StatusCode, err)
		}
	}

	if resp.StatusCode >= 400 || env.Error != nil {
		apiErr := &Error{Status: resp.StatusCode}
		if env.Error != nil {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
		} else {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return nil, apiErr
	}

	if env.Data == nil {
		return json.RawMessage("null"), nil
	}
	return env.Data, nil
}

// URLResult builds a tool result for binary/stream endpoints that should not be
// piped through MCP as bytes. It returns a small JSON object containing the
// API-relative URL the caller can fetch out-of-band.
func URLResult(path string) json.RawMessage {
	obj := map[string]string{
		"url":  path,
		"note": "binary/stream endpoint; fetch this URL directly, not via MCP",
	}
	b, _ := json.Marshal(obj)
	return b
}

// handlerRoundTripper turns an http.Handler into an http.RoundTripper so the
// in-process Client can reuse the exact same handlers (and their governance)
// without a network hop.
type handlerRoundTripper struct {
	handler http.Handler
}

func (rt handlerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		req.Body = io.NopCloser(bytes.NewReader(nil))
	}
	rec := &recorder{header: make(http.Header), body: &bytes.Buffer{}, status: http.StatusOK}
	rt.handler.ServeHTTP(rec, req)
	resp := &http.Response{
		StatusCode: rec.status,
		Status:     http.StatusText(rec.status),
		Header:     rec.header,
		Body:       io.NopCloser(bytes.NewReader(rec.body.Bytes())),
		Request:    req,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	return resp, nil
}

// recorder is a minimal http.ResponseWriter capturing an in-process response.
type recorder struct {
	header      http.Header
	body        *bytes.Buffer
	status      int
	wroteHeader bool
}

func (r *recorder) Header() http.Header { return r.header }

func (r *recorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
}

func (r *recorder) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.body.Write(p)
}

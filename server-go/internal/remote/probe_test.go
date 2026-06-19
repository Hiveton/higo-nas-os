package remote

import (
	"context"
	"net"
	"strings"
	"testing"
)

func TestStartChannelMeasuresRealReachability(t *testing.T) {
	// A real local listener stands in for the relay/public endpoint.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	svc := NewService()
	svc.SetProbeTarget(ln.Addr().String())

	status, err := svc.StartChannel(context.Background())
	if err != nil {
		t.Fatalf("start channel: %v", err)
	}
	if !strings.Contains(status.TunnelState, "实测") {
		t.Fatalf("expected real measured latency in tunnel state, got %q", status.TunnelState)
	}
	if !strings.Contains(status.Feedback, "握手成功") {
		t.Fatalf("expected successful handshake feedback, got %q", status.Feedback)
	}
}

func TestStartChannelReportsUnreachableEndpoint(t *testing.T) {
	// Reserve a port then close it so the dial is refused.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	svc := NewService()
	svc.SetProbeTarget(addr)

	status, err := svc.StartChannel(context.Background())
	if err != nil {
		t.Fatalf("start channel: %v", err)
	}
	if !strings.Contains(status.TunnelState, "不可达") {
		t.Fatalf("expected unreachable tunnel state, got %q", status.TunnelState)
	}
}

func TestStartChannelDemoFallbackWithoutProbeTarget(t *testing.T) {
	svc := NewService() // no probe target → representative estimate, demo unbroken
	status, err := svc.StartChannel(context.Background())
	if err != nil {
		t.Fatalf("start channel: %v", err)
	}
	if !strings.Contains(status.TunnelState, "52ms") {
		t.Fatalf("expected demo estimate without a probe target, got %q", status.TunnelState)
	}
}

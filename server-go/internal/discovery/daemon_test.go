package discovery

import (
	"encoding/json"
	"testing"

	"higoos/server-go/internal/identity"
)

type fakeSnap struct{ id identity.Identity }

func (f fakeSnap) Snapshot() identity.Identity { return f.id }

func TestParseProbe(t *testing.T) {
	good := []byte(`{"magic":"HIGOOS/1","type":"discover","nonce":"abc"}`)
	if p, ok := parseProbe(good); !ok || p.Nonce != "abc" {
		t.Fatalf("valid probe rejected: %+v ok=%v", p, ok)
	}
	for _, bad := range []string{
		`{"magic":"OTHER/1","type":"discover"}`,
		`{"magic":"HIGOOS/1","type":"announce"}`,
		`not json`,
	} {
		if _, ok := parseProbe([]byte(bad)); ok {
			t.Fatalf("invalid probe accepted: %s", bad)
		}
	}
}

func TestBuildReplyEchoesNonceAndIdentity(t *testing.T) {
	snap := fakeSnap{id: identity.Identity{DeviceID: "hg-1234abcd", Model: "HiGoOS NAS", HTTPPort: 8080}}
	payload, err := buildReply(probe{Nonce: "xyz"}, snap)
	if err != nil {
		t.Fatalf("build reply: %v", err)
	}
	var out announce
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Magic != magic || out.Type != typeAnswer {
		t.Fatalf("wrong envelope: %+v", out)
	}
	if out.Nonce != "xyz" {
		t.Fatalf("nonce not echoed: %q", out.Nonce)
	}
	if out.DeviceID != "hg-1234abcd" || out.HTTPPort != 8080 {
		t.Fatalf("identity not embedded: %+v", out.Identity)
	}
}

func TestThrottle(t *testing.T) {
	th := newThrottle()
	if !th.allow("10.0.0.1") {
		t.Fatal("first call should be allowed")
	}
	if th.allow("10.0.0.1") {
		t.Fatal("immediate repeat should be throttled")
	}
	if !th.allow("10.0.0.2") {
		t.Fatal("a different source should be allowed")
	}
}

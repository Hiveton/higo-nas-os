// Package iscsi is the iSCSI target domain. On Linux it drives the LIO kernel
// target: listing reads configfs (/sys/kernel/config/target/iscsi) and mutations
// shell out to `targetcli`; on dev hosts it uses a devstub. Deleting a target is
// destructive (governed by a UI confirm). Exporting block storage is high-risk.
package iscsi

import "time"

// Target is one iSCSI target (IQN) and its LUNs/ACLs/portals.
type Target struct {
	IQN     string   `json:"iqn"`
	LUNs    int      `json:"luns"`
	ACLs    []string `json:"acls"`    // allowed initiator IQNs
	Portals []string `json:"portals"` // ip:port
}

// HostCaps reports whether the LIO iSCSI target stack is usable.
type HostCaps struct {
	Available bool   `json:"available"` // targetcli + configfs present
	Backend   string `json:"backend"`   // LIO / devstub / none
	Portal    string `json:"portal"`    // default portal
	Note      string `json:"note,omitempty"`
}

// AuditEntry is the append-only action log (newest first).
type AuditEntry struct {
	ID     string    `json:"id"`
	Event  string    `json:"event"`
	Actor  string    `json:"actor,omitempty"`
	Target string    `json:"target,omitempty"`
	Action string    `json:"action"`
	Result string    `json:"result"`
	Time   time.Time `json:"time"`
}

// CreateTargetRequest creates an iSCSI target (IQN auto-generated when empty).
type CreateTargetRequest struct {
	IQN   string `json:"iqn,omitempty"`
	Actor string `json:"actor,omitempty"`
}

// AddLUNRequest adds a fileio-backed LUN to a target.
type AddLUNRequest struct {
	Name   string `json:"name"`
	SizeMB int    `json:"sizeMB"`
	Actor  string `json:"actor,omitempty"`
}

// AddACLRequest authorizes an initiator IQN on a target.
type AddACLRequest struct {
	Initiator string `json:"initiator"`
	Actor     string `json:"actor,omitempty"`
}

func cloneTargets(in []Target) []Target {
	out := make([]Target, 0, len(in))
	for _, t := range in {
		t.ACLs = append([]string(nil), t.ACLs...)
		t.Portals = append([]string(nil), t.Portals...)
		out = append(out, t)
	}
	return out
}

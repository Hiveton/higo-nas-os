package remote

import (
	"context"
	"strings"
	"testing"
)

func TestServiceSeedsFrontendRemoteStateDevicesPoliciesAndAlerts(t *testing.T) {
	service := NewService()

	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	devices, err := service.Devices(context.Background())
	if err != nil {
		t.Fatalf("devices: %v", err)
	}
	alerts, err := service.LoginAlerts(context.Background())
	if err != nil {
		t.Fatalf("login alerts: %v", err)
	}

	if !status.ChannelEnabled || status.ChannelState != "在线" {
		t.Fatalf("expected remote channel online, got %#v", status)
	}
	if status.Domain != "home-3.higo.link" || status.TunnelMode != TunnelModeRelay || !status.MFAEnabled {
		t.Fatalf("unexpected seeded status: %#v", status)
	}
	if len(status.Policies) != 3 || status.ActivePolicy.Key != "family" {
		t.Fatalf("expected 3 policies with family active, got %#v", status)
	}
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}
	if !devices[0].Bound || !devices[1].Bound || devices[2].Bound {
		t.Fatalf("unexpected seeded device binding state: %#v", devices)
	}
	if len(alerts) != 2 || alerts[0].Action != "已要求 MFA" {
		t.Fatalf("unexpected login alerts: %#v", alerts)
	}
}

func TestChannelMFAAndTunnelMutationsUpdateStatus(t *testing.T) {
	service := NewService()

	stopped, err := service.StopChannel(context.Background())
	if err != nil {
		t.Fatalf("stop channel: %v", err)
	}
	if stopped.ChannelEnabled || stopped.ChannelState != "已暂停" || !strings.Contains(stopped.TunnelState, "通道关闭") {
		t.Fatalf("expected stopped channel state, got %#v", stopped)
	}

	started, err := service.StartChannel(context.Background())
	if err != nil {
		t.Fatalf("start channel: %v", err)
	}
	if !started.ChannelEnabled || started.ChannelState != "在线" {
		t.Fatalf("expected started channel state, got %#v", started)
	}

	direct, err := service.UpdateTunnelMode(context.Background(), TunnelModeDirect)
	if err != nil {
		t.Fatalf("update tunnel mode: %v", err)
	}
	if direct.TunnelMode != TunnelModeDirect || !strings.Contains(direct.TunnelState, "直连优先") {
		t.Fatalf("expected direct tunnel state, got %#v", direct)
	}

	mfaOff, err := service.UpdateMFA(context.Background(), false)
	if err != nil {
		t.Fatalf("disable mfa: %v", err)
	}
	if mfaOff.MFAEnabled {
		t.Fatalf("expected MFA disabled, got %#v", mfaOff)
	}
	mfaOn, err := service.UpdateMFA(context.Background(), true)
	if err != nil {
		t.Fatalf("enable mfa: %v", err)
	}
	if !mfaOn.MFAEnabled || !strings.Contains(mfaOn.Feedback, "多因素认证已启用") {
		t.Fatalf("expected MFA enabled feedback, got %#v", mfaOn)
	}
}

func TestDomainTokenCreateAndRotateBumpVersionAndExpiry(t *testing.T) {
	service := NewService()

	created, err := service.CreateDomainToken(context.Background())
	if err != nil {
		t.Fatalf("create domain token: %v", err)
	}
	if created.Version != 3 || created.Domain != "home-3.higo.link" || created.Token == "" || created.ExpiresAt.IsZero() {
		t.Fatalf("unexpected created token: %#v", created)
	}

	rotated, err := service.RotateDomainToken(context.Background())
	if err != nil {
		t.Fatalf("rotate domain token: %v", err)
	}
	if rotated.Version != 4 || rotated.Domain != "home-4.higo.link" || rotated.Token == created.Token {
		t.Fatalf("expected rotated token version/domain/value, got created=%#v rotated=%#v", created, rotated)
	}

	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("status after rotate: %v", err)
	}
	if status.Domain != rotated.Domain || status.Token.Version != rotated.Version {
		t.Fatalf("status did not reflect rotated token: %#v", status)
	}
}

func TestDeviceBindingAndUnbindingAreObservableAndCloned(t *testing.T) {
	service := NewService()

	bound, err := service.BindDevice(context.Background(), "ipad")
	if err != nil {
		t.Fatalf("bind ipad: %v", err)
	}
	if !bound.Bound || bound.LastSeen != "刚刚绑定" {
		t.Fatalf("expected bound ipad, got %#v", bound)
	}

	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("status after bind: %v", err)
	}
	if status.BoundDeviceCount != 3 {
		t.Fatalf("expected 3 bound devices after bind, got %d", status.BoundDeviceCount)
	}

	unbound, err := service.UnbindDevice(context.Background(), "ipad")
	if err != nil {
		t.Fatalf("unbind ipad: %v", err)
	}
	if unbound.Bound || unbound.LastSeen != "已解绑" {
		t.Fatalf("expected unbound ipad, got %#v", unbound)
	}

	devices, err := service.Devices(context.Background())
	if err != nil {
		t.Fatalf("devices: %v", err)
	}
	devices[0].Name = "mutated by test"
	nextDevices, err := service.Devices(context.Background())
	if err != nil {
		t.Fatalf("devices again: %v", err)
	}
	if nextDevices[0].Name == "mutated by test" {
		t.Fatal("devices must be cloned before returning")
	}
}

func TestScanShareLinksAlternatesRiskAndSafeResults(t *testing.T) {
	service := NewService()

	risk, err := service.ScanShareLinks(context.Background())
	if err != nil {
		t.Fatalf("scan share links: %v", err)
	}
	if risk.State != ShareScanRisk || !strings.Contains(risk.Message, "公开下载权限") || len(risk.Checks) != 4 {
		t.Fatalf("expected first scan to find public download risk, got %#v", risk)
	}

	safe, err := service.ScanShareLinks(context.Background())
	if err != nil {
		t.Fatalf("scan share links again: %v", err)
	}
	if safe.State != ShareScanSafe || !strings.Contains(safe.Message, "仅限家庭访问") {
		t.Fatalf("expected second scan to be safe, got %#v", safe)
	}
}

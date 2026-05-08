package remote

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

type Service struct {
	mu           sync.RWMutex
	enabled      bool
	mfaEnabled   bool
	tunnelMode   TunnelMode
	token        DomainToken
	devices      []BoundDevice
	policies     []AccessPolicy
	activePolicy string
	alerts       []LoginAlert
	scanState    ShareScanState
	feedback     string
	statePath    string
}

type snapshot struct {
	Enabled      bool           `json:"enabled"`
	MFAEnabled   bool           `json:"mfaEnabled"`
	TunnelMode   TunnelMode     `json:"tunnelMode"`
	Token        DomainToken    `json:"token"`
	Devices      []BoundDevice  `json:"devices"`
	Policies     []AccessPolicy `json:"policies"`
	ActivePolicy string         `json:"activePolicy"`
	Alerts       []LoginAlert   `json:"alerts"`
	ScanState    ShareScanState `json:"scanState"`
	Feedback     string         `json:"feedback"`
}

func NewService() *Service {
	policies := []AccessPolicy{
		{Key: "family", Name: "家庭访问", Scope: "家庭成员可远程预览照片和文档", Risk: "低风险"},
		{Key: "team", Name: "团队协作", Scope: "允许项目空间 WebDAV 与分享链接", Risk: "中风险"},
		{Key: "guest", Name: "访客临时", Scope: "仅限 24 小时只读链接", Risk: "低风险"},
	}
	return &Service{
		enabled:    true,
		mfaEnabled: true,
		tunnelMode: TunnelModeRelay,
		token:      newDomainToken(3),
		devices: []BoundDevice{
			{ID: "iphone", Name: "Hiveton iPhone", Role: "管理员设备", Location: "上海", Bound: true, LastSeen: "刚刚"},
			{ID: "macbook", Name: "MacBook Pro", Role: "可信电脑", Location: "杭州", Bound: true, LastSeen: "12 分钟前"},
			{ID: "ipad", Name: "Family iPad", Role: "家庭成员", Location: "南京", Bound: false, LastSeen: "待绑定"},
		},
		policies:     policies,
		activePolicy: "family",
		alerts: []LoginAlert{
			{ID: "login-1", Location: "深圳", Device: "Chrome / Windows", Action: "已要求 MFA", State: "待确认"},
			{ID: "login-2", Location: "东京", Device: "Safari / iPhone", Action: "策略拒绝", State: "已拦截"},
		},
		scanState: ShareScanIdle,
		feedback:  "远程访问通道在线，最近一次策略审计于 14:10 完成",
	}
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "remote.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if !isZeroSnapshot(persisted) {
		service.enabled = persisted.Enabled
		service.mfaEnabled = persisted.MFAEnabled
		service.tunnelMode = persisted.TunnelMode
		service.token = persisted.Token
		service.devices = cloneDevices(persisted.Devices)
		service.policies = append([]AccessPolicy(nil), persisted.Policies...)
		service.activePolicy = persisted.ActivePolicy
		service.alerts = append([]LoginAlert(nil), persisted.Alerts...)
		service.scanState = persisted.ScanState
		service.feedback = persisted.Feedback
		if service.tunnelMode == "" {
			service.tunnelMode = TunnelModeRelay
		}
		if service.token.Domain == "" {
			service.token = newDomainToken(3)
		}
		if len(service.policies) == 0 {
			service.policies = NewService().policies
		}
		if service.activePolicy == "" {
			service.activePolicy = service.policies[0].Key
		}
	}
	return service, nil
}

func (s *Service) Status(ctx context.Context) (RemoteStatus, error) {
	if err := ctx.Err(); err != nil {
		return RemoteStatus{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLocked(), nil
}

func (s *Service) StartChannel(ctx context.Context) (RemoteStatus, error) {
	if err := ctx.Err(); err != nil {
		return RemoteStatus{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = true
	if s.tunnelMode == TunnelModeDisabled {
		s.tunnelMode = TunnelModeRelay
	}
	s.feedback = "远程通道已启动，内网穿透重新握手成功"
	if err := s.saveLocked(); err != nil {
		return RemoteStatus{}, err
	}
	return s.statusLocked(), nil
}

func (s *Service) StopChannel(ctx context.Context) (RemoteStatus, error) {
	if err := ctx.Err(); err != nil {
		return RemoteStatus{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = false
	s.feedback = "远程通道已暂停，新连接会被拒绝"
	if err := s.saveLocked(); err != nil {
		return RemoteStatus{}, err
	}
	return s.statusLocked(), nil
}

func (s *Service) UpdateTunnelMode(ctx context.Context, mode TunnelMode) (RemoteStatus, error) {
	if err := ctx.Err(); err != nil {
		return RemoteStatus{}, err
	}
	if !validTunnelMode(mode) {
		return RemoteStatus{}, fmt.Errorf("remote tunnel mode not supported: %s", mode)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tunnelMode = mode
	if mode == TunnelModeDisabled {
		s.enabled = false
	}
	s.feedback = fmt.Sprintf("内网穿透模式已切换为 %s", mode)
	if err := s.saveLocked(); err != nil {
		return RemoteStatus{}, err
	}
	return s.statusLocked(), nil
}

func (s *Service) UpdateMFA(ctx context.Context, enabled bool) (RemoteStatus, error) {
	if err := ctx.Err(); err != nil {
		return RemoteStatus{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mfaEnabled = enabled
	if enabled {
		s.feedback = "多因素认证已启用，异地登录必须二次确认"
	} else {
		s.feedback = "多因素认证已关闭，已写入安全审计"
	}
	if err := s.saveLocked(); err != nil {
		return RemoteStatus{}, err
	}
	return s.statusLocked(), nil
}

func (s *Service) CreateDomainToken(ctx context.Context) (DomainToken, error) {
	if err := ctx.Err(); err != nil {
		return DomainToken{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = newDomainToken(s.token.Version)
	s.feedback = fmt.Sprintf("已复制 %s 的短期访问令牌，有效期 10 分钟", s.token.Domain)
	return s.token, s.saveLocked()
}

func (s *Service) RotateDomainToken(ctx context.Context) (DomainToken, error) {
	if err := ctx.Err(); err != nil {
		return DomainToken{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = newDomainToken(s.token.Version + 1)
	s.feedback = fmt.Sprintf("远程域名令牌已轮换，新域名 %s 已生效", s.token.Domain)
	return s.token, s.saveLocked()
}

func (s *Service) Devices(ctx context.Context) ([]BoundDevice, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneDevices(s.devices), nil
}

func (s *Service) BindDevice(ctx context.Context, deviceID string) (BoundDevice, error) {
	if err := ctx.Err(); err != nil {
		return BoundDevice{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findDeviceLocked(deviceID)
	if !ok {
		return BoundDevice{}, fmt.Errorf("remote device not found: %s", deviceID)
	}
	s.devices[index].Bound = true
	s.devices[index].LastSeen = "刚刚绑定"
	s.feedback = fmt.Sprintf("%s 已绑定为可信设备", s.devices[index].Name)
	return s.devices[index], s.saveLocked()
}

func (s *Service) UnbindDevice(ctx context.Context, deviceID string) (BoundDevice, error) {
	if err := ctx.Err(); err != nil {
		return BoundDevice{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findDeviceLocked(deviceID)
	if !ok {
		return BoundDevice{}, fmt.Errorf("remote device not found: %s", deviceID)
	}
	s.devices[index].Bound = false
	s.devices[index].LastSeen = "已解绑"
	s.feedback = fmt.Sprintf("%s 已解绑，远程令牌已失效", s.devices[index].Name)
	return s.devices[index], s.saveLocked()
}

func (s *Service) LoginAlerts(ctx context.Context) ([]LoginAlert, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]LoginAlert(nil), s.alerts...), nil
}

func (s *Service) ScanShareLinks(ctx context.Context) (ShareScanResult, error) {
	if err := ctx.Err(); err != nil {
		return ShareScanResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scanState == ShareScanRisk {
		s.scanState = ShareScanSafe
	} else {
		s.scanState = ShareScanRisk
	}
	result := shareScanResult(s.scanState)
	s.feedback = result.Message
	return result, s.saveLocked()
}

func (s *Service) Policies(ctx context.Context) ([]AccessPolicy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]AccessPolicy(nil), s.policies...), nil
}

func (s *Service) SelectPolicy(ctx context.Context, key string) (RemoteStatus, error) {
	if err := ctx.Err(); err != nil {
		return RemoteStatus{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	policy, ok := s.findPolicyLocked(key)
	if !ok {
		return RemoteStatus{}, fmt.Errorf("remote access policy not found: %s", key)
	}
	s.activePolicy = key
	s.feedback = fmt.Sprintf("访问策略已切换为「%s」", policy.Name)
	if err := s.saveLocked(); err != nil {
		return RemoteStatus{}, err
	}
	return s.statusLocked(), nil
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Enabled:      s.enabled,
		MFAEnabled:   s.mfaEnabled,
		TunnelMode:   s.tunnelMode,
		Token:        s.token,
		Devices:      cloneDevices(s.devices),
		Policies:     append([]AccessPolicy(nil), s.policies...),
		ActivePolicy: s.activePolicy,
		Alerts:       append([]LoginAlert(nil), s.alerts...),
		ScanState:    s.scanState,
		Feedback:     s.feedback,
	})
}

func (s *Service) statusLocked() RemoteStatus {
	activePolicy, _ := s.findPolicyLocked(s.activePolicy)
	policies := append([]AccessPolicy(nil), s.policies...)
	devices := cloneDevices(s.devices)
	bound := 0
	for _, device := range devices {
		if device.Bound {
			bound++
		}
	}
	return RemoteStatus{
		ChannelEnabled:   s.enabled,
		ChannelState:     channelState(s.enabled),
		Domain:           s.token.Domain,
		TunnelMode:       s.tunnelMode,
		TunnelState:      tunnelState(s.enabled, s.tunnelMode),
		MFAEnabled:       s.mfaEnabled,
		Token:            s.token,
		TokenState:       "短期访问令牌有效期 10 分钟",
		BoundDeviceCount: bound,
		DeviceCount:      len(devices),
		ActivePolicy:     activePolicy,
		Policies:         policies,
		Feedback:         s.feedback,
	}
}

func (s *Service) findDeviceLocked(deviceID string) (int, bool) {
	for index, device := range s.devices {
		if device.ID == deviceID {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findPolicyLocked(key string) (AccessPolicy, bool) {
	for _, policy := range s.policies {
		if policy.Key == key {
			return policy, true
		}
	}
	return AccessPolicy{}, false
}

func newDomainToken(version int) DomainToken {
	return DomainToken{
		Version:   version,
		Domain:    fmt.Sprintf("home-%d.higo.link", version),
		Token:     fmt.Sprintf("higo-remote-v%d-%s", version, time.Now().UTC().Format("20060102150405")),
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	}
}

func validTunnelMode(mode TunnelMode) bool {
	switch mode {
	case TunnelModeRelay, TunnelModeDirect, TunnelModeDisabled:
		return true
	default:
		return false
	}
}

func channelState(enabled bool) string {
	if enabled {
		return "在线"
	}
	return "已暂停"
}

func tunnelState(enabled bool, mode TunnelMode) string {
	if !enabled || mode == TunnelModeDisabled {
		return "通道关闭，外部请求被拒绝"
	}
	return fmt.Sprintf("%s · TLS 1.3 · 52ms", mode)
}

func shareScanResult(state ShareScanState) ShareScanResult {
	result := ShareScanResult{
		State:  state,
		Checks: []string{"公开分享范围", "过期时间", "下载权限", "敏感文件标签"},
	}
	switch state {
	case ShareScanSafe:
		result.Message = "扫描完成：分享链接仅限家庭访问，7 天后自动过期。"
	case ShareScanRisk:
		result.Message = "扫描完成：发现公开下载权限，建议切换为访客临时策略。"
	default:
		result.Message = "等待扫描：检查分享链接范围、过期时间和敏感标签。"
	}
	return result
}

func cloneDevices(devices []BoundDevice) []BoundDevice {
	return append([]BoundDevice(nil), devices...)
}

func isZeroSnapshot(snapshot snapshot) bool {
	return !snapshot.Enabled &&
		!snapshot.MFAEnabled &&
		snapshot.TunnelMode == "" &&
		snapshot.Token.Domain == "" &&
		len(snapshot.Devices) == 0 &&
		len(snapshot.Policies) == 0 &&
		snapshot.ActivePolicy == "" &&
		len(snapshot.Alerts) == 0 &&
		snapshot.ScanState == "" &&
		snapshot.Feedback == ""
}

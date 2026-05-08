package security

import (
	"time"

	"higoos/server-go/internal/audit"
	"higoos/server-go/internal/iam"
)

type RiskActionState string

const (
	RiskActionStatePending   RiskActionState = "待处理"
	RiskActionStateConfirmed RiskActionState = "已确认"
	RiskActionStateBlocked   RiskActionState = "已阻止"
)

type IdentityPolicy struct {
	ID       string   `json:"id"`
	Role     string   `json:"role"`
	IAMRole  iam.Role `json:"iamRole"`
	Name     string   `json:"name"`
	MFA      bool     `json:"mfa"`
	FileACL  bool     `json:"fileAcl"`
	AppAdmin bool     `json:"appAdmin"`
	AITools  bool     `json:"aiTools"`
}

type IdentityPermissions struct {
	MFA      bool `json:"mfa"`
	FileACL  bool `json:"fileAcl"`
	AppAdmin bool `json:"appAdmin"`
	AITools  bool `json:"aiTools"`
}

type AiPolicy struct {
	ID         string `json:"id"`
	Space      string `json:"space"`
	SpaceID    string `json:"spaceId"`
	Indexed    bool   `json:"indexed"`
	CloudModel bool   `json:"cloudModel"`
	Sensitive  string `json:"sensitive"`
}

type AiPolicyUpdate struct {
	Indexed    *bool  `json:"indexed,omitempty"`
	CloudModel *bool  `json:"cloudModel,omitempty"`
	Sensitive  string `json:"sensitive,omitempty"`
}

type ShareLinkRisk struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Target    string          `json:"target"`
	Access    string          `json:"access"`
	Downloads int             `json:"downloads"`
	Risk      audit.RiskLevel `json:"risk"`
	RiskLabel string          `json:"riskLabel"`
	Active    bool            `json:"active"`
}

type SecurityRiskAction struct {
	ID                 string                 `json:"id"`
	Title              string                 `json:"title"`
	Level              audit.RiskLevel        `json:"level"`
	LevelLabel         string                 `json:"levelLabel"`
	Scope              string                 `json:"scope"`
	Actor              string                 `json:"actor"`
	State              RiskActionState        `json:"state"`
	Confirmed          bool                   `json:"confirmed"`
	Rollback           string                 `json:"rollback"`
	RequiredPermission string                 `json:"requiredPermission"`
	AffectedItemCount  int                    `json:"affectedItemCount"`
	AuditStatus        audit.RiskActionStatus `json:"auditStatus"`
}

type SecurityAuditEntry struct {
	ID        string            `json:"id"`
	Event     string            `json:"event"`
	Actor     string            `json:"actor"`
	Risk      audit.RiskLevel   `json:"risk"`
	RiskLabel string            `json:"riskLabel"`
	Reverted  bool              `json:"reverted"`
	Rollback  string            `json:"rollback"`
	Result    audit.AuditResult `json:"result"`
	Time      time.Time         `json:"time"`
}

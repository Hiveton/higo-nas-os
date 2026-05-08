package iam

type Role string

const (
	RoleAdmin        Role = "admin"
	RoleFamilyMember Role = "family_member"
	RoleTeamMember   Role = "team_member"
	RoleGuest        Role = "guest"
	RoleApp          Role = "app"
	RoleAgent        Role = "agent"
	RoleWorker       Role = "worker"
)

type SpaceType string

const (
	SpacePersonal SpaceType = "personal"
	SpaceFamily   SpaceType = "family"
	SpaceTeam     SpaceType = "team"
	SpaceShared   SpaceType = "shared"
	SpaceSystem   SpaceType = "system"
)

type ScopeType string

const (
	ScopeSpace ScopeType = "space"
	ScopeFile  ScopeType = "file"
	ScopeTool  ScopeType = "tool"
)

type Action string

const (
	ActionRead    Action = "read"
	ActionWrite   Action = "write"
	ActionDelete  Action = "delete"
	ActionShare   Action = "share"
	ActionManage  Action = "manage"
	ActionExecute Action = "execute"
)

type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

type Space struct {
	ID      string
	Type    SpaceType
	OwnerID string
}

type Scope struct {
	Type ScopeType
	ID   string
}

type Permission struct {
	ActorID string
	Role    Role
	SpaceID string
	Scope   Scope
	Action  Action
	Effect  Effect
	Reason  string
}

type EvaluationInput struct {
	ActorID string
	Role    Role
	Space   Space
	Scope   Scope
	Action  Action
}

type ACLDecision struct {
	Allowed bool
	Effect  Effect
	Reason  string
}

type Evaluator struct {
	permissions []Permission
}

func NewEvaluator(permissions ...Permission) Evaluator {
	return Evaluator{permissions: append([]Permission(nil), permissions...)}
}

func (e Evaluator) Evaluate(input EvaluationInput) ACLDecision {
	var allowed *Permission
	for i := range e.permissions {
		permission := e.permissions[i]
		if !permission.matches(input) {
			continue
		}
		if permission.Effect == EffectDeny {
			return ACLDecision{Allowed: false, Effect: EffectDeny, Reason: permission.reasonOr("explicit deny")}
		}
		if permission.Effect == EffectAllow {
			permissionCopy := permission
			allowed = &permissionCopy
		}
	}
	if allowed != nil {
		return ACLDecision{Allowed: true, Effect: EffectAllow, Reason: allowed.reasonOr("explicit allow")}
	}

	if builtinAllows(input) {
		return ACLDecision{Allowed: true, Effect: EffectAllow, Reason: "role policy allows action"}
	}
	return ACLDecision{Allowed: false, Effect: EffectDeny, Reason: "role policy denies action"}
}

func (p Permission) matches(input EvaluationInput) bool {
	if p.ActorID != "" && p.ActorID != input.ActorID {
		return false
	}
	if p.Role != "" && p.Role != input.Role {
		return false
	}
	if p.SpaceID != "" && p.SpaceID != "*" && p.SpaceID != input.Space.ID {
		return false
	}
	if p.Scope.Type != "" && p.Scope.Type != input.Scope.Type {
		return false
	}
	if p.Scope.ID != "" && p.Scope.ID != "*" && p.Scope.ID != input.Scope.ID {
		return false
	}
	if p.Action != "" && p.Action != "*" && p.Action != input.Action {
		return false
	}
	return true
}

func (p Permission) reasonOr(fallback string) string {
	if p.Reason != "" {
		return p.Reason
	}
	return fallback
}

func builtinAllows(input EvaluationInput) bool {
	switch input.Role {
	case RoleAdmin:
		return true
	case RoleFamilyMember:
		return allowsFamilyMember(input)
	case RoleTeamMember:
		return input.Space.Type == SpaceTeam && isReadWrite(input.Action) && input.Scope.Type != ScopeTool
	case RoleGuest:
		return input.Space.Type == SpaceShared && input.Action == ActionRead && input.Scope.Type != ScopeTool
	default:
		return false
	}
}

func allowsFamilyMember(input EvaluationInput) bool {
	if input.Scope.Type == ScopeTool {
		return false
	}
	switch input.Space.Type {
	case SpacePersonal:
		return input.Space.OwnerID == input.ActorID && isReadWrite(input.Action)
	case SpaceFamily:
		return input.Action == ActionRead || input.Action == ActionWrite || input.Action == ActionShare
	case SpaceShared:
		return input.Action == ActionRead
	default:
		return false
	}
}

func isReadWrite(action Action) bool {
	return action == ActionRead || action == ActionWrite
}

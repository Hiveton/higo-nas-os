package accounts

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"higoos/server-go/internal/state"
)

type Service struct {
	mu        sync.RWMutex
	seq       int
	users     []User
	groups    []Group
	grants    []SpaceGrant
	now       func() time.Time
	statePath string
}

type snapshot struct {
	Seq    int          `json:"seq"`
	Users  []User       `json:"users"`
	Groups []Group      `json:"groups"`
	Grants []SpaceGrant `json:"grants"`
}

func NewService() *Service {
	service := &Service{now: time.Now}
	service.seed()
	return service
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "accounts.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Users) > 0 || len(persisted.Groups) > 0 || len(persisted.Grants) > 0 {
		service.seq = persisted.Seq
		service.users = cloneUsers(persisted.Users)
		service.groups = cloneGroups(persisted.Groups)
		service.grants = cloneGrants(persisted.Grants)
	}
	return service, nil
}

func (s *Service) Summary(ctx context.Context) (Summary, error) {
	if err := ctx.Err(); err != nil {
		return Summary{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Summary{
		Users:  cloneUsers(s.users),
		Groups: cloneGroups(s.groups),
		Grants: cloneGrants(s.grants),
	}, nil
}

func (s *Service) CreateUser(ctx context.Context, request CreateUserRequest) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	username := strings.TrimSpace(request.Username)
	if username == "" {
		return User{}, fmt.Errorf("username is required")
	}
	if err := validatePassword(request.Password); err != nil {
		return User{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.findUserByUsernameLocked(username); ok {
		return User{}, fmt.Errorf("username already exists: %s", username)
	}
	now := s.now().UTC()
	s.seq++
	user := User{
		ID:          fmt.Sprintf("user-%03d-%s", s.seq, slugID(username)),
		Username:    username,
		DisplayName: strings.TrimSpace(request.DisplayName),
		Role:        defaultRole(request.Role),
		Status:      StatusActive,
		QuotaBytes:  request.QuotaBytes,
		Groups:      uniqueStrings(request.Groups),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if user.DisplayName == "" {
		user.DisplayName = username
	}
	s.users = append(s.users, user)
	s.syncGroupMembershipLocked(user.ID, user.Groups)
	return user, s.saveLocked()
}

func (s *Service) UpdateUser(ctx context.Context, id string, request UpdateUserRequest) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findUserLocked(id)
	if !ok {
		return User{}, fmt.Errorf("user not found: %s", id)
	}
	user := &s.users[index]
	if request.DisplayName != "" {
		user.DisplayName = strings.TrimSpace(request.DisplayName)
	}
	if request.Password != "" {
		if err := validatePassword(request.Password); err != nil {
			return User{}, err
		}
	}
	if request.Role != "" {
		user.Role = request.Role
	}
	if request.Status != "" {
		user.Status = request.Status
	}
	if request.QuotaBytes != nil {
		user.QuotaBytes = *request.QuotaBytes
	}
	if request.Groups != nil {
		user.Groups = uniqueStrings(request.Groups)
		s.syncGroupMembershipLocked(user.ID, user.Groups)
	}
	user.UpdatedAt = s.now().UTC()
	return *user, s.saveLocked()
}

func (s *Service) DeleteUser(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findUserLocked(id)
	if !ok {
		return fmt.Errorf("user not found: %s", id)
	}
	s.users = append(s.users[:index], s.users[index+1:]...)
	for i := range s.groups {
		s.groups[i].UserIDs = removeString(s.groups[i].UserIDs, id)
	}
	nextGrants := s.grants[:0]
	for _, grant := range s.grants {
		if !(grant.SubjectType == SubjectUser && grant.SubjectID == id) {
			nextGrants = append(nextGrants, grant)
		}
	}
	s.grants = nextGrants
	return s.saveLocked()
}

func (s *Service) CreateGroup(ctx context.Context, request CreateGroupRequest) (Group, error) {
	if err := ctx.Err(); err != nil {
		return Group{}, err
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return Group{}, fmt.Errorf("group name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	s.seq++
	group := Group{
		ID:          fmt.Sprintf("group-%03d-%s", s.seq, slugID(name)),
		Name:        name,
		Description: strings.TrimSpace(request.Description),
		UserIDs:     uniqueStrings(request.UserIDs),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.groups = append(s.groups, group)
	return group, s.saveLocked()
}

func (s *Service) UpdateGroupMembers(ctx context.Context, id string, request UpdateGroupMembersRequest) (Group, error) {
	if err := ctx.Err(); err != nil {
		return Group{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findGroupLocked(id)
	if !ok {
		return Group{}, fmt.Errorf("group not found: %s", id)
	}
	group := &s.groups[index]
	group.UserIDs = uniqueStrings(request.UserIDs)
	group.UpdatedAt = s.now().UTC()
	for i := range s.users {
		if containsString(group.UserIDs, s.users[i].ID) {
			if !containsString(s.users[i].Groups, group.ID) {
				s.users[i].Groups = append(s.users[i].Groups, group.ID)
			}
		} else {
			s.users[i].Groups = removeString(s.users[i].Groups, group.ID)
		}
	}
	return *group, s.saveLocked()
}

func (s *Service) GrantSpace(ctx context.Context, request SpaceGrantRequest) (SpaceGrant, error) {
	if err := ctx.Err(); err != nil {
		return SpaceGrant{}, err
	}
	if request.SubjectID == "" || request.SpaceID == "" {
		return SpaceGrant{}, fmt.Errorf("subjectId and spaceId are required")
	}
	subjectType := request.SubjectType
	if subjectType == "" {
		subjectType = SubjectUser
	}
	access := request.Access
	if access == "" {
		access = AccessReadOnly
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	for index := range s.grants {
		grant := &s.grants[index]
		if grant.SubjectID == request.SubjectID && grant.SubjectType == subjectType && grant.SpaceID == request.SpaceID {
			grant.Access = access
			grant.QuotaBytes = request.QuotaBytes
			grant.UpdatedAt = now
			return *grant, s.saveLocked()
		}
	}
	s.seq++
	grant := SpaceGrant{
		ID:          fmt.Sprintf("grant-%03d", s.seq),
		SubjectID:   request.SubjectID,
		SubjectType: subjectType,
		SpaceID:     request.SpaceID,
		Access:      access,
		QuotaBytes:  request.QuotaBytes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.grants = append(s.grants, grant)
	return grant, s.saveLocked()
}

func (s *Service) DeleteSpaceGrants(ctx context.Context, spaceID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if strings.TrimSpace(spaceID) == "" {
		return 0, fmt.Errorf("spaceId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	nextGrants := s.grants[:0]
	for _, grant := range s.grants {
		if grant.SpaceID == spaceID {
			removed++
			continue
		}
		nextGrants = append(nextGrants, grant)
	}
	s.grants = nextGrants
	if removed == 0 {
		return 0, nil
	}
	return removed, s.saveLocked()
}

func (s *Service) DeleteGrant(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("grant id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := -1
	for i, grant := range s.grants {
		if grant.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("grant not found: %s", id)
	}
	s.grants = append(s.grants[:index], s.grants[index+1:]...)
	return s.saveLocked()
}

func (s *Service) seed() {
	now := time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC)
	s.users = []User{
		{ID: "admin", Username: "admin", DisplayName: "管理员", Role: RoleAdmin, Status: StatusActive, QuotaBytes: 0, Groups: []string{"admins"}, CreatedAt: now, UpdatedAt: now},
	}
	s.groups = []Group{
		{ID: "admins", Name: "管理员组", Description: "系统管理权限", UserIDs: []string{"admin"}, CreatedAt: now, UpdatedAt: now},
		{ID: "family", Name: "家庭成员", Description: "家庭空间访问", UserIDs: []string{}, CreatedAt: now, UpdatedAt: now},
	}
	s.grants = []SpaceGrant{}
	s.seq = 2
}

func (s *Service) findUserLocked(id string) (int, bool) {
	for index, user := range s.users {
		if user.ID == id {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findUserByUsernameLocked(username string) (int, bool) {
	for index, user := range s.users {
		if strings.EqualFold(user.Username, username) {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findGroupLocked(id string) (int, bool) {
	for index, group := range s.groups {
		if group.ID == id {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) syncGroupMembershipLocked(userID string, groupIDs []string) {
	for i := range s.groups {
		if containsString(groupIDs, s.groups[i].ID) {
			if !containsString(s.groups[i].UserIDs, userID) {
				s.groups[i].UserIDs = append(s.groups[i].UserIDs, userID)
			}
		} else {
			s.groups[i].UserIDs = removeString(s.groups[i].UserIDs, userID)
		}
	}
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Seq:    s.seq,
		Users:  cloneUsers(s.users),
		Groups: cloneGroups(s.groups),
		Grants: cloneGrants(s.grants),
	})
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("password must include letters and digits")
	}
	return nil
}

func defaultRole(role UserRole) UserRole {
	if role == "" {
		return RoleUser
	}
	return role
}

func slugID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 && b.String()[b.Len()-1] != '-' {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "item"
	}
	return out
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func removeString(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func cloneUsers(users []User) []User {
	out := make([]User, 0, len(users))
	for _, user := range users {
		user.Groups = append([]string{}, user.Groups...)
		out = append(out, user)
	}
	return out
}

func cloneGroups(groups []Group) []Group {
	out := make([]Group, 0, len(groups))
	for _, group := range groups {
		group.UserIDs = append([]string{}, group.UserIDs...)
		out = append(out, group)
	}
	return out
}

func cloneGrants(grants []SpaceGrant) []SpaceGrant {
	return append([]SpaceGrant{}, grants...)
}

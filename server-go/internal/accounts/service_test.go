package accounts

import (
	"context"
	"testing"
)

func TestServiceCreatesUsersGroupsAndSpaceGrants(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	user, err := service.CreateUser(ctx, CreateUserRequest{
		Username:    "lin",
		DisplayName: "林同学",
		Password:    "Passw0rd!",
		Role:        RoleUser,
		QuotaBytes:  50 << 30,
		Groups:      []string{"family"},
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == "" || user.Username != "lin" || user.Status != StatusActive || user.QuotaBytes != 50<<30 {
		t.Fatalf("unexpected user: %#v", user)
	}

	group, err := service.CreateGroup(ctx, CreateGroupRequest{Name: "项目组", Description: "项目资料协作"})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	group, err = service.UpdateGroupMembers(ctx, group.ID, UpdateGroupMembersRequest{UserIDs: []string{user.ID}})
	if err != nil {
		t.Fatalf("update group members: %v", err)
	}
	if len(group.UserIDs) != 1 || group.UserIDs[0] != user.ID {
		t.Fatalf("unexpected group members: %#v", group)
	}

	grant, err := service.GrantSpace(ctx, SpaceGrantRequest{
		SubjectID:   group.ID,
		SubjectType: SubjectGroup,
		SpaceID:     "space-family",
		Access:      AccessReadWrite,
		QuotaBytes:  100 << 30,
	})
	if err != nil {
		t.Fatalf("grant space: %v", err)
	}
	if grant.ID == "" || grant.SubjectID != group.ID || grant.Access != AccessReadWrite {
		t.Fatalf("unexpected grant: %#v", grant)
	}

	summary, err := service.Summary(ctx)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if len(summary.Users) < 2 || len(summary.Groups) < 2 || len(summary.Grants) != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestServiceValidatesPasswordAndUniqueUsername(t *testing.T) {
	service := NewService()

	// Complexity rules are intentionally removed — a short password is accepted.
	if _, err := service.CreateUser(context.Background(), CreateUserRequest{
		Username:    "weak",
		DisplayName: "Weak",
		Password:    "123qwe",
	}); err != nil {
		t.Fatalf("simple password should be accepted: %v", err)
	}

	// An empty password is still rejected.
	if _, err := service.CreateUser(context.Background(), CreateUserRequest{
		Username:    "blank",
		DisplayName: "Blank",
		Password:    "",
	}); err == nil {
		t.Fatal("expected empty password to be rejected")
	}

	if _, err := service.CreateUser(context.Background(), CreateUserRequest{
		Username:    "admin",
		DisplayName: "Duplicate",
		Password:    "Passw0rd!",
	}); err == nil {
		t.Fatal("expected duplicate username to be rejected")
	}
}

func TestServiceDisablesAndDeletesUser(t *testing.T) {
	service := NewService()
	user, err := service.CreateUser(context.Background(), CreateUserRequest{
		Username:    "temp",
		DisplayName: "临时用户",
		Password:    "Passw0rd!",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	disabled, err := service.UpdateUser(context.Background(), user.ID, UpdateUserRequest{Status: StatusDisabled})
	if err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if disabled.Status != StatusDisabled {
		t.Fatalf("expected disabled user: %#v", disabled)
	}

	if err := service.DeleteUser(context.Background(), user.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	summary, _ := service.Summary(context.Background())
	for _, candidate := range summary.Users {
		if candidate.ID == user.ID {
			t.Fatalf("deleted user still listed: %#v", candidate)
		}
	}
}

func TestServiceDeletesSpaceGrants(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	if _, err := service.GrantSpace(ctx, SpaceGrantRequest{
		SubjectID:   "admin",
		SubjectType: SubjectUser,
		SpaceID:     "space-temp",
		Access:      AccessManage,
	}); err != nil {
		t.Fatalf("grant space: %v", err)
	}

	removed, err := service.DeleteSpaceGrants(ctx, "space-temp")
	if err != nil {
		t.Fatalf("delete space grants: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected one removed grant, got %d", removed)
	}
	summary, _ := service.Summary(ctx)
	if len(summary.Grants) != 0 {
		t.Fatalf("expected grants to be removed, got %#v", summary.Grants)
	}
}

func TestServiceDeletesGrantByID(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	grant, err := service.GrantSpace(ctx, SpaceGrantRequest{
		SubjectID:   "admin",
		SubjectType: SubjectUser,
		SpaceID:     "space-temp",
		Access:      AccessManage,
	})
	if err != nil {
		t.Fatalf("grant space: %v", err)
	}
	if err := service.DeleteGrant(ctx, grant.ID); err != nil {
		t.Fatalf("delete grant: %v", err)
	}
	summary, _ := service.Summary(ctx)
	if len(summary.Grants) != 0 {
		t.Fatalf("expected grant deletion, got %#v", summary.Grants)
	}
}

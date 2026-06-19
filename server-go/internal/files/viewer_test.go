package files

import "testing"

// sampleTree mimics a scanned NAS root: homes/ (alice, bob), groups/
// (group-001-team), and shared spaces team-space (display 团队空间) + home-space.
func sampleTree() FileNode {
	return FileNode{
		Name: "HiGoNAS", Path: "/", IsDir: true,
		Children: []FileNode{
			{Name: "homes", Space: "homes", Path: "/homes", IsDir: true, Children: []FileNode{
				{Name: "alice", Space: "homes", Path: "/homes/alice", IsDir: true},
				{Name: "bob", Space: "homes", Path: "/homes/bob", IsDir: true},
			}},
			{Name: "groups", Space: "groups", Path: "/groups", IsDir: true, Children: []FileNode{
				{Name: "group-001-team", Space: "groups", Path: "/groups/group-001-team", IsDir: true},
				{Name: "group-002-other", Space: "groups", Path: "/groups/group-002-other", IsDir: true},
			}},
			{Name: "团队空间", Space: "团队空间", Path: "/团队空间", IsDir: true},
			{Name: "家庭空间", Space: "家庭空间", Path: "/家庭空间", IsDir: true},
		},
	}
}

func childNames(n FileNode) map[string]bool {
	out := map[string]bool{}
	for _, c := range n.Children {
		out[c.Name] = true
	}
	return out
}

func TestScopeTreeNonAdmin(t *testing.T) {
	v := Viewer{
		Username:      "alice",
		GroupDirs:     []string{"group-001-team"},
		GrantedSpaces: []string{"team-space"}, // dir name; display 团队空间
	}
	scoped := scopeTree(sampleTree(), v)
	names := childNames(scoped)

	if !names["我的文件"] {
		t.Fatal("alice should see her personal folder as 我的文件")
	}
	if !names["group-001-team"] {
		t.Fatal("alice should see her group folder")
	}
	if !names["团队空间"] {
		t.Fatal("alice should see the granted shared space (by dir→display)")
	}
	if names["家庭空间"] {
		t.Fatal("alice must NOT see an ungranted shared space")
	}
	if names["group-002-other"] || names["homes"] || names["groups"] {
		t.Fatalf("alice must not see other groups or the raw homes/groups aggregates: %v", names)
	}
}

func TestCanAccessIsolation(t *testing.T) {
	s := &Service{}
	alice := Viewer{Username: "alice", GroupDirs: []string{"group-001-team"}, GrantedSpaces: []string{"team-space"}}

	cases := []struct {
		path string
		want bool
	}{
		{"/homes/alice/notes.txt", true},
		{"/homes/bob/secret.txt", false},
		{"/groups/group-001-team/shared.md", true},
		{"/groups/group-002-other/x.md", false},
		{"/团队空间/contract.pdf", true},  // granted (team-space → 团队空间)
		{"/家庭空间/photo.jpg", false},    // not granted
	}
	for _, tc := range cases {
		got := s.CanAccess(FileNode{Path: tc.path}, alice)
		if got != tc.want {
			t.Errorf("CanAccess(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}

	// Admin sees everything.
	admin := Viewer{Admin: true}
	if !s.CanAccess(FileNode{Path: "/homes/bob/secret.txt"}, admin) {
		t.Fatal("admin should access any file")
	}
}

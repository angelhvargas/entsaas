package auth

import "testing"

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{"owner", true},
		{"admin", true},
		{"member", true},
		{"viewer", true},
		{"superadmin", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := IsValidRole(tt.role); got != tt.valid {
			t.Errorf("IsValidRole(%q) = %v, want %v", tt.role, got, tt.valid)
		}
	}
}

func TestRoleRank(t *testing.T) {
	if RoleRank("owner") <= RoleRank("admin") {
		t.Error("owner should outrank admin")
	}
	if RoleRank("admin") <= RoleRank("member") {
		t.Error("admin should outrank member")
	}
	if RoleRank("member") <= RoleRank("viewer") {
		t.Error("member should outrank viewer")
	}
	if RoleRank("viewer") <= RoleRank("unknown") {
		t.Error("viewer should outrank unknown")
	}
}

func TestCanManageRole(t *testing.T) {
	tests := []struct {
		actor  string
		target string
		can    bool
	}{
		{"owner", "admin", true},
		{"owner", "member", true},
		{"admin", "member", true},
		{"admin", "owner", false},
		{"member", "member", false},
		{"viewer", "member", false},
	}

	for _, tt := range tests {
		if got := CanManageRole(tt.actor, tt.target); got != tt.can {
			t.Errorf("CanManageRole(%q, %q) = %v, want %v", tt.actor, tt.target, got, tt.can)
		}
	}
}

// Package auth provides role-based access control definitions.
package auth

// Role constants define the authorization hierarchy.
const (
	RoleOwner  = "owner"  // Full org control, billing, deletion
	RoleAdmin  = "admin"  // User management, settings
	RoleMember = "member" // Read/write access to projects
	RoleViewer = "viewer" // Read-only access
)

// ValidRoles is the set of all valid roles.
var ValidRoles = map[string]bool{
	RoleOwner:  true,
	RoleAdmin:  true,
	RoleMember: true,
	RoleViewer: true,
}

// IsValidRole checks if a role string is recognized.
func IsValidRole(role string) bool {
	return ValidRoles[role]
}

// RoleRank returns a numeric rank for a role (higher = more privileged).
func RoleRank(role string) int {
	switch role {
	case RoleOwner:
		return 40
	case RoleAdmin:
		return 30
	case RoleMember:
		return 20
	case RoleViewer:
		return 10
	default:
		return 0
	}
}

// CanManageRole returns true if the actor's role is sufficient to manage
// users with the target role (must be strictly higher).
func CanManageRole(actorRole, targetRole string) bool {
	return RoleRank(actorRole) > RoleRank(targetRole)
}

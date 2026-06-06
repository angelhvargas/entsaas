package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ── Sentinel errors ─────────────────────────────────────────────────────────

var (
	ErrDuplicateEmail   = errors.New("a user with this email already exists")
	ErrDuplicateSlug    = errors.New("an organization with this slug already exists")
	ErrDuplicateName    = errors.New("a resource with this name already exists")
	ErrNotFound         = errors.New("resource not found")
	ErrInvalidTransition = errors.New("invalid state transition")
)

// AppStore defines the authoritative persistence interface for the application.
// Implemented by *PostgresStore.
type AppStore interface {
	// ── Organizations ────────────────────────────────────────────────────
	CreateOrganization(ctx context.Context, name, slug string) (*Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
	GetOrganizationByID(ctx context.Context, id string) (*Organization, error)
	ListOrganizationsByEmail(ctx context.Context, email string) ([]*Organization, error)
	SuspendOrganization(ctx context.Context, id string, reason string) error
	UnsuspendOrganization(ctx context.Context, id string) error
	ScheduleOrganizationDeletion(ctx context.Context, id string, deleteAt time.Time) error
	PurgeOrganizationData(ctx context.Context, id string) error

	// ── Users ────────────────────────────────────────────────────────────
	CreateUser(ctx context.Context, email, role, orgID string) (*User, error)
	GetUserByEmail(ctx context.Context, orgID, email string) (*User, error)
	GetUsersByEmail(ctx context.Context, email string) ([]*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUsersByOrg(ctx context.Context, orgID string) ([]*User, error)
	UpdateUserRole(ctx context.Context, userID, orgID, role string) error
	UpdateUserStatus(ctx context.Context, userID, orgID string, isActive bool) error
	CountOrgOwners(ctx context.Context, orgID string) (int, error)

	// ── Credentials ──────────────────────────────────────────────────────
	CreateUserCredential(ctx context.Context, userID, passwordHash string) error
	GetUserCredential(ctx context.Context, userID string) (*UserCredential, error)
	UpdateUserCredential(ctx context.Context, userID, passwordHash string) error

	// ── Refresh Tokens ───────────────────────────────────────────────────
	CreateRefreshToken(ctx context.Context, tokenHash, userID, orgID, userAgent, ipAddress string, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID string) error

	// ── Password Reset ───────────────────────────────────────────────────
	CreateResetToken(ctx context.Context, tokenHash, userID string) error
	ConsumeResetToken(ctx context.Context, tokenHash string) (*ResetToken, error)
	ValidateResetToken(ctx context.Context, tokenHash string) error

	// ── Email Verification ───────────────────────────────────────────────
	CreateVerificationToken(ctx context.Context, tokenHash, userID string) error
	GetVerificationTokenByHash(ctx context.Context, tokenHash string) (*VerificationToken, error)
	MarkVerificationTokenUsed(ctx context.Context, tokenHash string) error
	MarkUserEmailVerified(ctx context.Context, userID string) error
	IsEmailVerified(ctx context.Context, userID string) (bool, error)

	// ── Invites ──────────────────────────────────────────────────────────
	CreateInvite(ctx context.Context, orgID, email, role, tokenHash string, expiresAt time.Time, createdBy string) (*Invite, error)
	GetInviteByTokenHash(ctx context.Context, tokenHash string) (*Invite, error)
	GetInvitesByOrg(ctx context.Context, orgID string) ([]*Invite, error)
	DeleteInvite(ctx context.Context, id, orgID string) error
	ConsumeInvite(ctx context.Context, tokenHash string) (*Invite, error) // marks as used (deletes row) + returns invite

	// ── Projects (multi-tenant workspaces) ───────────────────────────────
	CreateProject(ctx context.Context, name, orgID string) (*Project, error)
	GetProjectByID(ctx context.Context, projectID string) (*Project, error)
	ListProjectsByOrg(ctx context.Context, orgID string) ([]*Project, error)
	UpdateProjectName(ctx context.Context, projectID, orgID, newName string) error
	DeleteProject(ctx context.Context, projectID, orgID string) error

	// ── Audit Log ────────────────────────────────────────────────────────
	LogAuditEvent(ctx context.Context, actorID *string, orgID, action, entityType, entityID string, metadata map[string]any) error
	GetAuditLogs(ctx context.Context, orgID string, limit int, cursor string) ([]*AuditLogEntry, bool, error)

	// ── Preferences ──────────────────────────────────────────────────────
	GetUserPreferences(ctx context.Context, userID string) (map[string]any, error)
	SetUserPreferences(ctx context.Context, userID string, prefs map[string]any) error

	// ── Billing & Entitlements ───────────────────────────────────────────
	GetEffectiveEntitlements(ctx context.Context, orgID string) (string, map[string]any, error)
	SyncBillingPlan(ctx context.Context, slug, name, description string, entitlements map[string]any, isSelfServe bool) error
	CountProjects(ctx context.Context, orgID string) (int, error)
	CountUsers(ctx context.Context, orgID string) (int, error)

	// ── Subscriptions ───────────────────────────────────────────────────
	CreateSubscription(ctx context.Context, sub *Subscription) error
	UpdateSubscription(ctx context.Context, sub *Subscription) error
	UpsertSubscription(ctx context.Context, sub *Subscription) error
	GetSubscriptionByOrgID(ctx context.Context, orgID string) (*Subscription, error)
	GetSubscriptionByProviderID(ctx context.Context, providerSubID string) (*Subscription, error)
	CancelSubscription(ctx context.Context, orgID string, cancelAtPeriodEnd bool) error

	// ── Lifecycle ────────────────────────────────────────────────────────
	Close()
}

// ── Domain Types ────────────────────────────────────────────────────────────

type Organization struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Slug                string     `json:"slug"`
	CreatedAt           time.Time  `json:"created_at"`
	SuspendedAt         *time.Time `json:"suspended_at,omitempty"`
	SuspendedReason     *string    `json:"suspended_reason,omitempty"`
	DeletionScheduledAt *time.Time `json:"deletion_scheduled_at,omitempty"`
}

type User struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	OrgID           string    `json:"org_id"`
	IsActive        bool      `json:"is_active"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type UserCredential struct {
	UserID       string    `json:"user_id"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RefreshToken struct {
	TokenHash string    `json:"token_hash"`
	UserID    string    `json:"user_id"`
	OrgID     string    `json:"org_id"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	ExpiresAt time.Time `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ResetToken struct {
	TokenHash string    `json:"token_hash"`
	UserID    string    `json:"user_id"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type VerificationToken struct {
	TokenHash string    `json:"token_hash"`
	UserID    string    `json:"user_id"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Invite struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OrgID     string    `json:"org_id"`
	Status    string    `json:"status"` // active, paused, archived
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuditLogEntry struct {
	ID         uuid.UUID      `json:"id"`
	ActorID    *string        `json:"actor_id,omitempty"`
	OrgID      string         `json:"org_id"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Subscription struct {
	ID                     string     `json:"id"`
	OrgID                  string     `json:"org_id"`
	PlanSlug               string     `json:"plan_slug"`
	PlanID                 string     `json:"plan_id"` // internal UUID
	PlanVersionID          string     `json:"plan_version_id,omitempty"`
	Status                 string     `json:"status"` // active, trialing, past_due, canceled, paused
	ProviderSubscriptionID string     `json:"provider_subscription_id,omitempty"`
	ProviderCustomerID     string     `json:"provider_customer_id,omitempty"`
	CurrentPeriodStart     *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd       *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd      bool       `json:"cancel_at_period_end"`
	TrialEndAt             *time.Time `json:"trial_end_at,omitempty"`
	CanceledAt             *time.Time `json:"canceled_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

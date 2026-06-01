package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// PostgresStore implements the AppStore interface using pgx connection pool.
type PostgresStore struct {
	pool      *pgxpool.Pool
	masterKey []byte
	keyVer    int
}

// NewPostgresStore connects to Postgres and returns a new store.
func NewPostgresStore(dsn string, masterKey []byte, keyVersion int, maxConns, minConns int32) (*PostgresStore, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres DSN: %w", err)
	}

	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	if minConns > 0 {
		cfg.MinConns = minConns
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	log.Info().Str("host", cfg.ConnConfig.Host).Int32("max_conns", cfg.MaxConns).Msg("Postgres connected")
	return &PostgresStore{pool: pool, masterKey: masterKey, keyVer: keyVersion}, nil
}

// Pool returns the underlying pgxpool for direct access when needed.
func (s *PostgresStore) Pool() *pgxpool.Pool { return s.pool }

// Close closes the connection pool.
func (s *PostgresStore) Close() { s.pool.Close() }

// ── Organizations ───────────────────────────────────────────────────────────

func (s *PostgresStore) CreateOrganization(ctx context.Context, name, slug string) (*Organization, error) {
	org := &Organization{
		ID:        uuid.New().String(),
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
	}

	// SEC-17: Retry with random suffix on slug conflict instead of silently
	// swallowing the insert. This prevents data loss when two users register
	// with the same org name.
	for attempts := 0; attempts < 3; attempts++ {
		tag, err := s.pool.Exec(ctx,
			`INSERT INTO organizations (id, name, slug, created_at) VALUES ($1, $2, $3, $4)
			 ON CONFLICT (slug) DO NOTHING`,
			org.ID, org.Name, org.Slug, org.CreatedAt)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() > 0 {
			return org, nil
		}
		// Slug collision — append a random suffix and retry.
		suffix := uuid.New().String()[:6]
		org.Slug = slug + "-" + suffix
		org.ID = uuid.New().String()
	}

	return nil, fmt.Errorf("failed to create organization: slug %q still conflicts after retries", slug)
}

func (s *PostgresStore) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	org := &Organization{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, created_at, suspended_at, suspended_reason, deletion_scheduled_at 
		 FROM organizations WHERE slug = $1`, slug).
		Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.SuspendedAt, &org.SuspendedReason, &org.DeletionScheduledAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return org, err
}

func (s *PostgresStore) GetOrganizationByID(ctx context.Context, id string) (*Organization, error) {
	org := &Organization{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, created_at, suspended_at, suspended_reason, deletion_scheduled_at 
		 FROM organizations WHERE id = $1`, id).
		Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.SuspendedAt, &org.SuspendedReason, &org.DeletionScheduledAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return org, err
}

func (s *PostgresStore) ListOrganizationsByEmail(ctx context.Context, email string) ([]*Organization, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT o.id, o.name, o.slug, o.created_at, o.suspended_at, o.suspended_reason, o.deletion_scheduled_at
		 FROM organizations o
		 JOIN users u ON u.org_id = o.id
		 WHERE u.email = $1 AND u.is_active = true
		 ORDER BY o.name`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		o := &Organization{}
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.CreatedAt, &o.SuspendedAt, &o.SuspendedReason, &o.DeletionScheduledAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

func (s *PostgresStore) SuspendOrganization(ctx context.Context, id string, reason string) error {
	res, err := s.pool.Exec(ctx,
		`UPDATE organizations SET suspended_at = NOW(), suspended_reason = $2 WHERE id = $1`,
		id, reason)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) UnsuspendOrganization(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx,
		`UPDATE organizations SET suspended_at = NULL, suspended_reason = NULL WHERE id = $1`,
		id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ScheduleOrganizationDeletion(ctx context.Context, id string, deleteAt time.Time) error {
	res, err := s.pool.Exec(ctx,
		`UPDATE organizations SET deletion_scheduled_at = $2 WHERE id = $1`,
		id, deleteAt)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) PurgeOrganizationData(ctx context.Context, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM reset_tokens WHERE user_id IN (SELECT id FROM users WHERE org_id = $1)`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM verification_tokens WHERE user_id IN (SELECT id FROM users WHERE org_id = $1)`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM user_credentials WHERE user_id IN (SELECT id FROM users WHERE org_id = $1)`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM user_preferences WHERE user_id IN (SELECT id FROM users WHERE org_id = $1)`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE org_id = $1`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM invites WHERE org_id = $1`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM projects WHERE org_id = $1`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM users WHERE org_id = $1`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM audit_log WHERE org_id = $1`, id)
	if err != nil {
		return err
	}

	res, err := tx.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}

	return tx.Commit(ctx)
}

// ── Users ───────────────────────────────────────────────────────────────────

func (s *PostgresStore) CreateUser(ctx context.Context, email, role, orgID string) (*User, error) {
	u := &User{
		ID:        uuid.New().String(),
		Email:     email,
		Role:      role,
		OrgID:     orgID,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO users (id, email, role, org_id, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		u.ID, u.Email, u.Role, u.OrgID, u.IsActive, u.CreatedAt)
	return u, err
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, orgID, email string) (*User, error) {
	u := &User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, role, org_id, is_active, email_verified_at, created_at
		 FROM users WHERE org_id = $1 AND email = $2`, orgID, email).
		Scan(&u.ID, &u.Email, &u.Role, &u.OrgID, &u.IsActive, &u.EmailVerifiedAt, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *PostgresStore) GetUsersByEmail(ctx context.Context, email string) ([]*User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, email, role, org_id, is_active, email_verified_at, created_at
		 FROM users WHERE email = $1`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.OrgID, &u.IsActive, &u.EmailVerifiedAt, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, role, org_id, is_active, email_verified_at, created_at
		 FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.Role, &u.OrgID, &u.IsActive, &u.EmailVerifiedAt, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *PostgresStore) GetUsersByOrg(ctx context.Context, orgID string) ([]*User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, email, role, org_id, is_active, email_verified_at, created_at
		 FROM users WHERE org_id = $1 ORDER BY created_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.OrgID, &u.IsActive, &u.EmailVerifiedAt, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *PostgresStore) UpdateUserRole(ctx context.Context, userID, orgID, role string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET role = $1 WHERE id = $2 AND org_id = $3`, role, userID, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) UpdateUserStatus(ctx context.Context, userID, orgID string, isActive bool) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET is_active = $1 WHERE id = $2 AND org_id = $3`, isActive, userID, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) CountOrgOwners(ctx context.Context, orgID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE org_id = $1 AND role = 'owner' AND is_active = true`, orgID).
		Scan(&count)
	return count, err
}

// ── Credentials ─────────────────────────────────────────────────────────────

func (s *PostgresStore) CreateUserCredential(ctx context.Context, userID, passwordHash string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO user_credentials (user_id, password_hash, created_at, updated_at)
		 VALUES ($1, $2, NOW(), NOW())`, userID, passwordHash)
	return err
}

func (s *PostgresStore) GetUserCredential(ctx context.Context, userID string) (*UserCredential, error) {
	c := &UserCredential{}
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, password_hash, created_at, updated_at
		 FROM user_credentials WHERE user_id = $1`, userID).
		Scan(&c.UserID, &c.PasswordHash, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return c, err
}

func (s *PostgresStore) UpdateUserCredential(ctx context.Context, userID, passwordHash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE user_credentials SET password_hash = $1, updated_at = NOW() WHERE user_id = $2`,
		passwordHash, userID)
	return err
}

// ── Refresh Tokens ──────────────────────────────────────────────────────────

func (s *PostgresStore) CreateRefreshToken(ctx context.Context, tokenHash, userID, orgID, userAgent, ipAddress string, ttl time.Duration) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (token_hash, user_id, org_id, user_agent, ip_address, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		tokenHash, userID, orgID, userAgent, ipAddress, time.Now().Add(ttl))
	return err
}

func (s *PostgresStore) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	t := &RefreshToken{}
	err := s.pool.QueryRow(ctx,
		`SELECT token_hash, user_id, org_id, user_agent, ip_address, expires_at, revoked_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`, tokenHash).
		Scan(&t.TokenHash, &t.UserID, &t.OrgID, &t.UserAgent, &t.IPAddress, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *PostgresStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *PostgresStore) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

// ── Password Reset ──────────────────────────────────────────────────────────

func (s *PostgresStore) CreateResetToken(ctx context.Context, tokenHash, userID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO reset_tokens (token_hash, user_id, expires_at, created_at)
		 VALUES ($1, $2, NOW() + INTERVAL '1 hour', NOW())`, tokenHash, userID)
	return err
}

func (s *PostgresStore) ConsumeResetToken(ctx context.Context, tokenHash string) (*ResetToken, error) {
	t := &ResetToken{}
	err := s.pool.QueryRow(ctx,
		`UPDATE reset_tokens SET used_at = NOW()
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		 RETURNING token_hash, user_id, used_at, expires_at, created_at`,
		tokenHash).
		Scan(&t.TokenHash, &t.UserID, &t.UsedAt, &t.ExpiresAt, &t.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *PostgresStore) ValidateResetToken(ctx context.Context, tokenHash string) error {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM reset_tokens WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW())`,
		tokenHash).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

// ── Email Verification ──────────────────────────────────────────────────────

func (s *PostgresStore) CreateVerificationToken(ctx context.Context, tokenHash, userID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO verification_tokens (token_hash, user_id, expires_at, created_at)
		 VALUES ($1, $2, NOW() + INTERVAL '24 hours', NOW())`, tokenHash, userID)
	return err
}

func (s *PostgresStore) GetVerificationTokenByHash(ctx context.Context, tokenHash string) (*VerificationToken, error) {
	t := &VerificationToken{}
	err := s.pool.QueryRow(ctx,
		`SELECT token_hash, user_id, used_at, expires_at, created_at
		 FROM verification_tokens WHERE token_hash = $1`, tokenHash).
		Scan(&t.TokenHash, &t.UserID, &t.UsedAt, &t.ExpiresAt, &t.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *PostgresStore) MarkVerificationTokenUsed(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE verification_tokens SET used_at = NOW() WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *PostgresStore) MarkUserEmailVerified(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET email_verified_at = NOW() WHERE id = $1`, userID)
	return err
}

func (s *PostgresStore) IsEmailVerified(ctx context.Context, userID string) (bool, error) {
	var verified bool
	err := s.pool.QueryRow(ctx,
		`SELECT email_verified_at IS NOT NULL FROM users WHERE id = $1`, userID).Scan(&verified)
	if err == pgx.ErrNoRows {
		return false, ErrNotFound
	}
	return verified, err
}

// ── Invites ─────────────────────────────────────────────────────────────────

func (s *PostgresStore) CreateInvite(ctx context.Context, orgID, email, role, tokenHash string, expiresAt time.Time, createdBy string) (*Invite, error) {
	inv := &Invite{
		ID:        uuid.New().String(),
		OrgID:     orgID,
		Email:     email,
		Role:      role,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO invites (id, org_id, email, role, token_hash, expires_at, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		inv.ID, inv.OrgID, inv.Email, inv.Role, inv.TokenHash, inv.ExpiresAt, inv.CreatedBy, inv.CreatedAt)
	return inv, err
}

func (s *PostgresStore) GetInviteByTokenHash(ctx context.Context, tokenHash string) (*Invite, error) {
	inv := &Invite{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, email, role, token_hash, expires_at, created_by, created_at
		 FROM invites WHERE token_hash = $1`, tokenHash).
		Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.TokenHash, &inv.ExpiresAt, &inv.CreatedBy, &inv.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return inv, err
}

func (s *PostgresStore) GetInvitesByOrg(ctx context.Context, orgID string) ([]*Invite, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, org_id, email, role, token_hash, expires_at, created_by, created_at
		 FROM invites WHERE org_id = $1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []*Invite
	for rows.Next() {
		inv := &Invite{}
		if err := rows.Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.TokenHash, &inv.ExpiresAt, &inv.CreatedBy, &inv.CreatedAt); err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

func (s *PostgresStore) DeleteInvite(ctx context.Context, id, orgID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM invites WHERE id = $1 AND org_id = $2`, id, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ConsumeInvite atomically deletes the invite and returns it.
// Returns ErrNotFound if the token is invalid or already expired.
func (s *PostgresStore) ConsumeInvite(ctx context.Context, tokenHash string) (*Invite, error) {
	inv := &Invite{}
	err := s.pool.QueryRow(ctx,
		`DELETE FROM invites
		 WHERE token_hash = $1 AND expires_at > NOW()
		 RETURNING id, org_id, email, role, token_hash, expires_at, created_by, created_at`,
		tokenHash).
		Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.TokenHash, &inv.ExpiresAt, &inv.CreatedBy, &inv.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return inv, err
}

// ── Projects ────────────────────────────────────────────────────────────────

func (s *PostgresStore) CreateProject(ctx context.Context, name, orgID string) (*Project, error) {
	p := &Project{
		ID:        uuid.New().String(),
		Name:      name,
		OrgID:     orgID,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO projects (id, name, org_id, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.Name, p.OrgID, p.Status, p.CreatedAt, p.UpdatedAt)
	return p, err
}

func (s *PostgresStore) GetProjectByID(ctx context.Context, projectID string) (*Project, error) {
	p := &Project{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, org_id, status, created_at, updated_at
		 FROM projects WHERE id = $1`, projectID).
		Scan(&p.ID, &p.Name, &p.OrgID, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return p, err
}

func (s *PostgresStore) ListProjectsByOrg(ctx context.Context, orgID string) ([]*Project, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, org_id, status, created_at, updated_at
		 FROM projects WHERE org_id = $1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.OrgID, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *PostgresStore) UpdateProjectName(ctx context.Context, projectID, orgID, newName string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE projects SET name = $1, updated_at = NOW() WHERE id = $2 AND org_id = $3`,
		newName, projectID, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteProject(ctx context.Context, projectID, orgID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM projects WHERE id = $1 AND org_id = $2`, projectID, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Audit Log ───────────────────────────────────────────────────────────────

func (s *PostgresStore) LogAuditEvent(ctx context.Context, actorID *string, orgID, action, entityType, entityID string, metadata map[string]any) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO audit_log (id, actor_id, org_id, action, entity_type, entity_id, metadata, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		uuid.New(), actorID, orgID, action, entityType, entityID, metaJSON)
	return err
}

func (s *PostgresStore) GetAuditLogs(ctx context.Context, orgID string, limit int, cursor string) ([]*AuditLogEntry, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows pgx.Rows
	var err error
	if cursor != "" {
		rows, err = s.pool.Query(ctx,
			`SELECT id, actor_id, org_id, action, entity_type, entity_id, metadata, created_at
			 FROM audit_log WHERE org_id = $1 AND id < $2
			 ORDER BY created_at DESC LIMIT $3`,
			orgID, cursor, limit+1)
	} else {
		rows, err = s.pool.Query(ctx,
			`SELECT id, actor_id, org_id, action, entity_type, entity_id, metadata, created_at
			 FROM audit_log WHERE org_id = $1
			 ORDER BY created_at DESC LIMIT $2`,
			orgID, limit+1)
	}
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var entries []*AuditLogEntry
	for rows.Next() {
		e := &AuditLogEntry{}
		var metaJSON []byte
		if err := rows.Scan(&e.ID, &e.ActorID, &e.OrgID, &e.Action, &e.EntityType, &e.EntityID, &metaJSON, &e.CreatedAt); err != nil {
			return nil, false, err
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &e.Metadata)
		}
		entries = append(entries, e)
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}
	return entries, hasMore, rows.Err()
}

// ── Preferences ─────────────────────────────────────────────────────────────

func (s *PostgresStore) GetUserPreferences(ctx context.Context, userID string) (map[string]any, error) {
	var data []byte
	err := s.pool.QueryRow(ctx,
		`SELECT preferences FROM user_preferences WHERE user_id = $1`, userID).Scan(&data)
	if err == pgx.ErrNoRows {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var prefs map[string]any
	return prefs, json.Unmarshal(data, &prefs)
}

func (s *PostgresStore) SetUserPreferences(ctx context.Context, userID string, prefs map[string]any) error {
	data, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO user_preferences (user_id, preferences, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET preferences = $2, updated_at = NOW()`,
		userID, data)
	return err
}

// ── Billing & Entitlements ───────────────────────────────────────────

func (s *PostgresStore) GetEffectiveEntitlements(ctx context.Context, orgID string) (string, map[string]any, error) {
	var slug string
	var entitlementsBytes []byte

	// 1. Try to fetch entitlements from active/trialing subscription
	query := `
		SELECT p.slug, pv.entitlements
		FROM subscriptions s
		JOIN plans p ON s.plan_id = p.id
		JOIN plan_versions pv ON s.plan_version_id = pv.id
		WHERE s.org_id = $1 AND (s.status IN ('active', 'trialing') OR s.current_period_end > NOW())
		LIMIT 1
	`
	err := s.pool.QueryRow(ctx, query, orgID).Scan(&slug, &entitlementsBytes)
	if err != nil {
		if err == pgx.ErrNoRows {
			// 2. Fallback to free plan entitlements
			fallbackQuery := `
				SELECT p.slug, pv.entitlements
				FROM plans p
				JOIN plan_versions pv ON pv.plan_id = p.id
				WHERE p.slug = 'free'
				ORDER BY pv.version DESC
				LIMIT 1
			`
			err = s.pool.QueryRow(ctx, fallbackQuery).Scan(&slug, &entitlementsBytes)
			if err != nil {
				if err == pgx.ErrNoRows {
					return "", nil, ErrNotFound
				}
				return "", nil, err
			}
		} else {
			return "", nil, err
		}
	}

	var ents map[string]any
	if err := json.Unmarshal(entitlementsBytes, &ents); err != nil {
		return "", nil, err
	}

	return slug, ents, nil
}

func (s *PostgresStore) CountProjects(ctx context.Context, orgID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM projects WHERE org_id = $1", orgID).Scan(&count)
	return count, err
}

func (s *PostgresStore) CountUsers(ctx context.Context, orgID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE org_id = $1 AND is_active = true", orgID).Scan(&count)
	return count, err
}

// ── Subscriptions ───────────────────────────────────────────────────

// CreateSubscription upserts a subscription for an org. One active subscription per org.
func (s *PostgresStore) CreateSubscription(ctx context.Context, sub *Subscription) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO subscriptions (
			org_id, plan_id, plan_version_id, status,
			provider_subscription_id, provider_customer_id,
			current_period_start, current_period_end,
			cancel_at_period_end, trial_end_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (org_id) DO UPDATE SET
			plan_id                  = EXCLUDED.plan_id,
			plan_version_id          = EXCLUDED.plan_version_id,
			status                   = EXCLUDED.status,
			provider_subscription_id = EXCLUDED.provider_subscription_id,
			provider_customer_id     = EXCLUDED.provider_customer_id,
			current_period_start     = EXCLUDED.current_period_start,
			current_period_end       = EXCLUDED.current_period_end,
			cancel_at_period_end     = EXCLUDED.cancel_at_period_end,
			trial_end_at             = EXCLUDED.trial_end_at,
			updated_at               = NOW()
	`, sub.OrgID, sub.PlanID, nilIfEmpty(sub.PlanVersionID), sub.Status,
		nilIfEmpty(sub.ProviderSubscriptionID), nilIfEmpty(sub.ProviderCustomerID),
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd, sub.TrialEndAt)
	return err
}

// UpdateSubscription updates mutable fields on an existing subscription, matched by org_id.
func (s *PostgresStore) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE subscriptions SET
			plan_id                  = $2,
			plan_version_id          = $3,
			status                   = $4,
			provider_subscription_id = $5,
			provider_customer_id     = $6,
			current_period_start     = $7,
			current_period_end       = $8,
			cancel_at_period_end     = $9,
			trial_end_at             = $10,
			canceled_at              = $11,
			updated_at               = NOW()
		WHERE org_id = $1
	`, sub.OrgID, sub.PlanID, nilIfEmpty(sub.PlanVersionID), sub.Status,
		nilIfEmpty(sub.ProviderSubscriptionID), nilIfEmpty(sub.ProviderCustomerID),
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd, sub.TrialEndAt, sub.CanceledAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetSubscriptionByOrgID returns the subscription for an org, or ErrNotFound.
func (s *PostgresStore) GetSubscriptionByOrgID(ctx context.Context, orgID string) (*Subscription, error) {
	sub := &Subscription{}
	err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.org_id, p.slug, s.plan_id, COALESCE(s.plan_version_id::text, ''),
			s.status, COALESCE(s.provider_subscription_id, ''), COALESCE(s.provider_customer_id, ''),
			s.current_period_start, s.current_period_end,
			s.cancel_at_period_end, s.trial_end_at, s.canceled_at,
			s.created_at, s.updated_at
		FROM subscriptions s
		JOIN plans p ON s.plan_id = p.id
		WHERE s.org_id = $1
	`, orgID).Scan(
		&sub.ID, &sub.OrgID, &sub.PlanSlug, &sub.PlanID, &sub.PlanVersionID,
		&sub.Status, &sub.ProviderSubscriptionID, &sub.ProviderCustomerID,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.CancelAtPeriodEnd, &sub.TrialEndAt, &sub.CanceledAt,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return sub, nil
}

// GetSubscriptionByProviderID looks up a subscription by the external provider's subscription ID.
// Used by webhook handlers to reconcile incoming events.
func (s *PostgresStore) GetSubscriptionByProviderID(ctx context.Context, providerSubID string) (*Subscription, error) {
	sub := &Subscription{}
	err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.org_id, p.slug, s.plan_id, COALESCE(s.plan_version_id::text, ''),
			s.status, COALESCE(s.provider_subscription_id, ''), COALESCE(s.provider_customer_id, ''),
			s.current_period_start, s.current_period_end,
			s.cancel_at_period_end, s.trial_end_at, s.canceled_at,
			s.created_at, s.updated_at
		FROM subscriptions s
		JOIN plans p ON s.plan_id = p.id
		WHERE s.provider_subscription_id = $1
	`, providerSubID).Scan(
		&sub.ID, &sub.OrgID, &sub.PlanSlug, &sub.PlanID, &sub.PlanVersionID,
		&sub.Status, &sub.ProviderSubscriptionID, &sub.ProviderCustomerID,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.CancelAtPeriodEnd, &sub.TrialEndAt, &sub.CanceledAt,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return sub, nil
}

// CancelSubscription marks a subscription as canceling or immediately canceled.
func (s *PostgresStore) CancelSubscription(ctx context.Context, orgID string, cancelAtPeriodEnd bool) error {
	if cancelAtPeriodEnd {
		tag, err := s.pool.Exec(ctx, `
			UPDATE subscriptions SET cancel_at_period_end = true, updated_at = NOW()
			WHERE org_id = $1 AND status IN ('active', 'trialing')
		`, orgID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}
	// Immediate cancellation.
	tag, err := s.pool.Exec(ctx, `
		UPDATE subscriptions SET status = 'canceled', canceled_at = NOW(), updated_at = NOW()
		WHERE org_id = $1 AND status IN ('active', 'trialing', 'past_due')
	`, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// nilIfEmpty returns nil for empty strings (used for nullable columns).
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}


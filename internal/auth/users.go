package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Role defines what a user is allowed to do.
type Role string

const (
	RoleAdmin    Role = "admin"    // full access including destructive operations
	RoleOperator Role = "operator" // operational access, no structural deletes
	RoleViewer   Role = "viewer"   // read-only
)

// IsValid reports whether r is a known role.
func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleOperator || r == RoleViewer
}

// AtLeast reports whether r has at least the permissions of minimum.
// Order: admin > operator > viewer
func (r Role) AtLeast(minimum Role) bool {
	rank := map[Role]int{RoleAdmin: 3, RoleOperator: 2, RoleViewer: 1}
	return rank[r] >= rank[minimum]
}

// User represents an authenticated human user of the dashboard.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string // never expose in API responses
	Role         Role
	DisplayName  string
	DisabledAt   *time.Time // nil = active
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsActive reports whether the user account is not disabled.
func (u *User) IsActive() bool { return u.DisabledAt == nil }

// ErrUserNotFound is returned when no user matches the query.
var ErrUserNotFound = errors.New("auth: user not found")

// ErrEmailTaken is returned when creating a user with a duplicate email.
var ErrEmailTaken = errors.New("auth: email already registered")

// UserStore manages user persistence.
type UserStore interface {
	Create(ctx context.Context, email, passwordHash string, role Role, displayName string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context) ([]User, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role Role) error
	Disable(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	// CountByRole returns how many active users have the given role.
	// Used to prevent deleting the last admin.
	CountByRole(ctx context.Context, role Role) (int, error)
}

type pgUserStore struct{ pool *pgxpool.Pool }

// NewUserStore returns a PostgreSQL-backed UserStore.
func NewUserStore(pool *pgxpool.Pool) UserStore {
	return &pgUserStore{pool: pool}
}

func (s *pgUserStore) Create(ctx context.Context, email, passwordHash string, role Role, displayName string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role, display_name)
		VALUES ($1,$2,$3,$4)
		RETURNING id, email, password_hash, role, display_name, disabled_at, created_at, updated_at`,
		email, passwordHash, string(role), displayName,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, (*string)(&u.Role),
		&u.DisplayName, &u.DisabledAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return &u, nil
}

func (s *pgUserStore) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role, display_name, disabled_at, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, (*string)(&u.Role),
		&u.DisplayName, &u.DisabledAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &u, err
}

func (s *pgUserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role, display_name, disabled_at, created_at, updated_at
		FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, (*string)(&u.Role),
		&u.DisplayName, &u.DisabledAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &u, err
}

func (s *pgUserStore) List(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, email, password_hash, role, display_name, disabled_at, created_at, updated_at
		FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, (*string)(&u.Role),
			&u.DisplayName, &u.DisabledAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *pgUserStore) UpdateRole(ctx context.Context, id uuid.UUID, role Role) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET role=$1, updated_at=NOW() WHERE id=$2`, string(role), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *pgUserStore) Disable(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET disabled_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *pgUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *pgUserStore) CountByRole(ctx context.Context, role Role) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE role=$1 AND disabled_at IS NULL`, string(role),
	).Scan(&count)
	return count, err
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// VerifyPassword returns true if plain matches the bcrypt hash.
func VerifyPassword(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// isUniqueViolation detects PostgreSQL duplicate-key errors.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, &pgUniqueError{}) ||
		contains(err.Error(), "unique") || contains(err.Error(), "23505")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

type pgUniqueError struct{}

func (*pgUniqueError) Error() string { return "unique" }
func (e *pgUniqueError) Is(target error) bool {
	if target == nil {
		return false
	}
	return target.Error() == "unique"
}

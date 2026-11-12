// Package auth handles registration, login, logout and session lookup.
// User profiles themselves belong to the users package; auth uses it.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

type Repository struct {
	q *queries.Queries
}

func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

type credentials struct {
	UserID       uuid.UUID
	PasswordHash string
	Status       string
}

// Credentials looks up the login data for an email (case-insensitive).
func (r *Repository) Credentials(ctx context.Context, email string) (credentials, bool, error) {
	row, err := r.q.GetUserCredentialsByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return credentials{}, false, nil
	}
	if err != nil {
		return credentials{}, false, fmt.Errorf("get credentials: %w", err)
	}
	return credentials{UserID: row.ID, PasswordHash: row.PasswordHash, Status: row.Status}, true, nil
}

func (r *Repository) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) error {
	err := r.q.CreateSession(ctx, queries.CreateSessionParams{UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// ActiveSession finds an unexpired session of an active user.
func (r *Repository) ActiveSession(ctx context.Context, tokenHash []byte) (sessionID, userID uuid.UUID, ok bool, err error) {
	row, err := r.q.GetActiveSession(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("get session: %w", err)
	}
	return row.SessionID, row.UserID, true, nil
}

func (r *Repository) DeleteSession(ctx context.Context, tokenHash []byte) error {
	if err := r.q.DeleteSessionByTokenHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *Repository) DeleteExpiredSessions(ctx context.Context, userID uuid.UUID) error {
	if err := r.q.DeleteExpiredSessionsForUser(ctx, userID); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

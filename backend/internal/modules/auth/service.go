package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/users"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/password"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/token"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

var (
	// One error for "no such email" AND "wrong password": telling them apart
	// would let anyone test which emails have accounts.
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountSuspended   = errors.New("account suspended")
)

const (
	minPasswordLength = 8
	maxPasswordLength = 128
)

type Config struct {
	SessionTTL time.Duration
	Password   password.Params
}

type Service struct {
	pool *pgxpool.Pool
	cfg  Config
	// dummyHash is verified when an email is unknown, so "no such user"
	// takes as long as "wrong password" and timing does not leak accounts.
	dummyHash string
	now       func() time.Time
}

func NewService(pool *pgxpool.Pool, cfg Config) (*Service, error) {
	dummy, err := password.Hash("timing-equalizer-not-a-real-password", cfg.Password)
	if err != nil {
		return nil, err
	}
	return &Service{pool: pool, cfg: cfg, dummyHash: dummy, now: time.Now}, nil
}

// Session is what the handler puts in the cookie. Token is the raw secret;
// it exists only here and in the browser, never in the database or logs.
type Session struct {
	Token     string
	ExpiresAt time.Time
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// Register creates a customer account and logs it in.
//
// One transaction covers user + role + session. Without it, a failure after
// inserting the user would leave an account with no role: it could never be
// authorized, and its email could never be registered again.
func (s *Service) Register(ctx context.Context, in RegisterInput) (users.User, Session, error) {
	return s.createAccount(ctx, in, authctx.RoleCustomer, true)
}

// CreateUserWithRole creates an account without logging in. Used by the
// seed-admin command; goes through exactly the same validation as sign-up.
func (s *Service) CreateUserWithRole(ctx context.Context, in RegisterInput, role string) (users.User, error) {
	u, _, err := s.createAccount(ctx, in, role, false)
	return u, err
}

func (s *Service) createAccount(ctx context.Context, in RegisterInput, role string, withSession bool) (users.User, Session, error) {
	in, err := validateRegister(in)
	if err != nil {
		return users.User{}, Session{}, err
	}

	// Hash BEFORE opening the transaction: hashing is deliberately slow and
	// must not hold a database connection while it runs.
	hash, err := password.Hash(in.Password, s.cfg.Password)
	if err != nil {
		return users.User{}, Session{}, err
	}

	var (
		user users.User
		sess Session
	)
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		userRepo := users.NewRepository(tx)

		u, err := userRepo.Create(ctx, in.Email, hash, in.Name)
		if err != nil {
			return err // users.ErrEmailTaken is passed through to the handler
		}
		if err := userRepo.AssignRole(ctx, u.ID, role); err != nil {
			return err
		}
		u.Roles = []string{role}
		user = u

		if withSession {
			sess, err = s.createSession(ctx, NewRepository(tx), u.ID)
			return err
		}
		return nil
	})
	if err != nil {
		return users.User{}, Session{}, err
	}
	return user, sess, nil
}

// Login checks the credentials and starts a new session.
// previousToken is the cookie the browser already had, if any: it is deleted
// so logging in always yields a fresh token (session rotation). That defeats
// "session fixation", where an attacker plants a known token before you log in.
func (s *Service) Login(ctx context.Context, email, plain, previousToken string) (users.User, Session, error) {
	email = users.NormalizeEmail(email)
	errs := validation.Errors{}
	if email == "" {
		errs.Add("email", "is required")
	}
	if plain == "" {
		errs.Add("password", "is required")
	}
	if err := errs.Err(); err != nil {
		return users.User{}, Session{}, err
	}
	if len(plain) > 4*maxPasswordLength { // cannot be a valid password; skip hashing it
		return users.User{}, Session{}, ErrInvalidCredentials
	}

	repo := NewRepository(s.pool)
	cred, found, err := repo.Credentials(ctx, email)
	if err != nil {
		return users.User{}, Session{}, err
	}
	if !found {
		_, _ = password.Verify(plain, s.dummyHash)
		return users.User{}, Session{}, ErrInvalidCredentials
	}

	ok, err := password.Verify(plain, cred.PasswordHash)
	if err != nil {
		return users.User{}, Session{}, fmt.Errorf("verify password for user %s: %w", cred.UserID, err)
	}
	if !ok {
		return users.User{}, Session{}, ErrInvalidCredentials
	}
	// Checked only after the password is correct, so this answer cannot be
	// used to discover suspended accounts.
	if cred.Status != users.StatusActive {
		return users.User{}, Session{}, ErrAccountSuspended
	}

	if previousToken != "" {
		if err := repo.DeleteSession(ctx, token.Hash(previousToken)); err != nil {
			return users.User{}, Session{}, err
		}
	}
	if err := repo.DeleteExpiredSessions(ctx, cred.UserID); err != nil {
		return users.User{}, Session{}, err
	}
	sess, err := s.createSession(ctx, repo, cred.UserID)
	if err != nil {
		return users.User{}, Session{}, err
	}

	userRepo := users.NewRepository(s.pool)
	u, err := userRepo.GetByID(ctx, cred.UserID)
	if err != nil {
		return users.User{}, Session{}, err
	}
	if u.Roles, err = userRepo.RoleNames(ctx, u.ID); err != nil {
		return users.User{}, Session{}, err
	}
	return u, sess, nil
}

// Logout deletes the session server-side. Clearing the cookie alone is not
// enough: a copied token would keep working until it expired.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return NewRepository(s.pool).DeleteSession(ctx, token.Hash(rawToken))
}

// ResolveSession is used by the Authenticate middleware on every request.
// Roles are read fresh each time, so a role change applies immediately.
func (s *Service) ResolveSession(ctx context.Context, rawToken string) (authctx.Principal, bool, error) {
	sessionID, userID, ok, err := NewRepository(s.pool).ActiveSession(ctx, token.Hash(rawToken))
	if err != nil || !ok {
		return authctx.Principal{}, false, err
	}
	roles, err := users.NewRepository(s.pool).RoleNames(ctx, userID)
	if err != nil {
		return authctx.Principal{}, false, err
	}
	return authctx.Principal{UserID: userID, SessionID: sessionID, Roles: roles}, true, nil
}

func (s *Service) createSession(ctx context.Context, repo *Repository, userID uuid.UUID) (Session, error) {
	raw, hash, err := token.New()
	if err != nil {
		return Session{}, err
	}
	expiresAt := s.now().Add(s.cfg.SessionTTL).UTC()
	if err := repo.CreateSession(ctx, userID, hash, expiresAt); err != nil {
		return Session{}, err
	}
	return Session{Token: raw, ExpiresAt: expiresAt}, nil
}

func validateRegister(in RegisterInput) (RegisterInput, error) {
	errs := validation.Errors{}
	in.Name = users.CleanName(errs, "name", in.Name)
	in.Email = users.NormalizeEmail(in.Email)
	users.ValidateEmail(errs, "email", in.Email)

	// Length in characters, not bytes, so non-ASCII passwords are not penalised.
	// No other composition rules: length matters most (NIST SP 800-63B).
	switch n := utf8.RuneCountInString(in.Password); {
	case n < minPasswordLength:
		errs.Add("password", fmt.Sprintf("must be at least %d characters", minPasswordLength))
	case n > maxPasswordLength:
		errs.Add("password", fmt.Sprintf("must be at most %d characters", maxPasswordLength))
	}
	return in, errs.Err()
}

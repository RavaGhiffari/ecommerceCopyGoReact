package users

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// Service holds the rules for reading and changing a user's own profile.
// It takes plain Go values (no *gin.Context), so it can be called from HTTP,
// a CLI, or a test the same way.
type Service struct {
	repo *Repository
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{repo: NewRepository(pool)}
}

// GetProfile returns the user with their roles.
func (s *Service) GetProfile(ctx context.Context, id uuid.UUID) (User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	if u.Roles, err = s.repo.RoleNames(ctx, id); err != nil {
		return User{}, err
	}
	return u, nil
}

// UpdateProfileInput: nil means "leave unchanged". An empty Phone clears it.
// Email, status and roles are deliberately not changeable here.
type UpdateProfileInput struct {
	Name  *string
	Phone *string
}

func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, in UpdateProfileInput) (User, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	errs := validation.Errors{}
	name, phone := current.Name, current.Phone
	if in.Name != nil {
		name = CleanName(errs, "name", *in.Name)
	}
	if in.Phone != nil {
		phone = cleanPhone(errs, "phone", *in.Phone)
	}
	if err := errs.Err(); err != nil {
		return User{}, err
	}

	updated, err := s.repo.UpdateProfile(ctx, id, name, phone)
	if err != nil {
		return User{}, err
	}
	if updated.Roles, err = s.repo.RoleNames(ctx, id); err != nil {
		return User{}, err
	}
	return updated, nil
}

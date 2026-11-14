package orders

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// Request validation runs before any database access, so a Service without
// a pool is enough here.
func fieldErrors(t *testing.T, err error) validation.Errors {
	t.Helper()
	var fields validation.Errors
	require.True(t, errors.As(err, &fields), "want validation errors, got %v", err)
	return fields
}

func TestUpdateStatusForStore_Validation(t *testing.T) {
	s := &Service{}
	ctx, id := context.Background(), uuid.New()

	_, err := s.UpdateStatusForStore(ctx, id, id, "", "")
	assert.Contains(t, fieldErrors(t, err), "status")

	_, err = s.UpdateStatusForStore(ctx, id, id, "teleported", "")
	assert.Contains(t, fieldErrors(t, err), "status")

	_, err = s.UpdateStatusForStore(ctx, id, id, StatusShipped, strings.Repeat("é", MaxNoteLength+1))
	fields := fieldErrors(t, err)
	assert.Contains(t, fields, "note")
	assert.NotContains(t, fields, "status")
}

func TestListForStore_Validation(t *testing.T) {
	s := &Service{}
	_, err := s.ListForStore(context.Background(), uuid.New(), "lost", 0, 999)
	fields := fieldErrors(t, err)
	assert.Contains(t, fields, "status")
	assert.Contains(t, fields, "pageSize")
}

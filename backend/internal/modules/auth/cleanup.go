package auth

import (
	"context"
	"log/slog"
	"time"
)

// CleanupExpiredSessions deletes every expired session, batch rows per
// statement, and returns the total. Expired sessions already fail
// GetActiveSession, so this is housekeeping, not security: without it the
// table keeps a row for every login of a user who never comes back (Login
// only cleans up the sessions of the user who is logging in).
//
// Each DELETE is its own short statement (autocommit), so a big backlog never
// holds one long transaction. Deleting a session row conflicts with nothing a
// request does: a request either still finds it (not yet deleted, already
// expired, so it is refused anyway) or does not.
func (s *Service) CleanupExpiredSessions(ctx context.Context, batch int32) (int64, error) {
	repo := NewRepository(s.pool)
	var total int64
	for {
		n, err := repo.DeleteExpiredBatch(ctx, batch)
		total += n
		if err != nil {
			return total, err
		}
		if n < int64(batch) {
			return total, nil
		}
	}
}

// RunSessionCleanup runs CleanupExpiredSessions once at start and then every
// interval until ctx is cancelled. It runs in the API process next to the
// checkout sweeper; with several API instances each runs it, which is
// harmless (two DELETEs of the same rows: one deletes, the other finds none).
func (s *Service) RunSessionCleanup(ctx context.Context, interval time.Duration, batch int32) {
	s.cleanupOnce(ctx, batch)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.cleanupOnce(ctx, batch)
		}
	}
}

func (s *Service) cleanupOnce(ctx context.Context, batch int32) {
	n, err := s.CleanupExpiredSessions(ctx, batch)
	if err != nil {
		if ctx.Err() == nil { // a cancelled ctx is just the shutdown
			slog.ErrorContext(ctx, "delete expired sessions", "error", err, "deleted", n)
		}
		return
	}
	if n > 0 {
		slog.InfoContext(ctx, "deleted expired sessions", "count", n)
	}
}

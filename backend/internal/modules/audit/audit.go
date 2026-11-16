// Package audit records admin actions (Phase 6d, PLAN.md Q11).
//
// Write takes the CALLER's transaction, like the inventory functions: the
// admin action and its audit row commit together or roll back together, so
// the log can neither miss an action that happened nor show one that did not.
// The table is append-only; there is no update or delete.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

// Actions written in Phase 6d.
const (
	UserSuspend     = "user.suspend"
	UserReactivate  = "user.reactivate"
	StoreSuspend    = "store.suspend"
	StoreReactivate = "store.reactivate"
	CategoryCreate  = "category.create"
	CategoryUpdate  = "category.update"
	ReviewHide      = "review.hide"
	ReviewUnhide    = "review.unhide"
)

// Target types.
const (
	TargetUser     = "user"
	TargetStore    = "store"
	TargetCategory = "category"
	TargetReview   = "review"
)

// Entry is one admin action. Details holds a small snapshot that makes the row
// readable on its own (e.g. the email of the suspended user); never secrets.
type Entry struct {
	ActorID    uuid.UUID
	Action     string
	TargetType string
	TargetID   uuid.UUID
	Details    map[string]any
}

// Write inserts the entry using tx, which must be the transaction of the
// action itself.
func Write(ctx context.Context, tx queries.DBTX, e Entry) error {
	details := e.Details
	if details == nil {
		details = map[string]any{}
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("audit details: %w", err)
	}
	err = queries.New(tx).InsertAuditLog(ctx, queries.InsertAuditLogParams{
		ActorID: e.ActorID, Action: e.Action, TargetType: e.TargetType, TargetID: e.TargetID, Details: raw,
	})
	if err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

// Log is one row of the audit log page.
type Log struct {
	ID         uuid.UUID
	Action     string
	TargetType string
	TargetID   uuid.UUID
	Details    json.RawMessage
	CreatedAt  time.Time
	ActorID    uuid.UUID
	ActorName  string
	ActorEmail string
}

// List returns one page of the log, newest first, and the total row count.
func List(ctx context.Context, db queries.DBTX, limit, offset int32) ([]Log, int64, error) {
	q := queries.New(db)
	total, err := q.CountAuditLogs(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}
	rows, err := q.ListAuditLogs(ctx, queries.ListAuditLogsParams{PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	out := make([]Log, len(rows))
	for i, r := range rows {
		out[i] = Log{
			ID: r.ID, Action: r.Action, TargetType: r.TargetType, TargetID: r.TargetID,
			Details: json.RawMessage(r.Details), CreatedAt: r.CreatedAt,
			ActorID: r.ActorID, ActorName: r.ActorName, ActorEmail: r.ActorEmail,
		}
	}
	return out, total, nil
}

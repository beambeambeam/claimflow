package claim

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
)

// Edit replaces a claim's content in one transaction when the policy and version allow it.
func (s Service) Edit(ctx context.Context, actor Actor, id uuid.UUID, version int, in Input) (Claim, error) {
	now := s.now()
	var edited Claim

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		queries := db.New(tx)
		current, err := checkContentAction(ctx, queries, actor, id, version, ActionEdit)
		if err != nil {
			return err
		}
		if err := checkRules(in, time.Time{}, time.Time{}); err != nil {
			return err
		}

		if current != version {
			return ErrConflict
		}

		changed, err := queries.UpdateClaimContent(ctx, db.UpdateClaimContentParams{
			ID: id, Version: int32(version), //nolint:gosec // the handler reads a 31-bit version
			Title: in.Title, Purpose: in.Purpose, TotalSatang: inputTotal(in.Items), Now: timestamptz(now),
		})
		if err != nil { // coverage-ignore
			return fmt.Errorf("update claim content: %w", err)
		}
		if changed == 0 {
			return ErrConflict
		}

		if err := queries.DeleteClaimLineItems(ctx, id); err != nil { // coverage-ignore
			return fmt.Errorf("delete old line items: %w", err)
		}
		if err := insertItems(ctx, queries, id, in.Items); err != nil { // coverage-ignore
			return err
		}

		edited, err = load(ctx, queries, id)
		return err
	})
	if err != nil {
		return Claim{}, fmt.Errorf("edit claim: %w", err)
	}

	_, edited.AllowedActions = Allowed(actor.ID, edited.subject())
	return edited, nil
}

// Delete removes a draft and lets the database cascade the deletion to its line items.
func (s Service) Delete(ctx context.Context, actor Actor, id uuid.UUID, version int) error {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		queries := db.New(tx)
		current, err := checkContentAction(ctx, queries, actor, id, version, ActionDelete)
		if err != nil {
			return err
		}
		if current != version {
			return ErrConflict
		}

		changed, err := queries.DeleteDraftClaim(ctx, db.DeleteDraftClaimParams{
			ID: id, Version: int32(version), //nolint:gosec // the handler reads a 31-bit version
		})
		if err != nil { // coverage-ignore
			return fmt.Errorf("delete draft: %w", err)
		}
		if changed == 0 {
			return ErrConflict
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("delete claim: %w", err)
	}

	return nil
}

func checkContentAction(ctx context.Context, queries *db.Queries, actor Actor, id uuid.UUID, version int, action Action) (int, error) {
	row, err := findClaim(ctx, queries, id)
	if err != nil {
		return 0, err
	}

	current := int(row.Version)
	return current, checkVersionedAction(actor, claimSubject(&row), action, version, current)
}

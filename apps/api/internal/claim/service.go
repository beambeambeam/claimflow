package claim

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
)

var (
	// ErrForbidden means the actor may not take the action. For Create it is BR-15: no assigned reviewer.
	ErrForbidden = errors.New("claim: forbidden")
	// ErrNotFound means no claim has the id.
	ErrNotFound = errors.New("claim: not found")
	// ErrDraftNotSupported means Create was asked to save a draft, which F-05 adds in Sprint 2.
	ErrDraftNotSupported = errors.New("claim: drafts are not supported yet")
)

// bangkok is the zone of every business date and of the claim number year (AS-06). Thailand has no daylight saving time.
var bangkok = time.FixedZone("Asia/Bangkok", 7*60*60)

// Pool is the database handle of a Service: it runs queries and starts the transaction of each call (NFR-06).
type Pool interface {
	db.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Service is the claim module (SDS §3.1).
type Service struct {
	pool Pool
	now  func() time.Time
}

// NewService returns a Service over pool. now is the clock for submission times and the claim number year; tests inject a fixed one.
func NewService(pool Pool, now func() time.Time) Service {
	return Service{pool: pool, now: now}
}

// Create stores a claim and submits it in one transaction: the claim, its line items, and one SUBMIT history entry.
func (s Service) Create(ctx context.Context, actor Actor, in Input, submit bool) (Claim, error) {
	if err := checkRules(in, s.now()); err != nil {
		return Claim{}, err
	}

	return Claim{}, nil
}

// Get returns a claim with its line items and history. F-03 adds the visibility check and the actor.
func (s Service) Get(ctx context.Context, id uuid.UUID) (Claim, error) {
	return Claim{}, nil
}

// claimNumber builds CLM-<Bangkok year of now>-<seq, 6 digits> (SDS §2.2).
func claimNumber(now time.Time, seq int64) string {
	return fmt.Sprintf("CLM-%d-%06d", now.In(bangkok).Year(), seq)
}

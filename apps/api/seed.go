package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
)

// seedUser is one SDS §2.5 demo account. reviewer is the username of its reviewer, or empty for none.
type seedUser struct {
	username string
	name     string
	role     string
	position string
	reviewer string
}

// seedUsers lists every reviewer before the users they review, so each reviewer_id exists when it is used.
var seedUsers = []seedUser{
	{username: "fin.wanida", name: "Wanida", role: "finance", position: "Finance Officer"},
	{username: "rev.prasert", name: "Prasert", role: "reviewer", position: "Department Head", reviewer: "fin.wanida"},
	{username: "rev.malee", name: "Malee S.", role: "reviewer", position: "Section Head A", reviewer: "rev.prasert"},
	{username: "rev.somchai", name: "Somchai", role: "reviewer", position: "Section Head B", reviewer: "rev.prasert"},
	{username: "req.anan", name: "Anan K.", role: "requester", position: "Staff, Section A", reviewer: "rev.malee"},
	{username: "req.ploy", name: "Ploy", role: "requester", position: "Staff, Section A", reviewer: "rev.malee"},
	{username: "req.niran", name: "Niran", role: "requester", position: "Staff, Section B", reviewer: "rev.somchai"},
}

// seed upserts the SDS §2.5 demo users. Each credential account stores a bcrypt hash of password, made at
// cost. A second run updates the same rows. The password never reaches the output or an error message.
func seed(ctx context.Context, databaseURL, password string, cost int, out io.Writer) error {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil { // coverage-ignore
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)

	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		queries := db.New(tx)
		ids := make(map[string]uuid.UUID, len(seedUsers))

		for _, user := range seedUsers {
			reviewerID := uuid.NullUUID{}
			if user.reviewer != "" {
				reviewerID = uuid.NullUUID{UUID: ids[user.reviewer], Valid: true}
			}

			id, err := queries.UpsertSeedUser(ctx, db.UpsertSeedUserParams{
				ID:         uuid.Must(uuid.NewV7()),
				Name:       user.name,
				Email:      user.username + "@claimflow.test",
				Username:   user.username,
				Role:       user.role,
				Position:   user.position,
				ReviewerID: reviewerID,
			})
			if err != nil { // coverage-ignore
				return fmt.Errorf("upsert user %s: %w", user.username, err)
			}
			ids[user.username] = id

			hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
			if err != nil { // coverage-ignore
				return fmt.Errorf("hash password for %s: %w", user.username, err)
			}

			err = queries.UpsertSeedAccount(ctx, db.UpsertSeedAccountParams{
				ID:        uuid.Must(uuid.NewV7()),
				UserID:    id,
				AccountID: id.String(),
				Password:  pgtype.Text{String: string(hash), Valid: true},
			})
			if err != nil { // coverage-ignore
				return fmt.Errorf("upsert account %s: %w", user.username, err)
			}
		}

		return nil
	})
	if err != nil { // coverage-ignore
		return fmt.Errorf("seed: %w", err)
	}

	if _, err := fmt.Fprintf(out, "seeded %d users\n", len(seedUsers)); err != nil { // coverage-ignore
		return fmt.Errorf("write summary: %w", err)
	}

	return nil
}

var errSeedPasswordUnset = errors.New("SEED_USER_PASSWORD is not set")

// seedCommand runs `api seed`. It refuses an empty password, so no demo user is written with a blank credential.
func seedCommand(ctx context.Context, databaseURL, password string, cost int, out io.Writer) error {
	if password == "" {
		return errSeedPasswordUnset
	}

	return seed(ctx, databaseURL, password, cost, out)
}

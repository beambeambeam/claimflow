-- Claim tables (SDS §2.2, F-02 contract §4.1). IDs are UUIDv7, generated in Go. Money is integer satang (SDS §2.3).
-- Every enum value is added now, so later sprints need no enum migration.

-- +goose Up
CREATE TYPE claim_status AS ENUM (
    'DRAFT', 'SUBMITTED', 'RETURNED', 'PENDING_FINANCE', 'APPROVED', 'REJECTED', 'WITHDRAWN'
);

CREATE TYPE claim_action AS ENUM (
    'SUBMIT', 'APPROVE', 'REJECT', 'RETURN', 'RESUBMIT', 'WITHDRAW', 'FINAL_APPROVE', 'FINAL_REJECT'
);

CREATE TYPE expense_category AS ENUM (
    'TRAVEL', 'ACCOMMODATION', 'MEALS', 'SUPPLIES', 'TRAINING', 'OTHER'
);

-- claim_no_seq numbers claims as CLM-<Bangkok year>-<nextval>. It never resets, so numbers stay unique across years.
CREATE SEQUENCE claim_no_seq;

CREATE TABLE claims (
    id                   UUID PRIMARY KEY,
    claim_no             VARCHAR(20)  NOT NULL UNIQUE,
    requester_id         UUID         NOT NULL REFERENCES "user" (id),
    assigned_reviewer_id UUID REFERENCES "user" (id),
    title                VARCHAR(120) NOT NULL,
    purpose              TEXT         NOT NULL CHECK (char_length(purpose) BETWEEN 1 AND 1000),
    status               claim_status NOT NULL,
    total_satang         BIGINT       NOT NULL CHECK (total_satang > 0 AND total_satang <= 5000000),
    return_count         SMALLINT     NOT NULL DEFAULT 0 CHECK (return_count BETWEEN 0 AND 3),
    version              INTEGER      NOT NULL DEFAULT 0,
    first_submitted_at   TIMESTAMPTZ,
    submitted_at         TIMESTAMPTZ,
    decided_at           TIMESTAMPTZ,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CHECK (assigned_reviewer_id <> requester_id),
    CHECK (status = 'DRAFT' OR assigned_reviewer_id IS NOT NULL)
);

CREATE INDEX claims_requester_status_idx ON claims (requester_id, status);
CREATE INDEX claims_assigned_reviewer_status_idx ON claims (assigned_reviewer_id, status);

CREATE TABLE line_items (
    id            UUID PRIMARY KEY,
    claim_id      UUID             NOT NULL REFERENCES claims (id) ON DELETE CASCADE,
    line_no       SMALLINT         NOT NULL CHECK (line_no BETWEEN 1 AND 20),
    category      expense_category NOT NULL,
    description   VARCHAR(200)     NOT NULL,
    expense_date  DATE             NOT NULL,
    amount_satang BIGINT           NOT NULL CHECK (amount_satang > 0 AND amount_satang <= 2000000),
    receipt_ref   VARCHAR(50)      NOT NULL,
    UNIQUE (claim_id, line_no),
    CHECK (category <> 'MEALS' OR amount_satang <= 150000)
);

-- History is append-only (BR-11). The FK has no cascade, so a claim with history cannot be deleted.
CREATE TABLE claim_history (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    claim_id    UUID         NOT NULL REFERENCES claims (id),
    actor_id    UUID         NOT NULL REFERENCES "user" (id),
    action      claim_action NOT NULL,
    from_status claim_status,
    to_status   claim_status NOT NULL,
    comment     VARCHAR(500),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX claim_history_claim_id_idx ON claim_history (claim_id);

-- +goose Down
DROP TABLE claim_history;
DROP TABLE line_items;
DROP TABLE claims;
DROP SEQUENCE claim_no_seq;
DROP TYPE expense_category;
DROP TYPE claim_action;
DROP TYPE claim_status;

-- Indexes for the All Claims filters (F-09 contract §4). The title search has no index: a substring match on a few
-- hundred titles is a scan of a small table, and F-10/02 checks every list query with EXPLAIN on the load data.

-- +goose Up
CREATE INDEX line_items_category_claim_id_idx ON line_items (category, claim_id);
CREATE INDEX claims_status_created_at_idx ON claims (status, created_at DESC);

-- +goose Down
DROP INDEX claims_status_created_at_idx;
DROP INDEX line_items_category_claim_id_idx;

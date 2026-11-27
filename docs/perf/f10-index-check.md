# F-10/02 index check (F10-TEST-11, F10-AC-11)

`EXPLAIN (ANALYZE, BUFFERS)` of the list queries of `GET /claims` (`mine`, `review`, `finance`, `all`, with and without filters) and of the three summary queries of `GET /reports/summary`, on the load data.

- **Commit:** `1ae9425` (`feat: add the NFR-07 timed-run script`), the head of `feat/load-dataset` when the plans were taken (taken as `3c2416e`, the same code before the branch was rebased onto the F10-TEST-13 fix). The migrations are `00001` to `00003`, so the indexes are those of `00001_auth.sql`, `00002_claims.sql`, and `00003_claim_filters.sql`.
- **Database:** PostgreSQL 17.12 (`postgres:17` container) on a local stack: `migrate up`, `seed`, `seed-load`, then `ANALYZE`. Not the deployed database.
- **Data:** 250 claims (10 base and 240 `CLM-LOAD-0001` to `CLM-LOAD-0240`), 736 line items, 520 history rows, 7 users.
- **How:** each query of `internal/claim/queries.sql` and `internal/report/queries.sql` is written out with literal values for its parameters (the actors `req.anan`, `rev.malee`, and `fin.wanida`, page 1 of size 20, the range 2026-07-01 to 2026-11-30 as Bangkok-midnight bounds). Each ran twice; the plans below are from the second run, so the cache is warm. The plans use literal values (custom plans). The API sends parameters, and a prepared statement may switch to a generic plan after five runs; that plan was not measured here.

## Conclusion

Every query runs in **1.4 ms or less** (planning adds at most 1.2 ms) against the 1 s of NFR-07, so none misses it on the load data. **No index migration is added.**

Most plans are sequential scans of `claims`, which has 250 rows (11 to 14 pages): the planner is right to prefer them to an index at this size. The existing indexes are used where they pay: `claims_status_created_at_idx` for a `status` filter (Q8) and `line_items_category_claim_id_idx` for the `category` check (Q6). A new index would change nothing measurable here, and it would slow writes. The picture changes only with far more rows; NFR-07 is stated for this dataset, and the check should run again if the data grows by orders of magnitude.

| Query | Execution time (ms) |
| --- | --: |
| Q1 mine: ListRequesterClaims (req.anan, page 1, size 20) | 0.445 |
| Q1c mine: CountRequesterClaims | 0.114 |
| Q2 review: ListReviewClaims (rev.malee, page 1, size 20) | 0.255 |
| Q2c review: CountReviewClaims | 0.107 |
| Q3 finance: ListFinanceClaims (page 1, size 20) | 0.263 |
| Q3c finance: CountFinanceClaims | 0.124 |
| Q4 all, no filters, Finance Officer (fin.wanida): ListAllClaims | 0.872 |
| Q4c all, no filters, Finance Officer: CountAllClaims | 0.154 |
| Q5 all, no filters, reviewer (rev.malee): ListAllClaims | 0.331 |
| Q5c all, no filters, reviewer: CountAllClaims | 0.128 |
| Q6 all, status + category + range + q, Finance Officer: ListAllClaims (the timed run request) | 0.476 |
| Q6c all, status + category + range + q: CountAllClaims | 0.633 |
| Q7 all, category only (TRAVEL), Finance Officer: ListAllClaims | 0.506 |
| Q8 all, status only (SUBMITTED), Finance Officer: ListAllClaims | 0.141 |
| S1 summary: CountClaimsByStatus (Finance Officer, all claims) | 0.175 |
| S1r summary: CountClaimsByStatus (reviewer rev.malee) | 0.150 |
| S2 summary: ApprovedTotalsByMonthCategory (Finance Officer, no range) | 1.125 |
| S2x summary: ApprovedTotalsByMonthCategory (Finance Officer, range 2026-07-01 to 2026-11-30) | 1.313 |
| S2r summary: ApprovedTotalsByMonthCategory (reviewer rev.malee, no range) | 0.492 |
| S3 summary: AverageDecisionHours (Finance Officer, no range) | 0.368 |
| S3x summary: AverageDecisionHours (Finance Officer, range 2026-07-01 to 2026-11-30) | 0.241 |

## Plans

### Q1 mine: ListRequesterClaims (req.anan, page 1, size 20)

```
                                                       QUERY PLAN
-------------------------------------------------------------------------------------------------------------------------
 Limit  (cost=14.84..14.89 rows=20 width=126) (actual time=0.341..0.347 rows=20 loops=1)
   Buffers: shared hit=16
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.006..0.007 rows=1 loops=1)
           Filter: ((username)::text = 'req.anan'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Sort  (cost=13.75..13.86 rows=42 width=126) (actual time=0.334..0.337 rows=20 loops=1)
         Sort Key: c.created_at DESC, c.id DESC
         Sort Method: top-N heapsort  Memory: 33kB
         Buffers: shared hit=16
         ->  Nested Loop  (cost=0.00..12.63 rows=42 width=126) (actual time=0.085..0.216 rows=72 loops=1)
               Buffers: shared hit=10
               ->  Seq Scan on "user" r  (cost=0.00..1.09 rows=1 width=39) (actual time=0.062..0.063 rows=1 loops=1)
                     Filter: (id = (InitPlan 1).col1)
                     Rows Removed by Filter: 6
                     Buffers: shared hit=2
               ->  Seq Scan on claims c  (cost=0.00..11.12 rows=42 width=103) (actual time=0.020..0.121 rows=72 loops=1)
                     Filter: (requester_id = (InitPlan 1).col1)
                     Rows Removed by Filter: 178
                     Buffers: shared hit=8
 Planning:
   Buffers: shared hit=245
 Planning Time: 1.126 ms
 Execution Time: 0.445 ms
(25 rows)

```

### Q1c mine: CountRequesterClaims

```
                                               QUERY PLAN
---------------------------------------------------------------------------------------------------------
 Aggregate  (cost=12.32..12.33 rows=1 width=4) (actual time=0.080..0.081 rows=1 loops=1)
   Buffers: shared hit=9
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.006..0.007 rows=1 loops=1)
           Filter: ((username)::text = 'req.anan'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Seq Scan on claims  (cost=0.00..11.12 rows=42 width=0) (actual time=0.020..0.070 rows=72 loops=1)
         Filter: (requester_id = (InitPlan 1).col1)
         Rows Removed by Filter: 178
         Buffers: shared hit=9
 Planning:
   Buffers: shared hit=3
 Planning Time: 0.179 ms
 Execution Time: 0.114 ms
(15 rows)

```

### Q2 review: ListReviewClaims (rev.malee, page 1, size 20)

```
                                                        QUERY PLAN
---------------------------------------------------------------------------------------------------------------------------
 Limit  (cost=14.69..14.74 rows=20 width=126) (actual time=0.210..0.214 rows=20 loops=1)
   Buffers: shared hit=10
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.004..0.005 rows=1 loops=1)
           Filter: ((username)::text = 'rev.malee'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Sort  (cost=13.60..13.66 rows=25 width=126) (actual time=0.208..0.211 rows=20 loops=1)
         Sort Key: c.submitted_at, c.id
         Sort Method: top-N heapsort  Memory: 33kB
         Buffers: shared hit=10
         ->  Hash Join  (cost=1.16..13.02 rows=25 width=126) (actual time=0.063..0.162 rows=87 loops=1)
               Hash Cond: (c.requester_id = r.id)
               Buffers: shared hit=10
               ->  Seq Scan on claims c  (cost=0.00..11.75 rows=25 width=103) (actual time=0.017..0.083 rows=87 loops=1)
                     Filter: ((assigned_reviewer_id = (InitPlan 1).col1) AND (status = 'SUBMITTED'::claim_status))
                     Rows Removed by Filter: 163
                     Buffers: shared hit=9
               ->  Hash  (cost=1.07..1.07 rows=7 width=39) (actual time=0.025..0.025 rows=7 loops=1)
                     Buckets: 1024  Batches: 1  Memory Usage: 9kB
                     Buffers: shared hit=1
                     ->  Seq Scan on "user" r  (cost=0.00..1.07 rows=7 width=39) (actual time=0.002..0.004 rows=7 loops=1)
                           Buffers: shared hit=1
 Planning:
   Buffers: shared hit=31
 Planning Time: 0.480 ms
 Execution Time: 0.255 ms
(27 rows)

```

### Q2c review: CountReviewClaims

```
                                               QUERY PLAN
---------------------------------------------------------------------------------------------------------
 Aggregate  (cost=12.90..12.91 rows=1 width=4) (actual time=0.084..0.085 rows=1 loops=1)
   Buffers: shared hit=9
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.004..0.006 rows=1 loops=1)
           Filter: ((username)::text = 'rev.malee'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Seq Scan on claims  (cost=0.00..11.75 rows=25 width=0) (actual time=0.019..0.075 rows=87 loops=1)
         Filter: ((assigned_reviewer_id = (InitPlan 1).col1) AND (status = 'SUBMITTED'::claim_status))
         Rows Removed by Filter: 163
         Buffers: shared hit=9
 Planning Time: 0.138 ms
 Execution Time: 0.107 ms
(13 rows)

```

### Q3 finance: ListFinanceClaims (page 1, size 20)

```
                                                        QUERY PLAN
---------------------------------------------------------------------------------------------------------------------------
 Limit  (cost=12.64..12.67 rows=15 width=126) (actual time=0.212..0.219 rows=15 loops=1)
   Buffers: shared hit=9
   ->  Sort  (cost=12.64..12.67 rows=15 width=126) (actual time=0.210..0.214 rows=15 loops=1)
         Sort Key: c.submitted_at, c.id
         Sort Method: quicksort  Memory: 28kB
         Buffers: shared hit=9
         ->  Hash Join  (cost=1.16..12.34 rows=15 width=126) (actual time=0.076..0.187 rows=15 loops=1)
               Hash Cond: (c.requester_id = r.id)
               Buffers: shared hit=9
               ->  Seq Scan on claims c  (cost=0.00..11.12 rows=15 width=103) (actual time=0.022..0.118 rows=15 loops=1)
                     Filter: (status = 'PENDING_FINANCE'::claim_status)
                     Rows Removed by Filter: 235
                     Buffers: shared hit=8
               ->  Hash  (cost=1.07..1.07 rows=7 width=39) (actual time=0.042..0.043 rows=7 loops=1)
                     Buckets: 1024  Batches: 1  Memory Usage: 9kB
                     Buffers: shared hit=1
                     ->  Seq Scan on "user" r  (cost=0.00..1.07 rows=7 width=39) (actual time=0.006..0.010 rows=7 loops=1)
                           Buffers: shared hit=1
 Planning:
   Buffers: shared hit=4
 Planning Time: 0.507 ms
 Execution Time: 0.263 ms
(22 rows)

```

### Q3c finance: CountFinanceClaims

```
                                               QUERY PLAN
---------------------------------------------------------------------------------------------------------
 Aggregate  (cost=11.16..11.17 rows=1 width=4) (actual time=0.095..0.096 rows=1 loops=1)
   Buffers: shared hit=8
   ->  Seq Scan on claims  (cost=0.00..11.12 rows=15 width=0) (actual time=0.018..0.086 rows=15 loops=1)
         Filter: (status = 'PENDING_FINANCE'::claim_status)
         Rows Removed by Filter: 235
         Buffers: shared hit=8
 Planning Time: 0.089 ms
 Execution Time: 0.124 ms
(8 rows)

```

### Q4 all, no filters, Finance Officer (fin.wanida): ListAllClaims

```
                                                        QUERY PLAN
---------------------------------------------------------------------------------------------------------------------------
 Limit  (cost=21.67..21.72 rows=20 width=126) (actual time=0.818..0.827 rows=20 loops=1)
   Buffers: shared hit=10
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.006..0.009 rows=1 loops=1)
           Filter: ((username)::text = 'fin.wanida'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Sort  (cost=20.59..21.21 rows=249 width=126) (actual time=0.816..0.820 rows=20 loops=1)
         Sort Key: c.created_at DESC, c.id DESC
         Sort Method: top-N heapsort  Memory: 34kB
         Buffers: shared hit=10
         ->  Hash Join  (cost=1.16..13.96 rows=249 width=126) (actual time=0.055..0.605 rows=249 loops=1)
               Hash Cond: (c.requester_id = r.id)
               Buffers: shared hit=10
               ->  Seq Scan on claims c  (cost=0.00..11.75 rows=249 width=103) (actual time=0.028..0.165 rows=249 loops=1)
                     Filter: ((requester_id = (InitPlan 1).col1) OR (status <> 'DRAFT'::claim_status))
                     Rows Removed by Filter: 1
                     Buffers: shared hit=9
               ->  Hash  (cost=1.07..1.07 rows=7 width=39) (actual time=0.009..0.010 rows=7 loops=1)
                     Buckets: 1024  Batches: 1  Memory Usage: 9kB
                     Buffers: shared hit=1
                     ->  Seq Scan on "user" r  (cost=0.00..1.07 rows=7 width=39) (actual time=0.003..0.005 rows=7 loops=1)
                           Buffers: shared hit=1
 Planning:
   Buffers: shared hit=6
 Planning Time: 0.336 ms
 Execution Time: 0.872 ms
(27 rows)

```

### Q4c all, no filters, Finance Officer: CountAllClaims

```
                                                 QUERY PLAN
-------------------------------------------------------------------------------------------------------------
 Aggregate  (cost=13.46..13.47 rows=1 width=4) (actual time=0.131..0.132 rows=1 loops=1)
   Buffers: shared hit=9
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.004..0.006 rows=1 loops=1)
           Filter: ((username)::text = 'fin.wanida'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Seq Scan on claims c  (cost=0.00..11.75 rows=249 width=0) (actual time=0.019..0.107 rows=249 loops=1)
         Filter: ((requester_id = (InitPlan 1).col1) OR (status <> 'DRAFT'::claim_status))
         Rows Removed by Filter: 1
         Buffers: shared hit=9
 Planning Time: 0.139 ms
 Execution Time: 0.154 ms
(13 rows)

```

### Q5 all, no filters, reviewer (rev.malee): ListAllClaims

```
                                                                       QUERY PLAN
--------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit  (cost=18.57..18.62 rows=20 width=126) (actual time=0.285..0.290 rows=20 loops=1)
   Buffers: shared hit=11
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.004..0.005 rows=1 loops=1)
           Filter: ((username)::text = 'rev.malee'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   InitPlan 2
     ->  Seq Scan on "user" user_1  (cost=0.00..1.09 rows=1 width=16) (actual time=0.009..0.010 rows=1 loops=1)
           Filter: ((username)::text = 'rev.malee'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Sort  (cost=16.40..16.63 rows=93 width=126) (actual time=0.284..0.286 rows=20 loops=1)
         Sort Key: c.created_at DESC, c.id DESC
         Sort Method: top-N heapsort  Memory: 34kB
         Buffers: shared hit=11
         ->  Hash Join  (cost=1.16..13.92 rows=93 width=126) (actual time=0.048..0.205 rows=167 loops=1)
               Hash Cond: (c.requester_id = r.id)
               Buffers: shared hit=11
               ->  Seq Scan on claims c  (cost=0.00..12.38 rows=93 width=103) (actual time=0.031..0.123 rows=167 loops=1)
                     Filter: ((requester_id = (InitPlan 1).col1) OR ((status <> 'DRAFT'::claim_status) AND (assigned_reviewer_id = (InitPlan 2).col1)))
                     Rows Removed by Filter: 83
                     Buffers: shared hit=10
               ->  Hash  (cost=1.07..1.07 rows=7 width=39) (actual time=0.007..0.008 rows=7 loops=1)
                     Buckets: 1024  Batches: 1  Memory Usage: 9kB
                     Buffers: shared hit=1
                     ->  Seq Scan on "user" r  (cost=0.00..1.07 rows=7 width=39) (actual time=0.002..0.003 rows=7 loops=1)
                           Buffers: shared hit=1
 Planning:
   Buffers: shared hit=4
 Planning Time: 0.488 ms
 Execution Time: 0.331 ms
(32 rows)

```

### Q5c all, no filters, reviewer: CountAllClaims

```
                                                                 QUERY PLAN
--------------------------------------------------------------------------------------------------------------------------------------------
 Aggregate  (cost=14.78..14.79 rows=1 width=4) (actual time=0.101..0.103 rows=1 loops=1)
   Buffers: shared hit=10
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.004..0.006 rows=1 loops=1)
           Filter: ((username)::text = 'rev.malee'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   InitPlan 2
     ->  Seq Scan on "user" user_1  (cost=0.00..1.09 rows=1 width=16) (actual time=0.002..0.003 rows=1 loops=1)
           Filter: ((username)::text = 'rev.malee'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Seq Scan on claims c  (cost=0.00..12.38 rows=93 width=0) (actual time=0.025..0.088 rows=167 loops=1)
         Filter: ((requester_id = (InitPlan 1).col1) OR ((status <> 'DRAFT'::claim_status) AND (assigned_reviewer_id = (InitPlan 2).col1)))
         Rows Removed by Filter: 83
         Buffers: shared hit=10
 Planning Time: 0.168 ms
 Execution Time: 0.128 ms
(18 rows)

```

### Q6 all, status + category + range + q, Finance Officer: ListAllClaims (the timed run request)

```
                                                                                                                                                                                                QUERY PLAN
-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit  (cost=32.82..32.83 rows=1 width=126) (actual time=0.437..0.441 rows=11 loops=1)
   Buffers: shared hit=74
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.003..0.004 rows=1 loops=1)
           Filter: ((username)::text = 'fin.wanida'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Sort  (cost=31.73..31.74 rows=1 width=126) (actual time=0.436..0.438 rows=11 loops=1)
         Sort Key: c.created_at DESC, c.id DESC
         Sort Method: quicksort  Memory: 27kB
         Buffers: shared hit=74
         ->  Nested Loop Semi Join  (cost=0.28..31.72 rows=1 width=126) (actual time=0.056..0.423 rows=11 loops=1)
               Buffers: shared hit=74
               ->  Nested Loop  (cost=0.00..15.41 rows=1 width=126) (actual time=0.025..0.341 rows=18 loops=1)
                     Join Filter: (c.requester_id = r.id)
                     Rows Removed by Join Filter: 58
                     Buffers: shared hit=27
                     ->  Seq Scan on claims c  (cost=0.00..14.25 rows=1 width=103) (actual time=0.021..0.303 rows=18 loops=1)
                           Filter: ((COALESCE(first_submitted_at, created_at) >= '2026-06-30 17:00:00+00'::timestamp with time zone) AND (COALESCE(first_submitted_at, created_at) < '2026-11-30 17:00:00+00'::timestamp with time zone) AND ((title)::text ~~* '%supplies%'::text) AND (status = 'APPROVED'::claim_status) AND ((requester_id = (InitPlan 1).col1) OR (status <> 'DRAFT'::claim_status)))
                           Rows Removed by Filter: 232
                           Buffers: shared hit=9
                     ->  Seq Scan on "user" r  (cost=0.00..1.07 rows=7 width=39) (actual time=0.001..0.001 rows=4 loops=18)
                           Buffers: shared hit=18
               ->  Index Only Scan using line_items_category_claim_id_idx on line_items li  (cost=0.28..8.29 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=18)
                     Index Cond: ((category = 'SUPPLIES'::expense_category) AND (claim_id = c.id))
                     Heap Fetches: 11
                     Buffers: shared hit=47
 Planning:
   Buffers: shared hit=128
 Planning Time: 1.191 ms
 Execution Time: 0.476 ms
(31 rows)

```

### Q6c all, status + category + range + q: CountAllClaims

```
                                                                                                                                                                                          QUERY PLAN
-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Aggregate  (cost=31.65..31.67 rows=1 width=4) (actual time=0.579..0.582 rows=1 loops=1)
   Buffers: shared hit=56
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.005..0.007 rows=1 loops=1)
           Filter: ((username)::text = 'fin.wanida'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Nested Loop Semi Join  (cost=0.28..30.56 rows=1 width=0) (actual time=0.042..0.572 rows=11 loops=1)
         Buffers: shared hit=56
         ->  Seq Scan on claims c  (cost=0.00..14.25 rows=1 width=16) (actual time=0.030..0.476 rows=18 loops=1)
               Filter: ((COALESCE(first_submitted_at, created_at) >= '2026-06-30 17:00:00+00'::timestamp with time zone) AND (COALESCE(first_submitted_at, created_at) < '2026-11-30 17:00:00+00'::timestamp with time zone) AND ((title)::text ~~* '%supplies%'::text) AND (status = 'APPROVED'::claim_status) AND ((requester_id = (InitPlan 1).col1) OR (status <> 'DRAFT'::claim_status)))
               Rows Removed by Filter: 232
               Buffers: shared hit=9
         ->  Index Only Scan using line_items_category_claim_id_idx on line_items li  (cost=0.28..8.29 rows=1 width=16) (actual time=0.004..0.004 rows=1 loops=18)
               Index Cond: ((category = 'SUPPLIES'::expense_category) AND (claim_id = c.id))
               Heap Fetches: 11
               Buffers: shared hit=47
 Planning:
   Buffers: shared hit=10
 Planning Time: 0.640 ms
 Execution Time: 0.633 ms
(21 rows)

```

### Q7 all, category only (TRAVEL), Finance Officer: ListAllClaims

```
                                                                QUERY PLAN
-------------------------------------------------------------------------------------------------------------------------------------------
 Limit  (cost=43.71..43.76 rows=20 width=126) (actual time=0.452..0.459 rows=20 loops=1)
   Buffers: shared hit=22
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.003..0.005 rows=1 loops=1)
           Filter: ((username)::text = 'fin.wanida'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Sort  (cost=42.62..42.95 rows=132 width=126) (actual time=0.451..0.454 rows=20 loops=1)
         Sort Key: c.created_at DESC, c.id DESC
         Sort Method: top-N heapsort  Memory: 34kB
         Buffers: shared hit=22
         ->  Hash Join  (cost=24.02..39.11 rows=132 width=126) (actual time=0.206..0.386 rows=103 loops=1)
               Hash Cond: (c.requester_id = r.id)
               Buffers: shared hit=22
               ->  Hash Semi Join  (cost=22.86..37.39 rows=132 width=103) (actual time=0.163..0.309 rows=103 loops=1)
                     Hash Cond: (c.id = li.claim_id)
                     Buffers: shared hit=21
                     ->  Seq Scan on claims c  (cost=0.00..11.75 rows=249 width=103) (actual time=0.011..0.097 rows=249 loops=1)
                           Filter: ((requester_id = (InitPlan 1).col1) OR (status <> 'DRAFT'::claim_status))
                           Rows Removed by Filter: 1
                           Buffers: shared hit=9
                     ->  Hash  (cost=21.20..21.20 rows=133 width=16) (actual time=0.146..0.146 rows=133 loops=1)
                           Buckets: 1024  Batches: 1  Memory Usage: 15kB
                           Buffers: shared hit=12
                           ->  Seq Scan on line_items li  (cost=0.00..21.20 rows=133 width=16) (actual time=0.005..0.112 rows=133 loops=1)
                                 Filter: (category = 'TRAVEL'::expense_category)
                                 Rows Removed by Filter: 603
                                 Buffers: shared hit=12
               ->  Hash  (cost=1.07..1.07 rows=7 width=39) (actual time=0.018..0.019 rows=7 loops=1)
                     Buckets: 1024  Batches: 1  Memory Usage: 9kB
                     Buffers: shared hit=1
                     ->  Seq Scan on "user" r  (cost=0.00..1.07 rows=7 width=39) (actual time=0.012..0.014 rows=7 loops=1)
                           Buffers: shared hit=1
 Planning:
   Buffers: shared hit=24
 Planning Time: 0.570 ms
 Execution Time: 0.506 ms
(37 rows)

```

### Q8 all, status only (SUBMITTED), Finance Officer: ListAllClaims

```
                                                                          QUERY PLAN
---------------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit  (cost=1.80..10.86 rows=20 width=126) (actual time=0.101..0.106 rows=20 loops=1)
   Buffers: shared hit=33
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (never executed)
           Filter: ((username)::text = 'fin.wanida'::text)
   ->  Incremental Sort  (cost=0.71..46.03 rows=100 width=126) (actual time=0.101..0.103 rows=20 loops=1)
         Sort Key: c.created_at DESC, c.id DESC
         Presorted Key: c.created_at
         Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 30kB  Peak Memory: 30kB
         Buffers: shared hit=33
         ->  Nested Loop  (cost=0.29..41.53 rows=100 width=126) (actual time=0.023..0.070 rows=21 loops=1)
               Buffers: shared hit=30
               ->  Index Scan using claims_status_created_at_idx on claims c  (cost=0.14..37.41 rows=100 width=103) (actual time=0.013..0.031 rows=21 loops=1)
                     Index Cond: (status = 'SUBMITTED'::claim_status)
                     Buffers: shared hit=18
               ->  Memoize  (cost=0.14..0.28 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=21)
                     Cache Key: c.requester_id
                     Cache Mode: logical
                     Hits: 15  Misses: 6  Evictions: 0  Overflows: 0  Memory Usage: 1kB
                     Buffers: shared hit=12
                     ->  Index Scan using user_pkey on "user" r  (cost=0.13..0.27 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=6)
                           Index Cond: (id = c.requester_id)
                           Buffers: shared hit=12
 Planning:
   Buffers: shared hit=6
 Planning Time: 0.351 ms
 Execution Time: 0.141 ms
(27 rows)

```

### S1 summary: CountClaimsByStatus (Finance Officer, all claims)

```
                                                QUERY PLAN
-----------------------------------------------------------------------------------------------------------
 HashAggregate  (cost=12.37..12.46 rows=7 width=8) (actual time=0.144..0.146 rows=6 loops=1)
   Group Key: status
   Batches: 1  Memory Usage: 24kB
   Buffers: shared hit=8
   ->  Seq Scan on claims  (cost=0.00..11.12 rows=249 width=4) (actual time=0.012..0.089 rows=249 loops=1)
         Filter: (status <> 'DRAFT'::claim_status)
         Rows Removed by Filter: 1
         Buffers: shared hit=8
 Planning:
   Buffers: shared hit=3
 Planning Time: 0.127 ms
 Execution Time: 0.175 ms
(12 rows)

```

### S1r summary: CountClaimsByStatus (reviewer rev.malee)

```
                                                QUERY PLAN
----------------------------------------------------------------------------------------------------------
 HashAggregate  (cost=13.15..13.23 rows=7 width=8) (actual time=0.115..0.117 rows=6 loops=1)
   Group Key: claims.status
   Batches: 1  Memory Usage: 24kB
   Buffers: shared hit=9
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.004..0.006 rows=1 loops=1)
           Filter: ((username)::text = 'rev.malee'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Seq Scan on claims  (cost=0.00..11.75 rows=62 width=4) (actual time=0.023..0.088 rows=140 loops=1)
         Filter: ((status <> 'DRAFT'::claim_status) AND (assigned_reviewer_id = (InitPlan 1).col1))
         Rows Removed by Filter: 110
         Buffers: shared hit=9
 Planning Time: 0.202 ms
 Execution Time: 0.150 ms
(15 rows)

```

### S2 summary: ApprovedTotalsByMonthCategory (Finance Officer, no range)

```
                                                          QUERY PLAN
-------------------------------------------------------------------------------------------------------------------------------
 GroupAggregate  (cost=41.78..47.69 rows=197 width=44) (actual time=1.032..1.087 rows=30 loops=1)
   Group Key: (to_char((c.decided_at AT TIME ZONE 'Asia/Bangkok'::text), 'YYYY-MM'::text)), li.category
   Buffers: shared hit=28
   ->  Sort  (cost=41.78..42.27 rows=197 width=44) (actual time=1.020..1.032 rows=197 loops=1)
         Sort Key: (to_char((c.decided_at AT TIME ZONE 'Asia/Bangkok'::text), 'YYYY-MM'::text)), li.category
         Sort Method: quicksort  Memory: 32kB
         Buffers: shared hit=28
         ->  Hash Join  (cost=11.96..34.27 rows=197 width=44) (actual time=0.466..0.854 rows=197 loops=1)
               Hash Cond: (li.claim_id = c.id)
               Buffers: shared hit=20
               ->  Seq Scan on line_items li  (cost=0.00..19.36 rows=736 width=28) (actual time=0.010..0.096 rows=736 loops=1)
                     Buffers: shared hit=12
               ->  Hash  (cost=11.12..11.12 rows=67 width=24) (actual time=0.086..0.087 rows=67 loops=1)
                     Buckets: 1024  Batches: 1  Memory Usage: 12kB
                     Buffers: shared hit=8
                     ->  Seq Scan on claims c  (cost=0.00..11.12 rows=67 width=24) (actual time=0.008..0.071 rows=67 loops=1)
                           Filter: (status = 'APPROVED'::claim_status)
                           Rows Removed by Filter: 183
                           Buffers: shared hit=8
 Planning:
   Buffers: shared hit=30
 Planning Time: 0.535 ms
 Execution Time: 1.125 ms
(23 rows)

```

### S2x summary: ApprovedTotalsByMonthCategory (Finance Officer, range 2026-07-01 to 2026-11-30)

```
                                                                                                      QUERY PLAN
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 GroupAggregate  (cost=36.92..39.29 rows=79 width=44) (actual time=1.185..1.268 rows=30 loops=1)
   Group Key: (to_char((c.decided_at AT TIME ZONE 'Asia/Bangkok'::text), 'YYYY-MM'::text)), li.category
   Buffers: shared hit=20
   ->  Sort  (cost=36.92..37.12 rows=79 width=44) (actual time=1.169..1.188 rows=197 loops=1)
         Sort Key: (to_char((c.decided_at AT TIME ZONE 'Asia/Bangkok'::text), 'YYYY-MM'::text)), li.category
         Sort Method: quicksort  Memory: 32kB
         Buffers: shared hit=20
         ->  Hash Join  (cost=12.71..34.43 rows=79 width=44) (actual time=0.383..0.966 rows=197 loops=1)
               Hash Cond: (li.claim_id = c.id)
               Buffers: shared hit=20
               ->  Seq Scan on line_items li  (cost=0.00..19.36 rows=736 width=28) (actual time=0.012..0.132 rows=736 loops=1)
                     Buffers: shared hit=12
               ->  Hash  (cost=12.38..12.38 rows=27 width=24) (actual time=0.346..0.347 rows=67 loops=1)
                     Buckets: 1024  Batches: 1  Memory Usage: 12kB
                     Buffers: shared hit=8
                     ->  Seq Scan on claims c  (cost=0.00..12.38 rows=27 width=24) (actual time=0.009..0.320 rows=67 loops=1)
                           Filter: ((decided_at >= '2026-06-30 17:00:00+00'::timestamp with time zone) AND (decided_at < '2026-11-30 17:00:00+00'::timestamp with time zone) AND (status = 'APPROVED'::claim_status))
                           Rows Removed by Filter: 183
                           Buffers: shared hit=8
 Planning:
   Buffers: shared hit=13
 Planning Time: 0.512 ms
 Execution Time: 1.313 ms
(23 rows)

```

### S2r summary: ApprovedTotalsByMonthCategory (reviewer rev.malee, no range)

```
                                                          QUERY PLAN
-------------------------------------------------------------------------------------------------------------------------------
 GroupAggregate  (cost=36.04..37.54 rows=50 width=44) (actual time=0.424..0.452 rows=19 loops=1)
   Group Key: (to_char((c.decided_at AT TIME ZONE 'Asia/Bangkok'::text), 'YYYY-MM'::text)), li.category
   Buffers: shared hit=21
   InitPlan 1
     ->  Seq Scan on "user"  (cost=0.00..1.09 rows=1 width=16) (actual time=0.005..0.006 rows=1 loops=1)
           Filter: ((username)::text = 'rev.malee'::text)
           Rows Removed by Filter: 6
           Buffers: shared hit=1
   ->  Sort  (cost=34.95..35.07 rows=50 width=44) (actual time=0.414..0.420 rows=56 loops=1)
         Sort Key: (to_char((c.decided_at AT TIME ZONE 'Asia/Bangkok'::text), 'YYYY-MM'::text)), li.category
         Sort Method: quicksort  Memory: 27kB
         Buffers: shared hit=21
         ->  Hash Join  (cost=11.96..33.54 rows=50 width=44) (actual time=0.109..0.360 rows=56 loops=1)
               Hash Cond: (li.claim_id = c.id)
               Buffers: shared hit=21
               ->  Seq Scan on line_items li  (cost=0.00..19.36 rows=736 width=28) (actual time=0.009..0.096 rows=736 loops=1)
                     Buffers: shared hit=12
               ->  Hash  (cost=11.75..11.75 rows=17 width=24) (actual time=0.083..0.083 rows=21 loops=1)
                     Buckets: 1024  Batches: 1  Memory Usage: 10kB
                     Buffers: shared hit=9
                     ->  Seq Scan on claims c  (cost=0.00..11.75 rows=17 width=24) (actual time=0.015..0.076 rows=21 loops=1)
                           Filter: ((status = 'APPROVED'::claim_status) AND (assigned_reviewer_id = (InitPlan 1).col1))
                           Rows Removed by Filter: 229
                           Buffers: shared hit=9
 Planning:
   Buffers: shared hit=10
 Planning Time: 0.368 ms
 Execution Time: 0.492 ms
(28 rows)

```

### S3 summary: AverageDecisionHours (Finance Officer, no range)

```
                                                  QUERY PLAN
--------------------------------------------------------------------------------------------------------------
 Aggregate  (cost=12.13..12.15 rows=1 width=12) (actual time=0.336..0.338 rows=1 loops=1)
   Buffers: shared hit=8
   ->  Seq Scan on claims c  (cost=0.00..11.12 rows=100 width=16) (actual time=0.013..0.101 rows=100 loops=1)
         Filter: (status = ANY ('{APPROVED,REJECTED}'::claim_status[]))
         Rows Removed by Filter: 150
         Buffers: shared hit=8
 Planning:
   Buffers: shared hit=9
 Planning Time: 0.211 ms
 Execution Time: 0.368 ms
(10 rows)

```

### S3x summary: AverageDecisionHours (Finance Officer, range 2026-07-01 to 2026-11-30)

```
                                                                                                      QUERY PLAN
-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Aggregate  (cost=12.78..12.80 rows=1 width=12) (actual time=0.215..0.216 rows=1 loops=1)
   Buffers: shared hit=8
   ->  Seq Scan on claims c  (cost=0.00..12.38 rows=40 width=16) (actual time=0.015..0.158 rows=100 loops=1)
         Filter: ((status = ANY ('{APPROVED,REJECTED}'::claim_status[])) AND (decided_at >= '2026-06-30 17:00:00+00'::timestamp with time zone) AND (decided_at < '2026-11-30 17:00:00+00'::timestamp with time zone))
         Rows Removed by Filter: 150
         Buffers: shared hit=8
 Planning Time: 0.142 ms
 Execution Time: 0.241 ms
(8 rows)

```

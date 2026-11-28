# F-11 Receipt Files – Spec

Supersedes #61 (F-EXTRA-01), which was planned but never built. Decisions: `docs/adr/0002-receipt-files-in-s3-with-presigned-urls.md`. Vocabulary: `GLOSSARY.md` (**Receipt File**).

## Problem Statement

A requester proves each expense only with a typed receipt reference number. A reviewer or Finance Officer can't see the receipt itself, so they must trust the number or ask for the paper outside ClaimFlow. SRS D-09 and EX-05 left file upload out because file storage is not core to the workflow. The team now wants real receipt evidence on each line item.

## Solution

A requester can attach Receipt Files (JPEG, PNG, WebP, or PDF photos and scans, up to 10 MB each, at most five per line item) to any line item while the claim is editable: New Claim, Draft, or Returned. Anyone who can see the claim can open its Receipt Files from Claim Detail. The receipt reference stays required, and Receipt Files are optional.

The browser sends file bytes straight to an S3 bucket with short-lived presigned URLs. The API signs them only after its own checks, and the API never carries file bytes. Locally the bucket is RustFS, started by the compose stack. In production it is an AWS S3 bucket that env vars point to. The production image contains no storage server.

## User Stories

1. As a requester, I want to attach a photo of a receipt to a line item, so that my reviewer sees proof of payment.
2. As a requester, I want to attach a PDF receipt, so that I can use e-receipts and invoices as they arrive.
3. As a requester on a phone, I want to take the receipt photo with my camera from the picker, so that I don't need a separate step to save the photo.
4. As a requester, I want to attach up to five Receipt Files to one line item, so that multi-page receipts fit.
5. As a requester, I want the upload to start as soon as I pick a file, with a progress bar, so that I know when it is done.
6. As a requester, I want to remove a Receipt File before I submit, so that a wrong photo never reaches my reviewer.
7. As a requester, I want to cancel an upload that is still running, so that a slow, wrong file doesn't block me.
8. As a requester, I want an instant message when a file is too large or of the wrong type, so that I don't wait for an upload that will fail.
9. As a requester, I want the server to reject a file that only pretends to be an image or PDF, so that the claim holds only real receipts.
10. As a requester, I want my uploaded Receipt Files to stay on the form when the server rejects my claim for a rule error, so that I don't upload them again.
11. As a requester, I want a line item with no Receipt File to stay valid, so that a claim with only a receipt reference still works.
12. As a requester, I want to add or remove Receipt Files on a Draft, so that I can collect receipts before I submit.
13. As a requester, I want to add or remove Receipt Files on a Returned claim, so that I can fix what my reviewer asked for.
14. As a requester, I want the Receipt Files of a Draft I delete to be deleted too, so that nothing is left behind.
15. As a requester, I want the Receipt Files of a submitted claim to stay fixed, so that the reviewer decides on exactly what I sent.
16. As a requester, I want to see the Receipt Files of my own claims on Claim Detail, so that I can check what I sent.
17. As an assigned reviewer, I want to open each line item's Receipt Files from Claim Detail, so that I can check the amount and date before I decide.
18. As an assigned reviewer, I want a Receipt File to open in a new tab, so that I keep my place on the claim.
19. As an assigned reviewer, I want to still open the Receipt Files of a claim I already decided, so that I can answer questions later.
20. As a Finance Officer, I want to open the Receipt Files of any non-draft claim, so that I can check a high-value claim before the Final Decision.
21. As a reviewer who is not assigned, I want to be denied a claim's Receipt Files, so that receipts stay as private as the claim.
22. As a requester, I want other members to be unable to open my unlinked uploads, so that a file I picked by mistake leaks to nobody.
23. As a member, I want a download link to stop working after a few minutes, so that a copied link doesn't expose a receipt for long.
24. As an operator, I want unlinked uploads older than 24 hours and objects with no database row to be swept, so that the bucket doesn't fill with orphans.
25. As an operator, I want readiness to fail when the bucket can't be reached, so that a broken deploy shows up in the smoke test.
26. As a developer, I want `bun run deps:up` to start Postgres and RustFS with the bucket and its CORS ready, so that the dev servers work after one command.
27. As a developer, I want `docker compose up --build` to run the whole stack with Receipt Files working, so that the one-command local run (SR-07) still holds.
28. As an operator, I want one idempotent script to create the production bucket and its scoped access key, so that production storage is reproducible.
29. As an operator, I want the storage key pair held in Secret Manager, never in Git, so that SR-02 still holds.
30. As a grader, I want the SRS, feature inventory, SDS, and traceability matrix to describe F-11, so that the documents match the product.

## Implementation Decisions

### Storage

- One narrow storage interface, owned by the Receipt File module, with these operations: presign put, presign get, head object, read the first bytes of an object, delete object, list objects, and head bucket. One S3 adapter implements it with `aws-sdk-go-v2` (`service/s3` and `s3.PresignClient`). Don't use `feature/s3/manager`, which is deprecated, or `transfermanager`, which is still 0.x.
- Both SDK clients set `RequestChecksumCalculation` and `ResponseChecksumValidation` to `WhenRequired`, so RustFS and other S3-compatible stores accept the requests.
- Two clients share one config. The API calls the bucket through `S3_ENDPOINT`. URLs are signed for `S3_PUBLIC_ENDPOINT`, which defaults to `S3_ENDPOINT`, because the signed host must be the one the browser reaches. Path-style addressing is on only when an endpoint is set. In production both endpoints are unset, so the SDK uses the regional AWS endpoint.
- Config, in `.env.schema` and `.env.example`: `S3_BUCKET`, `S3_REGION`, `S3_ENDPOINT`, `S3_PUBLIC_ENDPOINT`, `AWS_ACCESS_KEY_ID` (sensitive), and `AWS_SECRET_ACCESS_KEY` (sensitive).
- Object keys are `receipts/<uuidv7>`, with no user input. There is one key per Receipt File, with no `pending/` prefix and no copy on complete.
- Readiness also runs head bucket, inside the existing readiness timeout.

### Receipt File lifecycle

- A Receipt File row holds: id, object key, uploader, original filename, content type, size, state (`UPLOADING` → `READY`), the line item it is linked to (nullable), and when it was created.
- Create: the requester sends the filename, the declared content type, and the declared size. The API checks the type against JPEG, PNG, WebP, and PDF, and the size against 1 byte to 10 MB. Failures are 415 or 413 with a field message (NFR-08). Then it inserts an `UPLOADING` row and returns a presigned PUT with `Content-Type` and `Content-Length` signed, a 5-minute expiry, and the headers the browser must send.
- Complete: the uploader asks the API to complete. The API runs head object to check that the stored size and type match the row. Then it reads the first bytes and sniffs the real type. On a match the row becomes `READY`. On a mismatch, or when the object is missing, the API deletes the object and row and returns 415 or 409.
- Link: line items in the claim create, update, and submit bodies gain `receiptFileIds`. Links are written in the same transaction as the claim change (NFR-06). Each ID is checked: it exists, the actor uploaded it, it is `READY`, no other line item uses it, and its line item ends up with five or fewer. A failure is a field error on that line item, and nothing is saved.
- Editable only while the claim is: unlinked, or linked to a `DRAFT` or `RETURNED` claim. Relinking or deleting a Receipt File on any other claim is 409 BR-03.
- Delete: the uploader deletes an unlinked or editable Receipt File. The row is removed in the transaction, and the object is deleted after commit. When the object delete fails, the row is still gone and the sweep removes the object later.
- Deleting a Draft deletes its Receipt Files the same way. Removing a Receipt File from a Returned claim writes no history entry of its own; the edit is the change.
- Withdraw, Reject, and Approve keep Receipt Files.

### Access

- New Route Permission `receipt:create`, granted to every role that holds `claim:create` (requester, reviewer). Finance gets none, because it cannot be a requester.
- Download: a linked Receipt File follows Visible Claim exactly. An unlinked one is visible only to its uploader. Anyone else gets 403 with no data, and an unknown ID gets 404.
- The download endpoint answers 302 to a presigned GET with a 5-minute expiry. The GET pins `response-content-type` to the stored type and `response-content-disposition` to `inline` with the original filename. The bucket stays private (Block Public Access on), so the file comes from the bucket's origin, never the app's.

### API contract (OpenAPI)

- `POST /receipt-files` `{filename, contentType, size}` → 201 `{id, uploadUrl, uploadHeaders, expiresAt}`. Needs `receipt:create`.
- `POST /receipt-files/{id}/complete` → 200 `ReceiptFile {id, filename, contentType, size}`. Uploader only.
- `DELETE /receipt-files/{id}` → 204. Uploader only, and only while editable.
- `GET /receipt-files/{id}/content` → 302 `Location` presigned GET.
- Line item input gains `receiptFileIds: uuid[]` (max 5). Line item output gains `receiptFiles: ReceiptFile[]`.

### Sweep

- A `sweep` subcommand next to `migrate` and `seed` deletes Receipt Files that are unlinked and older than 24 hours, objects with no row, and rows whose object is missing. It is safe to run again. It is run by hand; scheduling is out of scope.

### Web

- Each line-item row on New Claim and Edit Claim has a Receipt File picker: a file button or drop zone with `accept` for the four types, and camera capture allowed. Each file shows progress, a remove or cancel button, and a field message.
- The upload uses `XMLHttpRequest`, because `fetch` reports no upload progress. Remove aborts a running upload and deletes a finished one.
- Type and size are checked in the browser first, for fast feedback only. Only `READY` IDs go into the claim body. Uploaded files survive a server rule error.
- Claim Detail lists Receipt Files per line item. Each one opens the download endpoint in a new tab.

### Local and production infrastructure

- Compose adds `rustfs` (`rustfs/rustfs:1.0.7`, named volume, `/health` healthcheck, S3 port on loopback) and a one-shot `rustfs-init` (`amazon/aws-cli`). The init creates the bucket and sets CORS for `http://localhost:3000` and `http://localhost:8080` (PUT, GET, headers `content-type`, expose `ETag`). `app` waits for `rustfs-init`.
- `bun run db:up` becomes `bun run deps:up` and starts `db`, `rustfs`, and `rustfs-init`. There is no alias, and every reference is updated.
- A new idempotent `deploy/aws-storage.sh` (aws cli) creates:
  - the bucket in `ap-southeast-1`, with Block Public Access, SSE-S3, and versioning off
  - CORS for the Cloud Run URL
  - a lifecycle rule that aborts incomplete multipart uploads after 1 day
  - an IAM user whose policy allows only `s3:GetObject`, `s3:PutObject`, `s3:DeleteObject`, and `s3:ListBucket` on that bucket
- `deploy/setup.sh` and the deploy workflow put the key pair in Secret Manager and pass the `S3_*` and `AWS_*` env to the service and the migrate job.
- The seed creates no Receipt Files.

### Documents

- SRS: revise D-09 and EX-05 (file upload is in, OCR stays out) and add the Receipt File requirements. `features.md`: add the F-11 row and dependency edges (F-02, F-03, F-05, F-06). SDS: the storage component, the `receipt_files` table, the endpoints, and the permission. Update `traceability.md`. ADR 0002 is already written.

## Testing Decisions

- Test only external behavior through the highest seam: the API router with a real Postgres from `pgtest` and the real S3 adapter against an in-process `gofakes3` server. No hand-written storage fake, and no containers in Go tests. That keeps the adapter covered at 100% (`// coverage-ignore` only for real untestable glue).
- `gofakes3` does not check presigned signatures. Router tests check that the URL, headers, and expiry are issued, then write the object directly to simulate the browser PUT.
- Router tests cover every rule above: create checks, complete with match, mismatch, and missing object, the link rules and the five-per-item cap, frozen 409s, delete and Draft delete, the sweep, readiness with the bucket down, and the full authorization matrix (requester, assigned reviewer, unrelated reviewer, Finance Officer; linked versus unlinked; Draft versus submitted).
- Prior art: the existing `claims_*_test.go` router tests and `auth_routes_test.go` (the route inventory gains the new endpoints).
- Web component tests with a mocked API and a mocked XHR cover pick, progress, cancel, remove, the client-side messages, `receiptFileIds` in the body, survival after a rule error, and the Claim Detail list. Prior art: `features/claims/__test__`.
- Playwright `f11-receipt-files.spec.ts` against the seeded compose stack with RustFS is the only proof of the real presign, CORS, and RustFS path. A requester attaches a PNG and a PDF, submits, and the assigned reviewer opens both. CI has no Playwright job (#96), so it runs locally before release, per the Product DoD.
- Production storage is checked by hand: the deploy smoke test plus one upload and download on the deployed URL.

## Out of Scope

- OCR, thumbnails, resizing, HEIC, virus scanning, and file versions.
- Multipart or resumable uploads, and files over 10 MB.
- Browser-computed checksums (`x-amz-checksum-sha256`) and storing a SHA-256.
- Keyless GCP-to-AWS web identity federation.
- Scheduling the sweep (Cloud Scheduler, Cloud Run jobs).
- A `pending/` prefix lifecycle expiry.
- Seeded Receipt Files.
- A Playwright job in CI.

## Further Notes

- F-11 is part of Sprint 3. Tickets go to project 5 as `Backlog`, with squash-merged PRs into `sprint-3`.
- RustFS is pinned at 1.0.7 because its pre-1.0 history changed CORS behavior (bucket CORS works from 1.0.0-rc.2).
- `gofakes3` warns of breaking changes; pin it in `go.mod`.

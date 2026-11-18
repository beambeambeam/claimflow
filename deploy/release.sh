#!/usr/bin/env bash
# Releases one image to the existing Cloud Run resources (deploy.yml runs it; deploy/setup.sh created them).
#
#   PROJECT_ID, REGION, IMAGE_TAG (full commit SHA) are required. ROLLBACK=true skips the migration.
#
# 1. Resolves the tag to its digest, so the service runs exactly the image that was built for the commit.
# 2. Runs the migrate-and-seed job with that image and waits; a failure stops the release here.
# 3. Updates the service to the digest, with the settings every release must carry (#54): COOKIE_SECURE=true,
#    GIN_MODE=release, and a startup probe on /api/v1/ready, so a revision that cannot reach the database
#    never becomes ready and traffic stays on the previous one. Then sends all traffic to the new revision.
# In GitHub Actions it exports SERVICE_URL, DEPLOYED_IMAGE, and REVISION for the next steps.
set -euo pipefail

: "${PROJECT_ID:?PROJECT_ID is required}" "${REGION:?REGION is required}" "${IMAGE_TAG:?IMAGE_TAG is required}"
ROLLBACK="${ROLLBACK:-false}"
SERVICE="claimflow"
MIGRATE_JOB="claimflow-migrate"
IMAGE="$REGION-docker.pkg.dev/$PROJECT_ID/claimflow/claimflow"

[[ "$IMAGE_TAG" =~ ^[0-9a-f]{40}$ ]] || { echo "IMAGE_TAG must be a full 40-character commit SHA." >&2; exit 1; }
[[ "$ROLLBACK" == true || "$ROLLBACK" == false ]] || { echo "ROLLBACK must be true or false." >&2; exit 1; }

DIGEST_IMAGE="$(gcloud artifacts docker images describe "$IMAGE:$IMAGE_TAG" --project="$PROJECT_ID" \
  --format='value(image_summary.fully_qualified_digest)')"
[[ "$DIGEST_IMAGE" == "$IMAGE@sha256:"* ]] || { echo "No image for $IMAGE_TAG in Artifact Registry." >&2; exit 1; }
echo "Image: $DIGEST_IMAGE"

if [[ "$ROLLBACK" == false ]]; then
  gcloud run jobs update "$MIGRATE_JOB" --image="$DIGEST_IMAGE" --project="$PROJECT_ID" --region="$REGION" --quiet
  gcloud run jobs execute "$MIGRATE_JOB" --wait --project="$PROJECT_ID" --region="$REGION" --quiet
else
  echo "Rollback: the schema stays as it is (migrations are additive); no migration and no seed."
fi

gcloud run services update "$SERVICE" --image="$DIGEST_IMAGE" \
  --update-env-vars=COOKIE_SECURE=true,GIN_MODE=release \
  --startup-probe=httpGet.path=/api/v1/ready,httpGet.port=8080,periodSeconds=5,timeoutSeconds=3,failureThreshold=24 \
  --project="$PROJECT_ID" --region="$REGION" --quiet
gcloud run services update-traffic "$SERVICE" --to-latest --project="$PROJECT_ID" --region="$REGION" --quiet

SERVICE_URL="$(gcloud run services describe "$SERVICE" --project="$PROJECT_ID" --region="$REGION" --format='value(status.url)')"
REVISION="$(gcloud run services describe "$SERVICE" --project="$PROJECT_ID" --region="$REGION" \
  --format='value(status.latestReadyRevisionName)')"
echo "Service URL: $SERVICE_URL"
echo "Revision: $REVISION"

if [[ -n "${GITHUB_ENV:-}" ]]; then
  printf 'SERVICE_URL=%s\nDEPLOYED_IMAGE=%s\nREVISION=%s\n' "$SERVICE_URL" "$DIGEST_IMAGE" "$REVISION" >> "$GITHUB_ENV"
fi

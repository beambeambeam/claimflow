#!/usr/bin/env bash
# One-time GCP setup and first deploy of ClaimFlow (Lab 5, Student Guide Episode 5).
#
# Creates every resource the app needs, then runs the migrate-and-seed job and deploys the service:
#   1 APIs  2 Artifact Registry  3 Cloud SQL instance  4 database  5 runtime service account
#   6 database user and DATABASE_URL secret  7 SEED_USER_PASSWORD secret  8 secret access
#   9 image (tagged with the commit SHA)  10 migrate-and-seed job  11 Cloud Run service
#
# Safe to run again: every resource that exists is reused, and an existing secret is never replaced.
# Passwords are generated here, stored only in Secret Manager, and never printed.
#
# Run it in Cloud Shell, from the repository root, on a clean checkout of the `main` commit to deploy:
#   DRY_RUN=1 PROJECT_ID=<project-id> bash deploy/setup.sh   # print the plan, create nothing
#   PROJECT_ID=<project-id> bash deploy/setup.sh              # create and deploy (10 to 20 minutes)
#
# Cost: the Cloud SQL instance is billed for every hour it exists. docs/labs/lab-05/deployment.md has the
# resource names and the release procedure.
set -euo pipefail

DRY_RUN="${DRY_RUN:-0}"
REGION="${REGION:-asia-southeast1}"

if [ "$DRY_RUN" = "1" ]; then
  PROJECT_ID="${PROJECT_ID:-example-project}"
else
  PROJECT_ID="${PROJECT_ID:-$(gcloud config get-value project 2>/dev/null || true)}"
fi

if [ -z "$PROJECT_ID" ]; then
  echo "PROJECT_ID is not set and gcloud has no active project. Run: gcloud config set project <project-id>" >&2
  exit 1
fi

if [ "$DRY_RUN" != "1" ] && [ -n "$(git status --porcelain)" ]; then
  echo "The working tree has changes. The image must match a commit: commit, stash, or reset them first." >&2
  exit 1
fi

# Only an accepted release deploys (R-07): HEAD must be the latest commit of main.
if [ "$DRY_RUN" != "1" ]; then
  git fetch -q origin main
  if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]; then
    echo "HEAD is not origin/main. Check out the latest main commit to deploy: git switch main && git pull" >&2
    exit 1
  fi
fi

# Resource names. The deploy workflow (#54) uses the same ones.
APP="claimflow"
REPO="$APP"
SQL_INSTANCE="$APP-pg"
DB_NAME="$APP"
DB_USER="$APP"
SERVICE="$APP"
MIGRATE_JOB="$APP-migrate"
RUNNER="$APP-runtime"
RUNNER_EMAIL="$RUNNER@$PROJECT_ID.iam.gserviceaccount.com"
SECRET_DB_URL="$APP-database-url"
SECRET_SEED_PASSWORD="$APP-seed-password"
SQL_CONNECTION="$PROJECT_ID:$REGION:$SQL_INSTANCE"
SHA="$(git rev-parse HEAD)"
IMAGE="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO/$APP:$SHA"

# Prints a command, then runs it unless DRY_RUN=1. A --password value is never printed.
run() {
  local arg
  if [ "$DRY_RUN" = "1" ]; then printf '[DRY_RUN] '; else printf '+ '; fi
  for arg in "$@"; do
    case "$arg" in
      --password=*) printf '%s ' '--password=<REDACTED>' ;;
      *) printf '%q ' "$arg" ;;
    esac
  done
  printf '\n'
  if [ "$DRY_RUN" != "1" ]; then "$@"; fi
}

# Succeeds when the resource that the given describe command reads exists. In a dry run nothing exists, so
# the plan shows every create command.
exists() {
  [ "$DRY_RUN" != "1" ] && "$@" >/dev/null 2>&1
}

# Creates a secret and its first version in one call, so a run that dies midway never leaves a secret
# without a version. The value goes through stdin and is never printed.
create_secret() {
  local name="$1" value="$2"
  if [ "$DRY_RUN" = "1" ]; then
    echo "[DRY_RUN] gcloud secrets create $name --replication-policy=automatic --data-file=- (generated value, never printed)"
  else
    echo "+ gcloud secrets create $name --replication-policy=automatic --data-file=- (generated value, never printed)"
    printf '%s' "$value" | gcloud secrets create "$name" --replication-policy=automatic --data-file=- --project "$PROJECT_ID" >/dev/null
  fi
}

echo "ClaimFlow GCP setup: project $PROJECT_ID, region $REGION, commit $SHA, dry run $DRY_RUN"

# 1. APIs. IAM Credentials and STS are for Workload Identity Federation (Episode 6). The image is built
#    with Docker in Cloud Shell, so Cloud Build is not needed.
run gcloud services enable run.googleapis.com sqladmin.googleapis.com artifactregistry.googleapis.com \
  secretmanager.googleapis.com iam.googleapis.com iamcredentials.googleapis.com sts.googleapis.com \
  --project "$PROJECT_ID"

# 2. Artifact Registry repository for the images.
if ! exists gcloud artifacts repositories describe "$REPO" --location "$REGION" --project "$PROJECT_ID"; then
  run gcloud artifacts repositories create "$REPO" --repository-format=docker --location "$REGION" \
    --description="ClaimFlow images" --project "$PROJECT_ID"
fi

# 3. Cloud SQL for PostgreSQL 17, the same major version as local and tests. db-f1-micro needs the
#    Enterprise edition. Storage auto-increase is off from the start: a grown disk never shrinks.
if ! exists gcloud sql instances describe "$SQL_INSTANCE" --project "$PROJECT_ID"; then
  run gcloud sql instances create "$SQL_INSTANCE" --database-version=POSTGRES_17 --edition=enterprise \
    --tier=db-f1-micro --region="$REGION" --storage-size=10GB --storage-type=SSD --no-storage-auto-increase \
    --no-backup --project "$PROJECT_ID"
fi

# 4. Database.
if ! exists gcloud sql databases describe "$DB_NAME" --instance="$SQL_INSTANCE" --project "$PROJECT_ID"; then
  run gcloud sql databases create "$DB_NAME" --instance="$SQL_INSTANCE" --project "$PROJECT_ID"
fi

# 5. Runtime service account: it can only connect to Cloud SQL, and read the two secrets (step 8).
if ! exists gcloud iam service-accounts describe "$RUNNER_EMAIL" --project "$PROJECT_ID"; then
  run gcloud iam service-accounts create "$RUNNER" --display-name="ClaimFlow runtime" --project "$PROJECT_ID"
fi
run gcloud projects add-iam-policy-binding "$PROJECT_ID" --member="serviceAccount:$RUNNER_EMAIL" \
  --role=roles/cloudsql.client --condition=None --quiet --format=none

# 6. Database user and DATABASE_URL. The API connects through the Cloud SQL unix socket that Cloud Run
#    mounts at /cloudsql/<connection name>; pgx reads the socket folder from the `host` parameter.
if ! exists gcloud secrets describe "$SECRET_DB_URL" --project "$PROJECT_ID"; then
  DB_PASSWORD="$(openssl rand -hex 24)"
  if exists gcloud sql users describe "$DB_USER" --instance="$SQL_INSTANCE" --project "$PROJECT_ID"; then
    run gcloud sql users set-password "$DB_USER" --instance="$SQL_INSTANCE" --password="$DB_PASSWORD" \
      --project "$PROJECT_ID"
  else
    run gcloud sql users create "$DB_USER" --instance="$SQL_INSTANCE" --password="$DB_PASSWORD" \
      --project "$PROJECT_ID"
  fi
  create_secret "$SECRET_DB_URL" \
    "postgres://$DB_USER:$DB_PASSWORD@localhost/$DB_NAME?host=/cloudsql/$SQL_CONNECTION&sslmode=disable"
fi

# 7. SEED_USER_PASSWORD: the password of the 7 demo users. Share it with the Lab 6 partner team privately.
if ! exists gcloud secrets describe "$SECRET_SEED_PASSWORD" --project "$PROJECT_ID"; then
  create_secret "$SECRET_SEED_PASSWORD" "$(openssl rand -base64 18 | tr -d '/+=')"
fi

# 8. Secret access for the runtime account, per secret and never project-wide.
for secret in "$SECRET_DB_URL" "$SECRET_SEED_PASSWORD"; do
  run gcloud secrets add-iam-policy-binding "$secret" --member="serviceAccount:$RUNNER_EMAIL" \
    --role=roles/secretmanager.secretAccessor --project "$PROJECT_ID" --quiet --format=none
done

# 9. Image: the root Dockerfile, with APP_VERSION set to the commit SHA so /api/v1/health reports it.
#    The services below run it by digest, so the running image is exactly the one built here.
run gcloud auth configure-docker "$REGION-docker.pkg.dev" --quiet
run docker build --build-arg "APP_VERSION=$SHA" -t "$IMAGE" .
run docker push "$IMAGE"
if [ "$DRY_RUN" = "1" ]; then
  IMAGE_REF="${IMAGE%:*}@sha256:<digest>"
else
  IMAGE_REF="${IMAGE%:*}@$(gcloud artifacts docker images describe "$IMAGE" --format='value(image_summary.digest)')"
fi

# 10. Migrate, then seed, as a Cloud Run job that finishes before the service changes. The API never migrates
#     on startup. Both commands are safe to run again.
run gcloud run jobs deploy "$MIGRATE_JOB" --image "$IMAGE_REF" --region "$REGION" \
  --service-account "$RUNNER_EMAIL" --set-cloudsql-instances "$SQL_CONNECTION" \
  --set-secrets "DATABASE_URL=$SECRET_DB_URL:latest,SEED_USER_PASSWORD=$SECRET_SEED_PASSWORD:latest" \
  --command sh --args "-c,varlock run -- /app/api migrate up && varlock run -- /app/api seed" \
  --max-retries 0 --task-timeout 600s --project "$PROJECT_ID"

if ! run gcloud run jobs execute "$MIGRATE_JOB" --region "$REGION" --wait --project "$PROJECT_ID"; then
  echo "The $MIGRATE_JOB job failed, so the service was not deployed. Its log:" >&2
  echo "  gcloud logging read 'resource.type=cloud_run_job AND resource.labels.job_name=$MIGRATE_JOB' --project $PROJECT_ID --limit 50 --format='value(textPayload,jsonPayload.message)'" >&2
  exit 1
fi

# 11. The service: public for the demo, scales to zero, at most 2 instances. Cloud Run sets PORT=8080.
run gcloud run deploy "$SERVICE" --image "$IMAGE_REF" --region "$REGION" --allow-unauthenticated \
  --service-account "$RUNNER_EMAIL" --add-cloudsql-instances "$SQL_CONNECTION" \
  --set-secrets "DATABASE_URL=$SECRET_DB_URL:latest" \
  --min-instances 0 --max-instances 2 --memory 512Mi --cpu 1 --port 8080 --project "$PROJECT_ID"

if [ "$DRY_RUN" = "1" ]; then
  URL="https://$SERVICE-<hash>.$REGION.run.app"
else
  URL="$(gcloud run services describe "$SERVICE" --region "$REGION" --project "$PROJECT_ID" --format='value(status.url)')"
fi

echo
echo "Done. Service URL: $URL"
echo "  curl -s $URL/api/v1/health   # version must be $SHA"
echo "  curl -s $URL/api/v1/ready"
echo "The demo user password is in Secret Manager ($SECRET_SEED_PASSWORD); it is not printed."

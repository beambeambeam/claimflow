# Receipt Files live in S3, and browsers move the bytes with presigned URLs

Receipt Files are stored in an S3 bucket through `aws-sdk-go-v2`: RustFS in the local compose stack, and an AWS S3 bucket in production, even though the app runs on Cloud Run. The browser uploads and downloads with short-lived presigned URLs that the API signs only after its own checks. The API never carries file bytes. One S3 API covers both environments with the same config shape (endpoint, region, bucket, key pair), and Cloud Run is not loaded with file traffic or bound by its request size limit.

**Considered Options**

- Upload through the API with Go CDK, SeaweedFS locally and Cloud Storage in production (the #61 plan). Rejected: two storage backends behind an abstraction, and every byte passes through Cloud Run.
- Cloud Storage through its S3-compatible XML API with HMAC keys. Rejected: S3 compatibility gaps, and the team wanted AWS S3 itself.
- Keyless access from Cloud Run to AWS with web identity federation. Rejected for now: much more setup than one bucket-scoped access key held in Secret Manager.

**Consequences**

- The bucket needs CORS rules for the web origin, locally and in production.
- An upload takes two calls: the browser PUTs to the presigned URL, then asks the API to complete it. The API checks the stored object's size and first bytes before the Receipt File can be linked.
- Objects can be orphaned, so a sweep deletes unlinked Receipt Files and objects with no database row.
- The production image contains no storage server. Only env vars point it at the bucket.

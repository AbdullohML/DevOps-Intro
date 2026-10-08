# Lab 10 — Cloud Computing

## Task 1 — CI-Automated Push to GHCR

### Release workflow

The release workflow is stored at:

../.github/workflows/release.yml

It triggers on tags matching `v*`, builds QuickNotes from `app/`, and pushes both the version tag and `latest`.

The workflow uses only:

- contents: read
- packages: write

The external checkout action is pinned by its full 40-character SHA.

### Registry

Image:

ghcr.io/abdullohml/devops-intro/quicknotes

Release tags tested:

- v0.1.0
- v0.1.1
- latest

### Public pull evidence

The image was pulled using an empty Docker configuration with no GHCR authentication:

    sudo env DOCKER_CONFIG=/tmp/lab10-docker-clean \
      docker pull ghcr.io/abdullohml/devops-intro/quicknotes:v0.1.0

Result:

    v0.1.0: Pulling from abdullohml/devops-intro/quicknotes
    Digest: sha256:55218e9c04458f0db388e83e85d361eb39472102f1e0899608b684ffa5326be9
    Status: Downloaded newer image for ghcr.io/abdullohml/devops-intro/quicknotes:v0.1.0

The `latest` tag was also successfully pulled without authentication.

### Green release workflow

https://github.com/AbdullohML/DevOps-Intro/actions/runs/37844409567

The v0.1.1 workflow successfully:

1. built the image
2. pushed v0.1.1
3. pushed latest
4. invoked the Render deploy hook

### Design question a — OIDC vs GITHUB_TOKEN

For pushing a package to GHCR from the same GitHub repository, GITHUB_TOKEN with `packages: write` is sufficient because GitHub issues the temporary token automatically for the workflow.

OIDC is more useful when GitHub Actions needs to authenticate to an external cloud provider such as AWS, GCP, or Azure. The provider can trust GitHub's workload identity and issue short-lived credentials instead of requiring a long-lived access key stored in GitHub Secrets.

### Design question b — Why publish latest and an immutable version?

The immutable version tag such as `v0.1.1` is used for reproducibility, rollback, auditing, and deployments where the exact artifact must not change.

`latest` is still useful as a convenient pointer to the newest release for developers and users who just want the current version.

Production deployment should prefer an immutable version or digest when reproducibility matters.

### Design question c — Why only packages: write?

This follows the principle of least privilege.

The release workflow only needs to read repository content and publish packages, so it does not need broad write permissions.

If the workflow or one of its dependencies were compromised, a narrow token limits the attacker to the capabilities required by this job instead of allowing unrelated repository modifications through permissions such as `write: all`.

---

## Task 2 — Render Deployment

### Option used

Option A: Render Free Web Service.

Render did not require card verification, so the Codespaces fallback was not needed.

### Source choice

Render uses the existing CI-built image:

ghcr.io/abdullohml/devops-intro/quicknotes:v0.1.0

I chose the existing image instead of asking Render to rebuild the repository because this deploys the same artifact produced by CI and hardened/scanned in Lab 9.

Configuration details are documented in:

../cloud/render.md

### Render configuration

- Service: quicknotes-lab10
- Region: Frankfurt
- Instance type: Free
- PORT=10000
- ADDR=:10000
- Health check path: /health

Public URL:

https://quicknotes-lab10-0x46.onrender.com

### Public health check

A verbose curl request returned HTTP 200:

    GET /health HTTP/2
    HTTP/2 200
    content-type: application/json
    cache-control: no-store

    {"notes":0,"status":"ok"}

The `/notes` endpoint also returned successfully:

    []

### Port evidence

Render startup log:

    quicknotes listening on :10000 (notes loaded: 0)
    Your service is live

No port mismatch restart was required.

### Automated Render deployment

The Render deploy hook is stored in GitHub Actions as:

    RENDER_DEPLOY_HOOK_URL

The actual URL is stored as a GitHub repository secret and is not committed.

The release workflow sends the newly published image tag to the hook.

The v0.1.1 release caused Render to deploy:

    ghcr.io/abdullohml/devops-intro/quicknotes:v0.1.1

Render deployment result:

    Trigger: Deploy Hook
    Deploy succeeded

### Warm latency

Five consecutive requests:

    4.445227 s
    0.478830 s
    0.459259 s
    0.382967 s
    0.381257 s

Warm p50:

    0.459259 s

### Cold-start latency

Each valid cold measurement was made after more than 20 minutes without requests.

    Cold 1: 13.323672 s
    Cold 2: 14.706560 s
    Cold 3: 14.595189 s

All three cold requests were much slower than the warm p50 because the free Render service had to wake from its idle state.

### Note persistence test

Before spin-down, a note was created:

    {
      "id": 1,
      "title": "render-persistence-test",
      "body": "created before spin-down"
    }

Before sleeping, GET /notes returned the note.

After more than 20 minutes idle and the next cold start:

    []

The note disappeared.

QuickNotes stores its notes in the container filesystem. Render's free service filesystem is ephemeral, so the locally stored note is lost when the instance is replaced/restarted during the sleep/wake lifecycle.

### Design question d — Render spin-down vs Cloud Run scale-to-zero

Both systems avoid keeping compute continuously active when there is no traffic.

Render's free service takes much longer to wake because it is optimized as a general free hosted web service and may need to provision and start more of the runtime environment again.

Cloud Run is designed specifically for serverless container execution and keeps infrastructure optimized for fast request-driven instance startup, so its cold starts are normally much shorter.

Render optimizes for inexpensive/free general hosting, while Cloud Run optimizes for elastic serverless execution.

### Design question e — PORT and ADDR

Render injects `PORT` because the hosting platform controls how external traffic is routed to the application.

Docker `EXPOSE` is only image metadata and does not guarantee which port the hosting platform will route to.

For this service I configured:

    PORT=10000
    ADDR=:10000

This ensures Render and QuickNotes agree from the first boot.

If QuickNotes listened on a different port, Render could detect the actual listening port and restart the deployment. That adds unnecessary deployment time and makes the configuration less deterministic.

### Design question f — Existing image vs repository build

Using the existing GHCR image gives stronger reproducibility because Render runs exactly the same artifact produced by CI and scanned in Lab 9.

If Render instead builds directly from the repository, the build is convenient and may benefit from platform caching, but it creates another build path. That deployed artifact could differ from the image already tested and scanned by CI.

For this lab, using the existing versioned image keeps build, security scanning, registry publication, and deployment connected to the same artifact.

The persistence experiment also demonstrates an important limitation: QuickNotes writes notes to its local filesystem. The Render free instance has ephemeral local storage, so data written there does not survive the instance lifecycle. Persistent application data would need external durable storage such as a managed database or persistent disk.

---

## Final result

- Tagged release triggers GitHub Actions
- Image pushed to GHCR as version tag and latest
- GHCR image publicly pullable without authentication
- Render Free service publicly serves QuickNotes
- PORT and ADDR correctly configured
- /health and /notes verified
- CI release automatically redeploys Render through a secret deploy hook
- Warm p50 measured: 0.459259 s
- Three cold starts measured
- Ephemeral note behavior demonstrated
- Design questions a-f answered

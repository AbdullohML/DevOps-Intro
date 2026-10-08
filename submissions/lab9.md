# Lab 9 — DevSecOps: Trivy + OWASP ZAP

## Task 1 — Trivy

Trivy version: `aquasec/trivy:0.59.1`

Artifacts:

- [Initial image scan](../security/trivy/image.txt)
- [Final image scan](../security/trivy/image-after.txt)
- [Filesystem scan](../security/trivy/filesystem.txt)
- [Configuration scan](../security/trivy/config.txt)
- [CycloneDX SBOM](../security/trivy/quicknotes.cdx.json)

### Image scan — before fix

The original `quicknotes:lab6` image used Go `v1.24.13`.

```text
quicknotes:lab6 (debian 12.15)
Total: 0 (HIGH: 0, CRITICAL: 0)

healthcheck (gobinary)
Total: 19 (HIGH: 19, CRITICAL: 0)

quicknotes (gobinary)
Total: 19 (HIGH: 19, CRITICAL: 0)
```

The same 19 Go standard-library CVEs appeared in both Go binaries.

### HIGH/CRITICAL image findings triage

All 19 unique vulnerabilities were remediated by changing the builder image from:

```text
golang:1.24-alpine
```

to:

```text
golang:1.26.6-alpine
```

Fix commit: `887f01b`

| Finding | Severity | Affected components | Disposition | Reason |
|---|---|---|---|---|
| CVE-2026-25679 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-27145 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-32280 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-32281 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-32283 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-33811 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-33814 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-33818 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-39820 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-39821 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-39822 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-39836 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-42499 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-42504 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-56853 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-56858 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-56859 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-56860 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |
| CVE-2026-56862 | HIGH | quicknotes, healthcheck | FIX | Upgrade Go to 1.26.6; fixed in `887f01b`. |

### Image scan — after fix

```text
quicknotes:lab6 (debian 12.15)
==============================
Total: 0 (HIGH: 0, CRITICAL: 0)
```

### Filesystem scan

```text
.vagrant/machines/default/virtualbox/private_key (secrets)

Total: 1 (HIGH: 1, CRITICAL: 0)

HIGH: AsymmetricPrivateKey (private-key)
```

Triage:

| Finding | Severity | Disposition | Reason |
|---|---|---|---|
| `.vagrant/machines/default/virtualbox/private_key` | HIGH | ACCEPT | This is a locally generated Vagrant VM key. `.vagrant/` is ignored by Git and the key is not part of the repository or QuickNotes image. Re-evaluate by 2027-04-08, and remove/rotate it if the VM directory is ever shared. |

Verification:

```text
.gitignore:27:.vagrant/    .vagrant/machines/default/virtualbox/private_key
```

### Configuration scan

```text
app/Dockerfile (dockerfile)

Tests: 28 (SUCCESSES: 27, FAILURES: 1)
Failures: 1 (UNKNOWN: 0, LOW: 1, MEDIUM: 0, HIGH: 0, CRITICAL: 0)

AVD-DS-0026 (LOW): Add HEALTHCHECK instruction in your Dockerfile
```

There are no HIGH or CRITICAL configuration findings.

The LOW finding is not part of the required HIGH/CRITICAL triage. Runtime health checking is already configured for QuickNotes in the Compose setup from the previous container lab.

## CycloneDX SBOM — first 30 lines
```json
{
  "$schema": "http://cyclonedx.org/schema/bom-1.6.schema.json",
  "bomFormat": "CycloneDX",
  "specVersion": "1.6",
  "serialNumber": "urn:uuid:91636e66-7cf4-439e-8951-b96083c43550",
  "version": 1,
  "metadata": {
    "timestamp": "2026-10-08T17:44:08+00:00",
    "tools": {
      "components": [
        {
          "type": "application",
          "group": "aquasecurity",
          "name": "trivy",
          "version": "0.59.1"
        }
      ]
    },
    "component": {
      "bom-ref": "pkg:oci/quicknotes@sha256%3Aad239050e2e842d80200af867668d206e48cf0de3eb065554a3e3644ca846263?arch=amd64&repository_url=index.docker.io%2Flibrary%2Fquicknotes",
      "type": "container",
      "name": "quicknotes:lab6",
      "purl": "pkg:oci/quicknotes@sha256%3Aad239050e2e842d80200af867668d206e48cf0de3eb065554a3e3644ca846263?arch=amd64&repository_url=index.docker.io%2Flibrary%2Fquicknotes",
      "properties": [
        {
          "name": "aquasecurity:trivy:DiffID",
          "value": "sha256:114dde0fefebbca13165d0da9c500a66190e497a82a53dcaabc3172d630be1e9"
        },
        {
          "name": "aquasecurity:trivy:DiffID",
```

### Design questions

#### a) Why is CVE severity not enough?

Severity describes the potential impact of a vulnerability, but triage also needs context.

Important factors include whether our code can actually reach the vulnerable functionality, whether a working exploit exists, whether an attacker can access that functionality, whether the service is internet-facing, and what privileges the affected process has.

A HIGH CVE that is unreachable may be lower priority than a lower-severity issue directly exposed to untrusted users.

#### b) Why is a minimal/distroless base image such a strong security control?

A minimal image contains fewer packages, tools, shells, and libraries.

That means there are fewer components that can contain vulnerabilities and fewer tools available to an attacker after compromise.

Reducing the attack surface prevents whole classes of findings instead of trying to patch each unnecessary package individually.

#### c) When should `.trivyignore` be used?

It is appropriate when a finding has been investigated and there is a documented reason to suppress it, for example a verified false positive or a temporarily accepted risk with an owner and review date.

It becomes security theater when findings are added to `.trivyignore` simply to make the scan green without investigating whether the vulnerability is reachable or relevant.

#### d) What future problem does an SBOM solve?

An SBOM gives an inventory of the exact components and versions shipped in an artifact.

When a vulnerability such as Log4Shell is announced later, the organization can immediately search its SBOMs and determine which released services contain the affected component instead of manually investigating every application.

---

# Task 2 — OWASP ZAP Baseline

ZAP image:

```text
ghcr.io/zaproxy/zaproxy:2.16.1
```

Only `zap-baseline.py` was used. No active scan was run.

Artifacts:

- [Before HTML report](../security/zap/before.html)
- [Before JSON report](../security/zap/before.json)
- [Before console output](../security/zap/before.txt)
- [After HTML report](../security/zap/after.html)
- [After JSON report](../security/zap/after.json)
- [After console output](../security/zap/after.txt)

## Before-fix findings

```text
10116 | ZAP is Out of Date | Low (High) |
http://127.0.0.1:8080/robots.txt

10049-3 | Storable and Cacheable Content | Informational (Medium) |
http://127.0.0.1:8080
http://127.0.0.1:8080/robots.txt
http://127.0.0.1:8080/sitemap.xml
```

### ZAP triage

| ID | Finding | Risk | Affected URL | Disposition | Reason |
|---|---|---|---|---|---|
| 10049-3 | Storable and Cacheable Content | Informational | `/`, `/robots.txt`, `/sitemap.xml` | FIX | Responses could be stored by caches. Added global `Cache-Control: no-store` middleware. Fix commit `887f01b`. |
| 10116 | ZAP is Out of Date | Low | `/robots.txt` | ACCEPT | This finding concerns the pinned ZAP scanner, not QuickNotes. Lab execution intentionally uses the fixed `2.16.1` image for reproducibility. Re-evaluate scanner version by 2027-04-08. |

## Code fix

Security behavior is implemented as middleware wrapping the complete router:

```go
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
```

`Routes()` returns the wrapped router, so the middleware applies to both registered handlers and unmatched routes such as 404 responses.

A unit test verifies the header on both `/health` and `/does-not-exist`.

```text
go test ./...
ok      quicknotes
```

Manual verification:

```text
HTTP/1.1 404 Not Found
Cache-Control: no-store
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff

404 page not found
```

## Before / after ZAP evidence

Before:

```text
WARN-NEW: Storable and Cacheable Content [10049] x 3
```

After:

```text
WARN-NEW: Non-Storable Content [10049] x 3
```

The original `10049-3 Storable and Cacheable Content` finding no longer appears after the fix.

The after-scan reports the different informational rule `10049-1 Non-Storable Content`, which describes the intentional `Cache-Control: no-store` behavior.

### After-scan findings

| ID | Finding | Risk | Disposition | Reason |
|---|---|---|---|---|
| 10049-1 | Non-Storable Content | Informational | ACCEPT | Expected result of the intentional `Cache-Control: no-store` security policy. |
| 10116 | ZAP is Out of Date | Low | ACCEPT | Scanner-version warning, not an application vulnerability; pinned version retained for reproducibility. |

### Design questions

#### e) Why middleware instead of setting the header in every handler?

Middleware gives one central security policy for every route.

Per-handler settings are easy to forget when a new endpoint is added and can lead to inconsistent behavior. Wrapping the router automatically applies the policy to existing routes, future routes, and error responses.

#### f) What does `Content-Security-Policy: default-src 'none'` break?

`default-src 'none'` blocks loading resources such as JavaScript, CSS, images, fonts, frames, and other external content unless explicitly allowed.

That is generally acceptable for a JSON API like QuickNotes because it does not need to render browser resources.

For a normal website it would usually break the frontend, so the policy would need explicit allowlists for required resource types and origins.

#### g) Why not mark every informational ZAP finding as accepted?

An informational finding can still reveal a real weakness or useful security signal.

Automatically accepting everything without reading it creates a blind spot and makes the scan meaningless. Each finding should be understood first and then fixed, accepted, suppressed, or marked as a false positive with a reason.

---

## Final result

- Trivy image scan: `0 HIGH / 0 CRITICAL`
- Filesystem HIGH finding investigated and documented
- Configuration scan: `0 HIGH / 0 CRITICAL`
- CycloneDX SBOM generated
- ZAP passive baseline run before and after the change
- All ZAP findings triaged
- Cacheability finding fixed with router middleware
- Unit test added and passing
- Fix commit: `887f01b`

# Lab 3 — CI/CD: A PR-Gated Pipeline for QuickNotes

## Chosen path

**GitHub Actions**

## Task 1 — PR-gated CI

### 1.1 CI configuration

The CI workflow is located at:

`.github/workflows/ci.yml`

It is triggered by:

* pushes to `main`
* pull requests targeting `main`

The workflow contains three independent checks:

* `vet` — `go vet ./...`
* `test` — `go test -race -count=1 ./...`
* `lint` — `golangci-lint run`

The Go module is located in `app/`, so the workflow uses `app/` as the working directory.

The runtime environment is pinned to `ubuntu-24.04`, and Go is pinned to specific versions rather than using `latest`.

All third-party GitHub Actions are pinned to full commit SHAs.

The workflow also uses least-privilege permissions:

```yaml
permissions:
  contents: read
```

### 1.2 Working CI

Pull request:

https://github.com/AbdullohML/DevOps-Intro/pull/5

Green CI run:

https://github.com/AbdullohML/DevOps-Intro/actions/runs/34890308982

The final CI run completed successfully in 41 seconds.

The final pipeline contains:

* `ci / vet (1.23)`
* `ci / vet (1.24)`
* `ci / test (1.23)`
* `ci / test (1.24)`
* `ci / lint`
* `ci / ci-ok`

The `ci / ci-ok` check is required by the branch protection ruleset.

### 1.3 Deliberate CI failure

To prove that the PR gate detects broken code, I deliberately changed the expected HTTP status in `app/handlers_test.go`.

The correct assertion was:

```go
if rec.Code != http.StatusCreated {
```

I temporarily changed it to:

```go
if rec.Code != http.StatusOK {
```

The test then failed because the endpoint correctly returned HTTP 201.

Failed CI run:

https://github.com/AbdullohML/DevOps-Intro/actions/runs/34886236027

The relevant failure was:

```text
--- FAIL: TestCreateNote_RoundTrip
    handlers_test.go:64: expected 200, got 201: {"id":1,"title":"first","body":"hello","created_at":"2026-09-14T19:20:24.884244069Z"}
FAIL
FAIL quicknotes 0.016s
Error: Process completed with exit code 1
```

The incorrect assertion was then reverted and CI became green again.

### 1.4 Branch protection

The `main` branch is protected by the `protect-main` ruleset.

The ruleset requires:

* pull request before merging
* required status checks
* branches to be up to date before merging
* force pushes blocked

The required CI gate is:

`ci / ci-ok`

The branch protection configuration is shown in:

`rule.png`

A direct push to `main` was also tested. GitHub rejected it with:

```text
GH013: Repository rule violations found for refs/heads/main.

Changes must be made through a pull request.

3 of 3 required status checks are expected.
```

This confirms that direct pushes to `main` are blocked.

## Task 1 — Design questions

### a) Why pin `ubuntu-24.04` instead of using `ubuntu-latest`?

`ubuntu-latest` is a moving target. GitHub can change the Ubuntu version behind that label, which can change system packages, tools, libraries, or other parts of the CI environment.

Pinning `ubuntu-24.04` makes the environment predictable and reproducible. A future GitHub runner migration will not unexpectedly change the environment in which the project is tested.

### b) Why split vet, test and lint into independent jobs?

Independent jobs make failures easier to identify and allow the checks to run in parallel.

For example, if lint fails, it is immediately clear that the problem is related to linting. With one combined job, all checks would be in the same execution unit and a failure could prevent later checks from running.

Separate jobs also provide clearer status checks for the pull request.

### c) What real attack does SHA pinning prevent?

Pinning an action to a full commit SHA protects against a mutable tag or branch being changed to point to malicious code.

For example:

```yaml
uses: some/action@v1
```

depends on the current meaning of the `v1` tag.

A full commit SHA identifies one exact revision, so the workflow continues to execute the reviewed code even if a tag is later changed.

This is particularly important in light of the March 2025 `tj-actions/changed-files` compromise, where a compromised GitHub Action version could expose secrets from workflows using the affected action.

### d) What is `permissions:` and why use least privilege?

`permissions:` controls the permissions granted to the workflow's `GITHUB_TOKEN`.

This workflow only needs to read repository contents, so it uses:

```yaml
permissions:
  contents: read
```

This follows the principle of least privilege: a workflow should receive only the permissions necessary for its job.

If a workflow or one of its actions is compromised, limiting its permissions reduces the possible impact.

### e) GitLab: stage vs job; dependencies vs stages

A GitLab stage defines a broad phase of the pipeline. Jobs belonging to the same stage can run in parallel.

A job is an individual unit of work, such as running tests or linting.

Stages control the overall ordering of the pipeline, while dependencies can define which specific jobs a job depends on and which artifacts it receives.

# Task 2 — Faster and more selective CI

## 2.1 — Go module and build cache

Go caching was enabled using `actions/setup-go`:

```yaml
with:
  go-version: '1.24'
  cache: true
```

Caching is also enabled for the matrix jobs.

The cache is intended to reuse Go module and build cache data instead of repeating the same work on every run.

For this project, the speed improvement is limited because QuickNotes has essentially no third-party Go dependencies.

## 2.2 — Go version matrix

The `vet` and `test` jobs now run against both Go 1.23 and Go 1.24.

The matrix is configured as:

```yaml
strategy:
  fail-fast: false
  matrix:
    go: ['1.23', '1.24']
```

This produces separate checks for each Go version:

* `ci / vet (1.23)`
* `ci / vet (1.24)`
* `ci / test (1.23)`
* `ci / test (1.24)`

Lint remains a separate job using Go 1.24.

### Why `fail-fast: false`?

With `fail-fast: false`, a failure in one matrix job does not cancel the other matrix jobs.

This is useful for compatibility testing because it allows us to see whether a failure is specific to one Go version or affects all supported versions.

For example, if Go 1.23 fails but Go 1.24 passes, both results are still available.

`fail-fast: true` would be useful when the matrix is expensive and the remaining jobs provide little value after an early failure.

### Aggregation job

The matrix changes the names of the status checks, so the workflow also contains a stable `ci-ok` aggregation job.

It depends on:

```yaml
needs: [vet, test, lint]
```

and uses:

```yaml
if: always()
```

This ensures that the aggregation job is evaluated even when one of its dependencies fails.

It fails if any required job fails or is cancelled.

The branch protection ruleset requires:

`ci / ci-ok`

This prevents the matrix check names from breaking the branch protection configuration.

## 2.3 — Path filtering

The workflow only runs when files under `app/` or the CI workflow itself are changed:

```yaml
paths:
  - 'app/**'
  - '.github/workflows/ci.yml'
```

Therefore:

* changes under `app/` trigger CI
* changes to `.github/workflows/ci.yml` trigger CI
* README-only changes do not independently match the workflow paths

The README test was performed on the existing pull request. Since that pull request already contains changes under `app/` and `.github/workflows/ci.yml`, GitHub continued to run the workflow. This is expected because path filtering considers the files changed by the pull request as a whole, not only the newest commit.

## 2.4 — Wall-clock measurements

The following successful runs were used for comparison.

| Configuration                  | Successful runs |   Median |
| ------------------------------ | --------------: | -------: |
| Baseline — no cache, no matrix |   30s, 30s, 41s |  **30s** |
| Cache — no matrix              |             60s | **60s*** |
| Cache + matrix                 |   45s, 46s, 41s |  **45s** |

* Only one successful cache-only measurement was available, so this is not a statistically strong median.

### Observations

Caching did not improve the observed wall-clock time.

This is expected for QuickNotes because the project has essentially no third-party Go dependencies, so there is very little dependency-download work to eliminate.

The matrix also does not reduce the total amount of work performed. Its main benefit is testing compatibility across multiple Go versions while allowing the jobs to run in parallel.

The measurements suggest that a significant part of the wall-clock time comes from GitHub Actions runner provisioning, checkout, Go setup, and general CI overhead rather than dependency installation.

## Task 2.5 — Optimization and security questions

### f) Why cache Go module inputs rather than build outputs?

Build outputs are generated from source code and the build environment. They can become invalid whenever source files, dependencies, or the environment change.

Go module and build cache data can be reused when the relevant inputs have not changed.

Caching these inputs and intermediate data makes the cache more reproducible and avoids treating previously generated final binaries as trusted build outputs.

### g) What does `fail-fast: false` change and when would `true` be useful?

`fail-fast: false` allows all matrix jobs to continue even when one matrix job fails.

This is useful when we want complete information about compatibility across all supported Go versions.

`fail-fast: true` can be useful when the matrix is large or expensive and there is little value in running the remaining jobs after an early failure.

### h) What is the cache security risk?

An untrusted pull request could potentially attempt to influence or poison cached data.

If a protected branch later restored attacker-controlled cache contents and treated them as trusted, malicious files or artifacts could potentially affect the trusted workflow.

Therefore caches should not be treated as trusted executable sources. Cache scopes and restore behavior should prevent untrusted pull requests from supplying data that protected branches blindly trust.

## Final CI architecture

```text
                         ┌── vet (1.23) ──┐
                         ├── vet (1.24) ──┤
Pull Request ────────────┼── test (1.23) ─┼──> ci-ok ──> required gate
                         ├── test (1.24) ─┤
                         └── lint ────────┘
```

The final pipeline provides:

* pinned Ubuntu runner version
* pinned Go versions
* SHA-pinned GitHub Actions
* least-privilege permissions
* independent vet, test and lint jobs
* Go 1.23 and 1.24 compatibility testing
* Go caching
* path-based workflow filtering
* a stable `ci-ok` aggregation gate
* protected `main` branch
* deliberate failure testing proving the PR gate works
* wall-clock measurements comparing the pipeline configurations

## Links

Pull request:

https://github.com/AbdullohML/DevOps-Intro/pull/5

Final successful CI run:

https://github.com/AbdullohML/DevOps-Intro/actions/runs/34890308982

Deliberate failed test run:

https://github.com/AbdullohML/DevOps-Intro/actions/runs/34886236027

Repository:

https://github.com/AbdullohML/DevOps-Intro

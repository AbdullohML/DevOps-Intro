# Lab 11 — Reproducible Builds with Nix

## Task 1 — Reproducible Go Build

### Flake

The project uses `flake.nix` at the repository root.

It pins nixpkgs through `flake.lock` and builds QuickNotes using `buildGoModule`.

Relevant configuration:

    {
      description = "QuickNotes reproducible build";

      inputs = {
        nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
      };

      outputs = { self, nixpkgs }:
        let
          system = "x86_64-linux";

          pkgs = import nixpkgs {
            inherit system;
          };

          quicknotes = pkgs.buildGoModule {
            pname = "quicknotes";
            version = "0.1.0";

            src = ./app;

            vendorHash = null;

            env.CGO_ENABLED = "0";

            ldflags = [
              "-s"
              "-w"
            ];
          };

          dockerImage = pkgs.dockerTools.buildImage {
            name = "quicknotes";
            tag = "nix";

            copyToRoot = pkgs.buildEnv {
              name = "quicknotes-root";
              paths = [
                pkgs.cacert
              ];
            };

            extraCommands = ''
              mkdir -p data
              chmod 0777 data
            '';

            config = {
              Entrypoint = [
                "${quicknotes}/bin/quicknotes"
              ];

              ExposedPorts = {
                "8080/tcp" = {};
              };

              User = "65532:65532";
            };
          };
        in
        {
          packages.${system} = {
            inherit quicknotes;
            default = quicknotes;
            docker = dockerImage;
          };

          devShells.${system}.default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.gopls
              pkgs.golangci-lint
            ];
          };
        };
    }

`flake.lock` is committed together with the flake.

The resolved nixpkgs revision is:

    b6018f87da91d19d0ab4cf979885689b469cdd41

### vendorHash note

QuickNotes currently has no external Go module dependencies:

    module quicknotes

    go 1.24

The initial build used a fake vendor hash. Nix reported:

    go: no dependencies to vendor
    vendor folder is empty, please set 'vendorHash = null;'

Therefore `vendorHash = null` is intentional for this project. There is no dependency vendor tree to hash.

### Build proof

Command:

    nix build .#quicknotes

Result:

    /nix/store/qhgs34649nfmggddl4d48ysn8qhw79xp-quicknotes-0.1.0

Binary inspection:

    result/bin/quicknotes: ELF 64-bit LSB executable, x86-64,
    statically linked, stripped

The binary size was approximately 5.7 MB.

### Runtime proof

QuickNotes was started directly from the Nix result:

    ./result/bin/quicknotes

Health request:

    curl -s http://localhost:8080/health

Response:

    {"notes":0,"status":"ok"}

### Reproducibility proof

Environment A:

    nix-store --query --hash "$(readlink -f result)"

Output:

    sha256:0a74h640yszx099ikrydxvilqckq4mh0g1wml44bis2z7lmyp5c4

Environment B was a fresh `nixos/nix` Docker container using a clean exported copy of the committed repository.

Output:

    Store path: /nix/store/qhgs34649nfmggddl4d48ysn8qhw79xp-quicknotes-0.1.0
    Store hash:
    sha256:0a74h640yszx099ikrydxvilqckq4mh0g1wml44bis2z7lmyp5c4

The two independent environments produced identical hashes.

### Development shell

The flake also exposes a development shell.

Command:

    nix develop --command bash -c '
      go version
      gopls version
      golangci-lint version
    '

Output:

    go version go1.25.10 linux/amd64
    golang.org/x/tools/gopls v0.20.0
    golangci-lint has version 2.6.2 built with go1.25.10

### Design question a — Why can plain go build differ between machines?

A plain `go build` does not by itself pin the complete build environment.

Different machines can use different Go compiler versions, resolved module versions, paths, native tools, or build metadata. Build IDs and environment-dependent metadata may also affect the produced binary.

Options such as `-trimpath`, a pinned Go toolchain, a fixed module graph, and a reproducible build environment reduce these differences.

Nix makes these inputs part of the build definition instead of relying on whatever happens to be installed on the host.

### Design question b — What does vendorHash cover?

For `buildGoModule`, `vendorHash` pins the fixed-output dependency set produced from the Go module dependencies.

If any dependency in that set changes, the hash no longer matches and Nix rejects the build.

For QuickNotes there are no external modules. Nix explicitly rejected a fake hash because the generated vendor directory was empty and instructed that `vendorHash = null` should be used.

If a project had external dependencies, leaving the dependency closure unpinned would remove an important reproducibility guarantee and normally require another valid dependency-fetching strategy.

### Design question c — Why is flake.lock important?

`flake.nix` can refer to a moving nixpkgs channel such as `nixos-25.11`.

`flake.lock` resolves that input to an exact revision and content hash. This fixes the versions of Go, build helpers, libraries, dockerTools, and the rest of nixpkgs used by the build.

Deleting the lock file allows Nix to resolve the channel again. A later nixpkgs revision could select different tools or dependencies and therefore produce a different derivation or output.

### Design question d — buildGoModule vs buildGoApplication

`buildGoModule` is the standard nixpkgs helper for Go projects using Go modules. It integrates naturally with the Go module graph and the Nix dependency hash mechanism.

`buildGoApplication` is useful in workflows that use a separately generated dependency lock or dependency description.

For QuickNotes I chose `buildGoModule` because the project is very small, already uses `go.mod`, and currently has no external Go dependencies. It is the simplest helper that satisfies the requirements.

---

## Task 2 — Deterministic Container Image

### Nix image

The flake exposes:

    packages.x86_64-linux.docker

It is built with:

    nix build .#docker

The image is created entirely by `dockerTools.buildImage`; Docker is not used during the image build.

Configuration:

- QuickNotes Nix binary is the entrypoint
- port `8080/tcp` is exposed
- runtime user is `65532:65532`
- `/data` is created as a writable runtime directory
- image can be loaded with Docker after Nix produces it

Docker inspection:

    User=65532:65532
    Entrypoint=["/nix/store/qhgs34649nfmggddl4d48ysn8qhw79xp-quicknotes-0.1.0/bin/quicknotes"]
    Ports={"8080/tcp":{}}

Runtime test:

    curl -s http://localhost:18080/health

Response:

    {"notes":0,"status":"ok"}

### Independent image reproducibility

Environment A:

    sha256sum result

Output:

    4ea15df97735ca18205e9ea888e4f6cefeec482fcca269a2965047bb45b74954  result

Environment B was a fresh `nixos/nix` container and a clean exported copy of the committed branch.

Output:

    Image digest:
    4ea15df97735ca18205e9ea888e4f6cefeec482fcca269a2965047bb45b74954

The image tarball digest is identical across both independent builds.

### Image size

The Nix output tarball:

    3.3 MB

Docker-reported loaded Nix image size:

    10210253 bytes

Lab 6 Docker image size:

    5838720 bytes

In this project the Lab 6 distroless image is smaller according to Docker's loaded-image size metric.

The important result for this lab is reproducibility rather than size: the Nix image was bit-for-bit identical between independent environments.

### Lab 6 Docker comparison

Two fresh builds were executed:

    docker build --no-cache -t qn-lab6:run1 ./app
    docker build --no-cache -t qn-lab6:run2 ./app

The resulting image IDs were:

    run1:
    sha256:43c38e7197d43aea68433d0c5b41ee8c964f136e61dcc985776ac47c0e86170c

    run2:
    sha256:b1b6d55002f5669f28674a46f5f7feb43a171d7592828e3dfd03c446499f8b3f

Both had the same reported size:

    5838720 bytes

but their image IDs differed.

This demonstrates that the normal Docker build was not bit-for-bit reproducible in this experiment.

### Design question e — Why can docker build be non-deterministic?

A conventional Docker build can include information that changes between builds even when the Dockerfile and Git SHA are unchanged.

Examples include layer creation timestamps, image configuration metadata, BuildKit provenance or attestations, package-manager state, unpinned external inputs, and metadata inherited from the build environment.

In this experiment the application source and image size were unchanged, but the two fresh Docker builds produced different image IDs.

Nix instead builds from a content-addressed and locked dependency graph and normalizes build inputs and metadata.

### Design question f — Reproducibility vs signatures

A signature proves that a particular signer approved or produced a particular artifact.

It does not prove that the artifact corresponds to the reviewed source code.

With a reproducible build, a security auditor can independently rebuild the source using the declared build recipe and compare the resulting digest with the published artifact.

If the digests match, the auditor has evidence that the distributed artifact corresponds to the reproducible source and build inputs.

Signatures and reproducibility therefore solve different problems and are strongest when used together.

### Design question g — Trade-offs of Nix

Nix provides strong dependency pinning, reproducible environments, isolated builds, and content-addressed caching.

The costs are additional complexity:

- the Nix language and model have a learning curve
- errors can be difficult to understand
- first builds may download a large dependency closure
- teams need Nix-specific tooling and cache knowledge
- some ecosystems integrate less naturally than others

Docker remains the default for many teams because Dockerfiles are widely understood, supported by most CI/CD systems and cloud platforms, and usually easier to introduce into an existing project.

---

## Bonus — CI-Verified Reproducibility

Workflow:

    .github/workflows/nix-repro.yml

The workflow runs on pushes and pull requests.

It contains:

1. Build A on a fresh GitHub-hosted runner
2. Build B on another fresh GitHub-hosted runner
3. Compare digests, which depends on both builds and fails if their outputs differ

The Nix installer and checkout actions are pinned by full commit SHA.

### Deliberately broken run

Red CI run:

https://github.com/AbdullohML/DevOps-Intro/actions/runs/38777441223

For this run, Build A deliberately modified a copy of the produced image after the Nix build.

Both independent build jobs completed, and the comparison gate detected that their SHA-256 digests differed.

This demonstrated that the CI gate actually catches divergence instead of only succeeding in the normal case.

### Final green run

Green CI run:

https://github.com/AbdullohML/DevOps-Intro/actions/runs/38777726858

Comparison log:

    Build A: 4ea15df97735ca18205e9ea888e4f6cefeec482fcca269a2965047bb45b74954
    Build B: 4ea15df97735ca18205e9ea888e4f6cefeec482fcca269a2965047bb45b74954
    SUCCESS: image digests are identical

This proves that two fresh CI runners independently produced the same image bytes.

### Design question h — Laptop reproducibility vs CI reproducibility

Two builds on one laptop can accidentally share state: the same filesystem, Nix store, environment variables, caches, credentials, or local configuration.

CI gives stronger evidence because each build job starts on a fresh hosted runner and the result is recorded in an auditable workflow log.

It also turns reproducibility into an enforced project property. A future change that causes the two builds to diverge makes CI fail automatically.

For a security auditor, that is much stronger than a developer saying that two local builds matched once.

### Design question i — Why two parallel jobs?

Running the build twice in a single job does not provide strong independence.

The second invocation can reuse the same Nix store, derivations, caches, environment, filesystem state, and host configuration. It may even be a pure cache hit instead of performing an independent build.

Two parallel jobs run on separate fresh runners with separate stores and environments.

Therefore matching results provide stronger evidence that the output is determined by the declared inputs rather than hidden state left by the first build.

### Design question j — SOURCE_DATE_EPOCH and dockerTools

Build timestamps can leak through generated files, archive metadata, compiler or linker metadata, and container image metadata such as creation timestamps.

`SOURCE_DATE_EPOCH` is the conventional way for build systems to replace wall-clock time with a deterministic timestamp.

Nix already removes many timestamp sources by building in isolated derivations and normalizing store outputs. In this lab the generated Nix artifacts visibly use epoch-style timestamps such as January 1, 1970.

`dockerTools.buildImage` constructs the image from Nix store paths with deterministic metadata rather than using the current wall-clock time for normal Docker layer creation.

A flake could still reintroduce nondeterminism if a build script explicitly generated content from the current time or another undeclared changing input.

---

## Final Result

Task 1:

- `nix build .#quicknotes` succeeds
- QuickNotes binary is static and stripped
- `/health` works
- two independent store hashes are identical
- `flake.lock` is committed
- dev shell contains Go, gopls, and golangci-lint
- design questions a-d answered

Task 2:

- `nix build .#docker` succeeds
- image loads in Docker
- image runs QuickNotes as non-root
- `8080/tcp` is exposed
- two independent image SHA-256 digests are identical
- Lab 6 Docker builds produced different image IDs
- image-size comparison documented
- design questions e-g answered

Bonus:

- two independent parallel CI build jobs
- third job compares digests
- deliberately broken red run recorded
- final matching green run recorded
- design questions h-j answered

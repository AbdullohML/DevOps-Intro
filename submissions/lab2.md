
important work progress

# Lab 2 — Version Control Deep Dive

## Task 1 — Git Object Model + Reflog Recovery

### 1.1 Plumbing chain: HEAD → tree → blob → file

git rev-parse HEAD:
    524cdc6193d9da3c2f644cb56e9f9175a3efff86

git cat-file -t HEAD:
    commit

git cat-file -p HEAD:
    tree 4e8941f4762188e39dde75dbbc42c9b8b0a2f920
    parent cfbe366dc10f167748c8806fbfaaa4e82a8172cf
    author Abdullojon <ab.muminov@innopolis.university> 1789076501 +0300
    committer Abdullojon <ab.muminov@innopolis.university> 1789076501 +0300
    gpgsig -----BEGIN SSH SIGNATURE-----
     U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgOdYf6LdkXn6234nYw9cgUh2O0f
     VX2HYhsm+O+duAa+4AAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
     AAAAQJ5W1HxmO4qc5y8iHWVC3F1jvgSKDH+A8KX0ouRWUCpkhJv+Ehn2YXfoosEjpMwyRG
     8nEcuPKDM+8+4skiQ4vAs=
     -----END SSH SIGNATURE-----

    fix: move PR template to repo root

    Signed-off-by: Abdullojon <ab.muminov@innopolis.university>

git cat-file -p 4e8941f4762188e39dde75dbbc42c9b8b0a2f920:
    040000 tree 1d07791eee3c3dd0955a02402b05b3a357816d8d	.github
    100644 blob 1c0a1e94b7bbdd951f456cda51af6b8484cc3cee	.gitignore
    100644 blob d10c04c6e7e0014f4fe883599c11747c15012d4e	README.md
    040000 tree 7d0898a908e274ea809722844cdbd836f3b1c05a	app
    040000 tree f4f047dd07b128eda5f899dfdaaf193f0291eaa2	labs
    040000 tree c0ac2d55cf4335df659b347df3d19d0594a06b6c	lectures

git cat-file -p 1c0a1e94b7bbdd951f456cda51af6b8484cc3cee  (.gitignore blob):
    # ⚠️  KEEP THIS FILE MINIMAL.
    #
    # This .gitignore is inherited by every student fork. Anything listed here
    # is something a student CANNOT `git add` without `-f`. So this file must
    # ONLY contain:
    #   (a) instructor-only paths (refs/), and
    #   (b) machine-generated junk that NOBODY should ever commit.
    #
    # Do NOT add lab DELIVERABLES here (scan reports, SBOMs, go.sum, k8s
    # manifests, CI workflows, Dockerfiles, playbooks, dashboards, …). Students
    # are told to commit those in their submission PRs — ignoring them upstream
    # silently breaks the lab. When in doubt, leave it OUT of this file.

    # ── Instructor-only ─────────────────────────────────────────────
    # Reference submissions (dry-run worked examples). Never pushed upstream;
    # students never see these. This is the one path that is intentionally hidden.
    refs/

    # ── Machine-generated junk (no one commits these) ───────────────
    # Compiled binaries / local runtime state
    app/quicknotes
    app/data/
    /quicknotes
    *.exe

    # Vagrant runtime state (Lab 5) — the Vagrantfile IS committed; .vagrant/ is not
    .vagrant/

    # Nix build symlinks (Lab 11) — flake.nix + flake.lock ARE committed; result is not
    result
    result-*

    # Terraform state — MUST never be committed (can contain secrets)
    *.tfstate
    *.tfstate.backup
    .terraform/

    # Python virtualenvs / caches
    .venv/
    __pycache__/
    *.pyc

    # Editor / IDE
    .vscode/
    .idea/
    *.swp

    # OS noise
    .DS_Store
    Thumbs.db

    # Local agent config (not part of the course)
    .claude/

    # NOTE: deliberately NOT ignored, because students commit them as lab evidence:
    #   submissions/labN.md        (lab reports)
    #   .github/workflows/*.yml    (Lab 3 CI)
    #   Dockerfile, compose.yaml   (Lab 6)
    #   ansible/                   (Lab 7)
    #   monitoring/                (Lab 8)
    #   *.sbom.cdx.json, zap-*.html/json, trivy-*.txt   (Lab 9 scan evidence)
    #   flake.nix, flake.lock      (Lab 11)
    #   wasm/main.go, spin.toml, go.sum   (Lab 12)

### 1.2 Inside .git/

ls -la .git/:
    total 56
    drwxrwxr-x  7 abdullloh abdullloh 4096 Sep 11 01:07 .
    drwxrwxr-x  7 abdullloh abdullloh 4096 Sep 11 01:07 ..
    -rw-rw-r--  1 abdullloh abdullloh   99 Sep 11 01:02 COMMIT_EDITMSG
    -rw-rw-r--  1 abdullloh abdullloh  794 Sep 11 00:24 FETCH_HEAD
    -rw-rw-r--  1 abdullloh abdullloh   21 Sep 11 01:07 HEAD
    -rw-rw-r--  1 abdullloh abdullloh  464 Sep 11 01:06 config
    -rw-rw-r--  1 abdullloh abdullloh   73 Sep 11 00:24 description
    drwxrwxr-x  2 abdullloh abdullloh 4096 Sep 11 00:24 hooks
    -rw-rw-r--  1 abdullloh abdullloh 3183 Sep 11 01:07 index
    drwxrwxr-x  2 abdullloh abdullloh 4096 Sep 11 00:24 info
    drwxrwxr-x  3 abdullloh abdullloh 4096 Sep 11 00:24 logs
    drwxrwxr-x 46 abdullloh abdullloh 4096 Sep 11 01:02 objects
    -rw-rw-r--  1 abdullloh abdullloh  112 Sep 11 00:24 packed-refs
    drwxrwxr-x  5 abdullloh abdullloh 4096 Sep 11 00:24 refs

cat .git/HEAD:
    ref: refs/heads/main

ls .git/refs/heads/:
    feature
    main

ls .git/objects/ | head:
    0a 0b 0c 0d 0e 0f 13 19 1a 1d

find .git/objects -type f | wc -l:
    48

Interpretation:
HEAD is a symbolic ref pointing at refs/heads/main. Branches live in
.git/refs/heads/. Object data is stored under .git/objects/ in
subdirectories named by the first two hex characters of each object's
SHA-1. 48 loose object files means most objects have not yet been packed
into a packfile — normal for a young clone that has never had `git gc`
run on it.

### 1.3 Disaster + reflog recovery

Chain of HEAD movements (git reflog | head -8):

    9f41b7d HEAD@{0}: reset: moving to HEAD~2
    c91a8fe HEAD@{1}: commit: wip(lab2): more progress
    e3f9cf5 HEAD@{2}: commit: wip(lab2): start
    9f41b7d HEAD@{3}: checkout: moving from feature/lab2 to feature/lab2
    9f41b7d HEAD@{4}: reset: moving to 9f41b7d
    9f41b7d HEAD@{5}: reset: moving to HEAD~2
    524cdc6 HEAD@{6}: checkout: moving from main to feature/lab2
    524cdc6 HEAD@{7}: checkout: moving from feature/lab1 to main

Recovery:

    $ git reset --hard c91a8fe
    HEAD is now at c91a8fe wip(lab2): more progress

git status after recovery:
    On branch feature/lab2
    nothing to commit, working tree clean

git log --oneline -3 after recovery:
    c91a8fe (HEAD -> feature/lab2) wip(lab2): more progress
    e3f9cf5 wip(lab2): start
    524cdc6 fix: move PR template to repo root

Explanation — what if `git gc` had run between the bad reset and recovery?
Reflog entries are kept for 30 days by default, so an ordinary `git gc`
during that window leaves the "lost" commits reachable via the reflog and
they can still be recovered. But `git gc` can be configured (or run by CI)
to prune unreachable objects immediately, and `git reflog expire
--expire=now` removes the only references pointing at the dangling
commits. Once that happens the wip commits are gone from .git/objects and
no command can bring them back — which is why capturing the SHA from
reflog before doing anything else is the safe move.

## Task 2 — Signed Tag & Rebase

### 2.1 Signed annotated tag

    $ git tag -a -s "v0.1.0-lab2-${USER}" -m "Lab 2 milestone — version control deep dive"
    $ git push origin "v0.1.0-lab2-${USER}"
     * [new tag]  v0.1.0-lab2-abdullloh -> v0.1.0-lab2-abdullloh

    $ git tag -l --format='%(refname:short) %(objecttype) %(*objecttype)'
    v0.0.1 tag commit
    v0.1.0-lab2-abdullloh tag commit

    $ git tag -v "v0.1.0-lab2-${USER}"
    object 524cdc6193d9da3c2f644cb56e9f9175a3efff86
    type commit
    tag v0.1.0-lab2-abdullloh
    tagger Abdullojon <ab.muminov@innopolis.university> 1789079135 +0300

    Lab 2 milestone — version control deep dive
    Good "git" signature for abdullloh with ED25519 key SHA256:+B8NZ7G5HtIxqBHp7P9ChWASKTlf/AG2kgY7JAZpVdM

The tag is an annotated tag object (`tag` type) pointing at a commit
(`commit` type), not a lightweight ref. `git tag -v` verifies the SSH
signature against my registered signing key.

### 2.2 Rebase + force-with-lease

Before rebase — `git log --oneline --graph ae65970 -5`:

    * ae65970 docs(lab2): task 1 object model and reflog recovery
    * c91a8fe wip(lab2): more progress
    * e3f9cf5 wip(lab2): start
    * 9f41b7d (upstream/main, upstream/HEAD) docs(lab7): make seed.json shipping explicit; require bonus artifacts, not logs
    * 8de962e docs(lab11): fix nixpkgs pin vs go.mod collision; add network fallback pitfalls

After rebase — `git log --oneline -5`:

    302db30 (HEAD -> feature/lab2, origin/feature/lab2) docs(lab2): task 1 object model and reflog recovery
    8635ced wip(lab2): more progress
    1a0042e wip(lab2): start
    51c32f8 (origin/main, origin/HEAD, main) docs: upstream moved while you worked
    524cdc6 (tag: v0.1.0-lab2-abdullloh) fix: move PR template to repo root

Force-push used --force-with-lease, not --force:

    $ git push --force-with-lease origin feature/lab2
     + ae65970...302db30 feature/lab2 -> feature/lab2 (forced update)

### Merge vs rebase — when to choose which

I'd rebase a private feature branch onto main to keep a linear, readable
history and avoid pointless merge commits for a single-author workstream —
that's what I did here. I'd choose merge when the branch is shared (someone
else may have already pulled it) or when the branch contains meaningful
sub-features whose individual commits I want to preserve as-is. I'd never
rebase a branch that's already been pushed and consumed by others, because
the rewritten SHAs break their clones. Rule of thumb: rebase local/private,
merge shared/public, and always use --force-with-lease when a rebase forces
a rewrite.

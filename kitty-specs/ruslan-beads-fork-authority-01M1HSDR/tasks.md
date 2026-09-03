# Work Packages: Ruslan Beads Fork Authority

**Mission**: `ruslan-beads-fork-authority-01M1HSDR`
**Planning branch**: `codex/beads-fork-authority`
**Final fork target**: `rusliksu/beads:main` through a reviewed pull request
**Excluded gates**: merge, release, package publication, installer changes, and active `bd` replacement

## Subtask Index

| ID | Description | Work Package | Parallel |
|----|-------------|--------------|----------|
| T001 | Audit the generated Spec Kitty bootstrap against repository policy | WP01 | No |
| T002 | Retain canonical project state and restore credential-safe ignores | WP01 | No |
| T003 | Author the canonical fork-maintenance operator guide | WP01 | Yes |
| T004 | Validate bootstrap health and a narrow reviewable diff | WP01 | No |
| T005 | Add failing acceptance tests for sync-manifest behavior | WP02 | No |
| T006 | Implement the read-only cross-platform sync preflight | WP02 | No |
| T007 | Sanitize provenance evidence and classify blockers | WP02 | No |
| T008 | Prove deterministic output in disposable repositories | WP02 | No |
| T009 | Add failing contract tests for the fork canary workflow | WP03 | No |
| T010 | Implement the fork-only Windows/Linux build matrix | WP03 | No |
| T011 | Bind canary artifacts and outcomes to the exact candidate SHA | WP03 | No |
| T012 | Validate workflow safety, timeouts, and non-install behavior | WP03 | No |
| T013 | Add failing acceptance tests for fork release identity | WP04 | No |
| T014 | Implement the validation-only identity helper | WP04 | No |
| T015 | Document the candidate identity and future release gate | WP04 | Yes |
| T016 | Run focused and repository-level quality gates | WP04 | No |

## WP01 — Persist Fork Governance Bootstrap

**Prompt**: [tasks/WP01-persist-fork-governance-bootstrap.md](tasks/WP01-persist-fork-governance-bootstrap.md)
**Priority**: P1
**Dependencies**: none
**Goal**: Keep only the project-local Spec Kitty state needed for reproducible missions and establish one canonical fork-maintenance entry point without changing Beads behavior.
**Independent test**: A clean clone can identify fork/upstream roles and pass Spec Kitty health checks while machine-local runtime state and credential material remain ignored.
**Estimated prompt size**: approximately 190 lines

- [ ] T001 Audit the generated Spec Kitty bootstrap against repository policy (WP01)
- [ ] T002 Retain canonical project state and restore credential-safe ignores (WP01)
- [ ] T003 Author the canonical fork-maintenance operator guide (WP01)
- [ ] T004 Validate bootstrap health and a narrow reviewable diff (WP01)

**Implementation sketch**: Classify every bootstrap file, retain only stable project identity/command metadata, restore Beads credential and proxied database ignores, then write `engdocs/FORK_MAINTENANCE.md` around the approved authority and gate model.

**Parallel opportunities**: T003 may be drafted while T001 classifies generated files, but both must reconcile before T004.
**Risks**: generated manifests may encode machine-local state; `.gitignore` edits may accidentally expose credentials; the guide may duplicate upstream product documentation.

## WP02 — Produce Sync Provenance Manifest

**Prompt**: [tasks/WP02-produce-sync-provenance-manifest.md](tasks/WP02-produce-sync-provenance-manifest.md)
**Priority**: P1
**Dependencies**: WP01
**Goal**: Add a read-only `pwsh` preflight that compares pinned refs and emits deterministic, credential-safe `fork-sync-manifest/v1` evidence.
**Independent test**: Disposable repositories cover identical, upstream-ahead, fork-diverged, conflict, missing-ref, and credential-bearing-remote scenarios without changing the checked-out branch or files.
**Estimated prompt size**: approximately 230 lines

- [ ] T005 Add failing acceptance tests for sync-manifest behavior (WP02)
- [ ] T006 Implement the read-only cross-platform sync preflight (WP02)
- [ ] T007 Sanitize provenance evidence and classify blockers (WP02)
- [ ] T008 Prove deterministic output in disposable repositories (WP02)

**Implementation sketch**: Commit acceptance tests first, prove meaningful failure, implement exact-SHA resolution/count/conflict forecasting, then harden evidence sanitization and rerun focused gates.

**Parallel opportunities**: none inside the WP; test-first sequencing is intentional.
**Risks**: Git-version differences, secret-bearing URLs, stale refs, and treating mergeability as proof of behavior equivalence.

## WP03 — Add Fork Candidate Build Canary

**Prompt**: [tasks/WP03-add-fork-candidate-build-canary.md](tasks/WP03-add-fork-candidate-build-canary.md)
**Priority**: P1
**Dependencies**: WP01, WP02
**Goal**: Add a fork-scoped pull-request canary that builds and smokes the same candidate commit on Windows and Linux and publishes checksum/manifests as ephemeral evidence.
**Independent test**: Contract tests prove both platforms, exact-SHA checkout, fork guard, required build tag, timeouts, smoke commands, evidence upload, and absence of release/install actions.
**Estimated prompt size**: approximately 220 lines

- [ ] T009 Add failing contract tests for the fork canary workflow (WP03)
- [ ] T010 Implement the fork-only Windows/Linux build matrix (WP03)
- [ ] T011 Bind canary artifacts and outcomes to the exact candidate SHA (WP03)
- [ ] T012 Validate workflow safety, timeouts, and non-install behavior (WP03)

**Implementation sketch**: Lock the workflow contract with tests, create a two-platform matrix guarded to `rusliksu/beads`, build/smoke without installation, and upload commit-bound evidence with explicit failure semantics.

**Parallel opportunities**: WP03 can run in parallel with WP04 after WP02 is accepted because their owned files do not overlap.
**Risks**: skipped/canceled jobs misreported as pass, unsigned binary warnings, action drift, and accidental release workflow invocation.

## WP04 — Define Fork Release Identity

**Prompt**: [tasks/WP04-define-fork-release-identity.md](tasks/WP04-define-fork-release-identity.md)
**Priority**: P2
**Dependencies**: WP01, WP02
**Goal**: Validate a visibly fork-specific prerelease identity with exact fork/upstream provenance while leaving tags, release channels, installers, and active binaries untouched.
**Independent test**: Accepted examples emit `fork-release-identity/v1`; upstream-identical, malformed, missing-provenance, or mutable-ref-only inputs fail without repository or environment mutation.
**Estimated prompt size**: approximately 210 lines

- [ ] T013 Add failing acceptance tests for fork release identity (WP04)
- [ ] T014 Implement the validation-only identity helper (WP04)
- [ ] T015 Document the candidate identity and future release gate (WP04)
- [ ] T016 Run focused and repository-level quality gates (WP04)

**Implementation sketch**: Define tests from the schema, implement pure validation/output, document the contract and prohibited side effects, then run targeted tests and repository gates without publishing or installing.

**Parallel opportunities**: WP04 can run in parallel with WP03 after WP02 is accepted.
**Risks**: identity collision with upstream, version drift during review, and an innocent validation path triggering existing release tooling.

## Delivery Sequence

```text
WP01 -> WP02 -> { WP03 || WP04 } -> mission review
```

MVP is WP01 + WP02: it establishes authority and provides a reproducible, read-only upstream comparison. WP03 is required before any candidate may be called cross-platform proven. WP04 completes the approved identity contract but does not authorize publication.

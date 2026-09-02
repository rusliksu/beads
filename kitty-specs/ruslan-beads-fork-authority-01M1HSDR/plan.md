# Implementation Plan: Ruslan Beads Fork Authority

**Branch**: `codex/beads-fork-authority` | **Date**: 2026-09-02 | **Spec**: [spec.md](spec.md)  
**Input**: Behavior-identical fork authority, reviewed upstream synchronization, Windows/Linux build canary, and fork release identity without publishing or installing anything.

## Summary

Persist the minimum repository-local Spec Kitty governance needed to own future work, then add a read-only PowerShell sync preflight that emits a machine-readable provenance manifest, a fork-only GitHub Actions canary that builds the exact candidate commit on Windows and Linux, and a release-identity validator that rejects upstream-indistinguishable candidate names. All additions sit outside Beads runtime and storage boundaries. Upstream adoption remains a normal task-branch pull request into `rusliksu/beads:main`; merge, release publication, and active `bd` replacement remain separate operator gates.

## Engineering Alignment

- **Invariant**: Fork `main` is stable authority and changes only through reviewed pull requests; upstream refs are evidence inputs, never an implicit write source.
- **Lifecycle**: discover pinned refs → emit sync manifest → open/review candidate PR → run two-platform canary on the exact PR commit → operator may merge later. Release identity validation may run on a candidate but never publishes it.
- **Failure posture**: conflict, missing ref, unknown behavior equivalence, unexecuted canary, or one-platform failure is a blocking result rather than a warning promoted to success.
- **Product boundary**: no command, database, schema, issue model, Dolt driver, or orchestration behavior changes.
- **Delivery boundary**: no direct push to fork `main`, auto-merge, tag, GitHub release, package publication, installer change, alias change, or executable replacement.

## Technical Context

**Language/Version**: Go 1.26.5 for repository contract tests; PowerShell 7 for cross-platform operator helpers; GitHub Actions YAML for hosted canaries  
**Primary Dependencies**: Existing Git CLI, PowerShell, GitHub Actions, `actions/checkout`, `actions/setup-go`, and existing Go module dependencies only; no new runtime dependency  
**Storage**: Versioned JSON evidence manifests and Markdown guidance only; no Beads/Dolt schema or persistent product data change  
**Testing**: Test-first Go contract tests under `scripts/`, targeted helper dry-runs in disposable Git repositories, `spec-kitty` artifact validation, and hosted Windows/Linux canary runs tied to one commit  
**Target Platform**: Windows 11 / `windows-latest` and Linux / `ubuntu-latest`; scripts use `pwsh` on both  
**Project Type**: Repository-maintenance tooling and governance documentation around a single Go CLI repository  
**Performance Goals**: Local no-fetch preflight completes within 60 seconds for the Beads repository; each hosted platform canary completes or times out within 15 minutes  
**Constraints**: Behavior-identical fork, PR-only stable-branch changes, exact-SHA evidence, no secret content in logs, no install/release side effects, no new product schema or orchestration surface  
**Scale/Scope**: One fork, one configured upstream, one stable branch, one sync candidate at a time, two required build platforms, four focused implementation work packages

## Charter Check

The project-local Spec Kitty charter is absent. The canonical product boundary in `engdocs/PROJECT_CHARTER.md` and repository instructions were applied instead.

- **Core scope**: PASS — tooling governs the fork repository; it does not add issue-tracker features.
- **Orchestration boundary**: PASS — no agent routing, retries, scheduling, or workflow semantics enter Beads core.
- **Storage boundary**: PASS — no Dolt driver, storage retry, database, or schema change.
- **Schema boundary**: PASS — evidence uses standalone JSON files, not issue metadata or first-class schema.
- **Contributor protection**: PASS WITH GATE — implementation and PR handling must read `CONTRIBUTING.md` and `PR_MAINTAINER_GUIDELINES.md`, then run `scripts/pr-preflight.sh --search "fork upstream sync release identity" --repo gastownhall/beads` before code work.
- **PR-only delivery**: PASS — task branch and review are mandatory; operator retains merge authority.
- **Credential discipline**: PASS — helpers print repository URLs and SHAs only, never credential-bearing remote URLs or auth material.

Post-design re-check: the design remains outside runtime, storage, and schema boundaries. No charter exception or complexity waiver is required.

## Project Structure

### Documentation (this mission)

```text
kitty-specs/ruslan-beads-fork-authority-01M1HSDR/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── fork-release-identity.schema.json
│   └── fork-sync-manifest.schema.json
├── tasks.md
└── tasks/
```

### Repository Surfaces

```text
.github/
└── workflows/
    └── fork-canary.yml

.kittify/
├── config.yaml
├── metadata.yaml
├── agent_profiles_manifest.json
├── canonical-events.jsonl
└── command-skills-manifest.json

engdocs/
├── FORK_MAINTENANCE.md
└── FORK_RELEASE_IDENTITY.md

scripts/
├── fork-sync-preflight.ps1
├── fork-release-identity.ps1
├── fork_sync_preflight_test.go
├── fork_release_identity_test.go
└── fork_canary_workflow_test.go
```

Existing bootstrap surfaces `.gitattributes`, `.gitignore`, `AGENTS.md`, and `.claudeignore` are retained only where Spec Kitty project governance or credential-safe ignore behavior requires them.

**Structure Decision**: Keep fork authority as repository-edge tooling and documentation. A PowerShell helper serves the Windows-first operator and runs unchanged under `pwsh` on Linux; Go tests match the repository's existing script-contract test convention and avoid adding Pester or another package manager.

## Implementation Concern Map

### IC-01 — Project Governance Bootstrap

- **Purpose**: Preserve the minimum generated Spec Kitty project state and canonical operator entry point while restoring Beads credential/Dolt ignore protections.
- **Relevant requirements**: FR-001, FR-003, FR-008
- **Affected surfaces**: `.kittify/`, `.gitattributes`, `.gitignore`, `AGENTS.md`, `.claudeignore`, `engdocs/FORK_MAINTENANCE.md`
- **Sequencing/depends-on**: none
- **Risks**: Generated manifests can create a noisy diff; ignored machine-local state must not be committed; upstream Beads guidance must remain authoritative for product development.

### IC-02 — Sync Provenance Preflight

- **Purpose**: Produce deterministic, read-only evidence for a pinned fork/upstream comparison before any candidate branch or PR is created.
- **Relevant requirements**: FR-002, FR-003, FR-004, FR-006, FR-008
- **Affected surfaces**: `scripts/fork-sync-preflight.ps1`, `scripts/fork_sync_preflight_test.go`, mission contract schema
- **Sequencing/depends-on**: IC-01
- **Risks**: Remote URLs may contain credentials; force-pushed refs can invalidate prior evidence; `git merge-tree` behavior must be tested across the supported Git range.

### IC-03 — Cross-Platform Candidate Canary

- **Purpose**: Build and smoke-test the exact pull-request commit on Windows and Linux, publishing commit-bound evidence without installing the binary.
- **Relevant requirements**: FR-005, FR-006, FR-008
- **Affected surfaces**: `.github/workflows/fork-canary.yml`, `scripts/fork_canary_workflow_test.go`
- **Sequencing/depends-on**: IC-01 and the manifest contract from IC-02
- **Risks**: A skipped/canceled job can look green at workflow level; workflow must be fork-scoped and pin actions; generated binaries must remain ephemeral artifacts.

### IC-04 — Fork Release Identity

- **Purpose**: Define and validate a fork-specific prerelease identity with exact fork and upstream provenance, without creating a tag or release.
- **Relevant requirements**: FR-002, FR-006, FR-007, FR-008
- **Affected surfaces**: `engdocs/FORK_RELEASE_IDENTITY.md`, `scripts/fork-release-identity.ps1`, `scripts/fork_release_identity_test.go`, mission contract schema
- **Sequencing/depends-on**: IC-02
- **Risks**: An identity that resembles upstream can cause accidental installation; upstream version changes during review must not mutate a pinned candidate.

## Design Decisions

1. **Evidence before mutation**: The sync helper defaults to local refs and produces evidence only. Fetching, candidate-branch creation, pushing, PR creation, merging, releasing, and installing are separate explicit commands or gates.
2. **PowerShell as the single operator helper language**: It is native to the primary Windows environment and available as `pwsh` on hosted Linux runners. No duplicate Bash implementation is introduced.
3. **Fork-only workflow guard**: The canary includes an explicit `github.repository == 'rusliksu/beads'` guard so it remains inert if later carried back upstream.
4. **Exact-commit evidence**: Sync manifests and canary artifacts include the evaluated Git SHA; branch names alone are never sufficient evidence.
5. **Fork prerelease format**: Candidate identities follow `<upstream-version>-ruslan.<sequence>+upstream.<short-sha>` and are validation-only in this mission.
6. **No edit to upstream release workflow**: `.github/workflows/release.yml`, `.goreleaser.yml`, installers, package metadata, and active channels remain unchanged. A future release mission may add a fork publisher after separate approval.

## Test Strategy

- Each code-bearing WP begins with an acceptance-test-only commit, proves the intended failure, then adds the smallest implementation commit that makes it pass.
- Script tests use disposable temporary repositories, sanitized fake remotes, and exact JSON assertions; no production `.beads` database is touched.
- Workflow contract tests assert repository guard, pull-request SHA checkout, required Windows/Linux jobs, timeout, build tag, smoke commands, and artifact SHA/manifests.
- Identity tests cover accepted fork prerelease examples and reject upstream-identical, malformed, mutable-ref-only, or provenance-incomplete identities.
- Relevant gates: targeted Go tests, `make ci-pr-lint`, `make test` when source/script contract changes warrant it, Spec Kitty validation, and GitHub-hosted canary evidence after push. A billing-blocked or unexecuted job is reported as unexecuted, not passed.

## Delivery Gates

1. **Planning gate**: spec, plan, tasks, requirement coverage, and analysis are internally consistent.
2. **Implementation preflight**: repository contributor/maintainer guidance read and upstream PR search completed.
3. **Local validation gate**: targeted tests and repository-required lint/test commands pass without installing `bd`.
4. **Remote review gate**: task branch may be pushed and a draft PR opened only by the parent/operator workflow; fork `main` remains unchanged.
5. **Canary gate**: Windows and Linux jobs execute successfully on the exact PR commit.
6. **Merge gate**: explicit operator action after review; no auto-merge.
7. **Release/install gates**: excluded from this mission and require separate explicit authorization.

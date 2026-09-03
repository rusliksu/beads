---
work_package_id: WP03
title: Add Fork Candidate Build Canary
dependencies:
- WP01
- WP02
requirement_refs:
- FR-005
- FR-006
- FR-008
planning_base_branch: codex/beads-fork-authority
merge_target_branch: codex/beads-fork-authority
branch_strategy: Implement in the computed WP03 lane after WP02; WP03 may proceed parallel to WP04 because ownership does not overlap.
subtasks:
- T009
- T010
- T011
- T012
history:
- timestamp: '2026-09-02T19:39:47Z'
  event: planned
  agent: planner-priti
agent_profile: implementer-ivan
authoritative_surface: .github/workflows/
create_intent:
- .github/workflows/fork-canary.yml
- scripts/fork_canary_workflow_test.go
execution_mode: code_change
owned_files:
- .github/workflows/fork-canary.yml
- scripts/fork_canary_workflow_test.go
role: implementer
tags: []
tracker_refs: []
---

## ⚡ Do This First: Load Agent Profile

Load the assigned profile before any other action:

```text
/ad-hoc-profile-load implementer-ivan
```

Use only the WP03 lane and owned workflow/test files. Do not modify the upstream release workflow or any installer.

## Objective

Add a GitHub Actions workflow that runs only for `rusliksu/beads`, checks out the exact pull-request candidate commit, builds and smokes it on Windows and Linux, and uploads short-lived checksum/manifests bound to that SHA.

This is a canary, not a release pipeline. It must never tag, publish, install, sign with production credentials, change package channels, or merge a PR.

## Preflight and Branch Strategy

Read contributor/maintainer guidance and run:

```bash
scripts/pr-preflight.sh --search "windows linux build canary workflow" --repo gastownhall/beads
```

- Planning/base: `codex/beads-fork-authority`
- Mission target: `codex/beads-fork-authority`
- Later PR target: `rusliksu/beads:main`
- Workspace: WP03 lane from `lanes.json`
- Implement command:

```text
spec-kitty agent action implement WP03 --agent codex --mission ruslan-beads-fork-authority-01M1HSDR
```

## T009 — Workflow Contract Tests First

Create `scripts/fork_canary_workflow_test.go` and commit it before the workflow.

Tests must parse or structurally inspect `.github/workflows/fork-canary.yml` and fail meaningfully until the workflow exists. Assert:

- pull-request trigger targeting fork `main`, plus optional manual dispatch that cannot publish;
- explicit `github.repository == 'rusliksu/beads'` job/workflow guard;
- `windows-latest` and `ubuntu-latest` both required;
- checkout pinned action SHA and exact PR head/candidate SHA semantics;
- setup-go pinned action SHA and `go-version-file: go.mod`;
- 15-minute or tighter per-job timeout;
- build uses `gms_pure_go` and creates a task-local artifact path;
- `version` and `help` smoke checks execute the just-built binary by explicit path;
- manifest/checksum includes `github.sha` or resolved candidate SHA;
- artifact retention is short and missing files fail;
- no release, tag, package publish, installer, PATH replacement, or production signing step.

Demonstrate that changing one meaningful workflow value (for example removing the Windows leg or repository guard) makes the test fail for that contract.

## T010 — Two-Platform Build Matrix

Create `.github/workflows/fork-canary.yml` using the repository's existing pinned action versions and cache conventions when useful.

Each platform leg must:

1. check out the exact candidate commit;
2. set up Go from `go.mod`;
3. disable Beads metrics/event flush;
4. build `cmd/bd` with repository-required tags into a runner-temporary directory;
5. execute `version` and `help` from that exact path;
6. fail if build or smoke is skipped/fails;
7. avoid any installation path or shell alias.

Use platform-native checksum commands or a small portable approach already available on runners. Do not introduce a package install step.

## T011 — Commit-Bound Evidence

For each platform, write a manifest containing:

- candidate SHA;
- platform/runner;
- Go version;
- build tags;
- build/smoke outcome;
- binary checksum;
- workflow run identifier.

Upload platform-specific artifacts with short retention and `if-no-files-found: error`. Names must include platform and candidate identity sufficiently to avoid ambiguity.

Do not claim a combined pass inside one platform job. Cross-platform readiness is the conjunction of both required jobs and is assessed by review/branch protection.

## T012 — Safety and Validation

Run:

```text
go test ./scripts -run TestForkCanaryWorkflow -count=1
./scripts/check-build-tags.sh
git diff --check
```

If a local workflow linter already exists, use it; do not install a new linter without separate dependency approval. After a later authorized push, record actual job conclusions and confirm that jobs executed. A workflow definition test does not prove hosted execution.

## Definition of Done

- [ ] Test-only commit precedes workflow implementation.
- [ ] Contract mutation proves tests detect a missing platform/guard.
- [ ] Both required platforms build/smoke the exact same candidate SHA.
- [ ] Evidence is immutable-SHA-bound and ephemeral.
- [ ] Repository guard prevents upstream-side execution.
- [ ] No release, install, tag, package, merge, or signing credential use occurs.
- [ ] Hosted execution is correctly labeled pending until a later authorized push.

## Risks and Stop Conditions

Stop if the fork cannot run Actions, required workflows are billing-blocked, action pinning conflicts with repository policy, or one platform only compiles without executing smoke checks. Do not weaken the two-platform requirement to make the workflow green.

## Reviewer Guidance

Inspect `if` conditions and event SHAs carefully; pull-request merge refs can differ from head commits. Confirm the manifest SHA matches the built source. Treat skipped and canceled legs as unexecuted. Search the workflow for `release`, `publish`, `install`, `PATH`, tag pushes, and secrets.

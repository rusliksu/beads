---
work_package_id: WP02
title: Produce Sync Provenance Manifest
dependencies:
- WP01
requirement_refs:
- FR-002
- FR-003
- FR-004
- FR-006
- FR-008
planning_base_branch: codex/beads-fork-authority
merge_target_branch: codex/beads-fork-authority
branch_strategy: Implement in the lane worktree allocated by lanes.json after WP01 is accepted; later delivery is by reviewed fork PR.
subtasks:
- T005
- T006
- T007
- T008
history:
- timestamp: '2026-09-02T19:39:47Z'
  event: planned
  agent: planner-priti
agent_profile: implementer-ivan
authoritative_surface: scripts/fork-sync-preflight.ps1
create_intent:
- scripts/fork-sync-preflight.ps1
- scripts/fork_sync_preflight_test.go
execution_mode: code_change
owned_files:
- scripts/fork-sync-preflight.ps1
- scripts/fork_sync_preflight_test.go
role: implementer
tags: []
tracker_refs: []
---

## ⚡ Do This First: Load Agent Profile

Before reading implementation files or editing anything, load:

```text
/ad-hoc-profile-load implementer-ivan
```

Work only in the allocated WP02 lane and the two owned files. Do not touch remotes, installed binaries, releases, or the production Beads database.

## Objective

Create a PowerShell 7 helper that resolves two local Git refs, computes a pinned fork/upstream comparison, forecasts conflicts without changing the worktree, and writes `fork-sync-manifest/v1` JSON.

The helper is an evidence generator. It must not create/switch branches, merge/rebase/cherry-pick, push, open a PR, tag, publish, install, or infer behavior equivalence solely from Git mergeability.

## Required Context and Preflight

Read `engdocs/FORK_MAINTENANCE.md`, the mission contracts, and repository contribution/PR guidance. Run:

```bash
scripts/pr-preflight.sh --search "fork sync provenance manifest" --repo gastownhall/beads
```

Stop if overlapping contributor work exists until the parent decides whether to build on it.

## Branch Strategy

- Planning/base branch: `codex/beads-fork-authority`
- Mission target: `codex/beads-fork-authority`
- Later PR target: `rusliksu/beads:main`
- Workspace: WP02 lane from `lanes.json`
- Implement command:

```text
spec-kitty agent action implement WP02 --agent codex --mission ruslan-beads-fork-authority-01M1HSDR
```

## T005 — Acceptance Tests First

Create `scripts/fork_sync_preflight_test.go` and commit it separately before production script changes.

Use `t.TempDir()` repositories with repo-local hooks. Cover:

1. identical fork/upstream refs → `already_identical`;
2. upstream-only commits and explicit equivalence input → `ready_for_pr`;
3. fork-only commits → equivalence cannot default to unchanged;
4. both sides changed without conflict → conservative blocked/unknown result;
5. forecasted content conflict → blocked with path evidence;
6. missing/non-commit ref → deterministic non-zero failure;
7. credential-bearing remote URL → credentials never appear in JSON/stdout/stderr;
8. no-fetch mode → no remote-tracking ref or working-tree mutation;
9. output schema fields and 40-character lowercase SHAs;
10. repeat run with the same refs → semantic output identical except allowed generation time.

Prove the first run fails for the intended missing helper/contract reason. Parser or fixture setup failures are not acceptance evidence.

## T006 — Implement Read-Only Preflight

Create `scripts/fork-sync-preflight.ps1` with strict parameter validation and `Set-StrictMode`.

Required inputs:

- fork ref, default `refs/remotes/origin/main`;
- upstream ref, default `refs/remotes/upstream/main`;
- output path;
- explicit equivalence assertion or conservative default `unknown`;
- `-NoFetch` default/flag semantics that never fetch implicitly.

Required Git operations are read-only: verify repository, resolve commits with `^{commit}`, compute merge base and directional counts, and forecast conflicts with plumbing that does not update index/HEAD/worktree.

Never rely on a branch name as evidence after resolving it. The manifest uses the exact SHAs.

## T007 — Sanitize and Classify Evidence

Implement the contract from `contracts/fork-sync-manifest.schema.json`.

Rules:

- output repository identifiers (`rusliksu/beads`, `gastownhall/beads`), not credential-bearing URLs;
- never echo environment variables or raw authenticated remotes;
- sorted unique conflict paths and blockers;
- `ready_for_pr` only for a non-empty upstream range, zero conflicts, zero unreviewed fork behavior divergence, and explicit `behavior_equivalence=unchanged`;
- `already_identical` only when both directional counts are zero;
- all ambiguous states block;
- write JSON atomically to the requested output path;
- stdout may summarize status/SHAs but must not become the only evidence surface.

Do not add automatic fetch, candidate branch creation, or GitHub operations.

## T008 — Deterministic Verification

Run targeted tests on Windows. If `pwsh` is available on Linux in the validation environment, run the same suite there; otherwise hosted canary coverage is the later evidence gate.

Minimum commands:

```text
go test ./scripts -run TestForkSyncPreflight -count=1
git diff --check
```

Also run the helper in a disposable repo and compare `git status --porcelain=v1`, current HEAD, current branch, index tree, and remote-tracking refs before/after. They must be unchanged.

## Definition of Done

- [ ] Test-only commit exists before implementation commit.
- [ ] Meaningful acceptance mutation/failure was observed.
- [ ] All contract scenarios pass in disposable repositories.
- [ ] JSON contains exact immutable provenance and conservative blockers.
- [ ] No credential-bearing value is emitted.
- [ ] No fetch, branch, index, worktree, remote, Beads DB, release, or install mutation occurs.

## Risks and Stop Conditions

Stop if supported Git cannot perform a non-mutating conflict forecast reliably, if remote sanitization cannot be proven, or if PowerShell behavior differs materially between Windows and Linux. Report the exact failing command/output and do not substitute a mutating trial merge.

## Reviewer Guidance

Review negative paths first. Try to trick the helper with annotated tags, short SHAs, missing refs, URLs containing user info, unrelated histories, dirty worktrees, and both-sides divergence. The central review question is whether any uncertain state can be mislabeled ready.

---
work_package_id: WP04
title: Define Fork Release Identity
dependencies:
- WP01
- WP02
requirement_refs:
- FR-002
- FR-006
- FR-007
- FR-008
planning_base_branch: codex/beads-fork-authority
merge_target_branch: codex/beads-fork-authority
branch_strategy: Planning artifacts for this mission were generated on codex/beads-fork-authority. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into codex/beads-fork-authority unless the human explicitly redirects the landing branch.
subtasks:
- T013
- T014
- T015
- T016
history:
- timestamp: '2026-09-02T19:39:47Z'
  event: planned
  agent: planner-priti
agent_profile: implementer-ivan
authoritative_surface: scripts/fork-release-identity.ps1
create_intent:
- scripts/fork-release-identity.ps1
- scripts/fork_release_identity_test.go
- engdocs/FORK_RELEASE_IDENTITY.md
execution_mode: code_change
owned_files:
- scripts/fork-release-identity.ps1
- scripts/fork_release_identity_test.go
- engdocs/FORK_RELEASE_IDENTITY.md
role: implementer
tags: []
tracker_refs: []
---

## ⚡ Do This First: Load Agent Profile

Load before inspecting implementation surfaces:

```text
/ad-hoc-profile-load implementer-ivan
```

Stay within the WP04 lane and owned files. Do not call existing release, bump-version, GoReleaser, installer, package-manager, or tag commands.

## Objective

Define and validate a fork-specific candidate identity that is visibly distinct from upstream and traces to exact fork/upstream commits. Emit only a JSON identity record with `published=false` and `installed=false`.

The helper is pure validation/output. This WP does not create a release candidate in GitHub, build a distributable package, change version sources, or install anything.

## Preflight and Branch Strategy

Read `RELEASING.md` only to identify surfaces that must remain untouched, then run:

```bash
scripts/pr-preflight.sh --search "fork release identity prerelease version" --repo gastownhall/beads
```

- Planning/base: `codex/beads-fork-authority`
- Mission target: `codex/beads-fork-authority`
- Later PR target: `rusliksu/beads:main`
- Workspace: WP04 lane from `lanes.json`
- Implement command:

```text
spec-kitty agent action implement WP04 --agent codex --mission ruslan-beads-fork-authority-01M1HSDR
```

## T013 — Identity Acceptance Tests First

Create `scripts/fork_release_identity_test.go` as a separate test-only commit.

Cover:

- accepted `1.2.3-ruslan.1+upstream.abcdef0` shape;
- sequence greater than zero;
- exact 40-character lowercase fork/upstream SHAs;
- build-metadata short SHA prefixes the provided upstream SHA;
- candidate version base equals upstream version;
- upstream-identical `1.2.3` rejected;
- wrong owner marker, uppercase SHA, missing metadata, mutable ref, or malformed SemVer rejected;
- output always contains `published=false` and `installed=false`;
- existing Git refs, status, tags, remotes, PATH, aliases, and installed executable metadata remain unchanged;
- repeat runs produce semantically identical records except generation time if included.

Demonstrate a meaningful failure before implementing the helper. A missing PowerShell executable in the test environment is a blocker, not a passing skip for the core contract.

## T014 — Validation-Only Helper

Create `scripts/fork-release-identity.ps1` with strict inputs:

- `CandidateVersion`;
- `UpstreamVersion`;
- `ForkSha`;
- `UpstreamSha`;
- output path.

Validate the mission schema and cross-field invariants. Write JSON atomically. It may inspect local commits only when an explicit repository path/ref validation option is provided; by default the SHAs are values to validate, not refs to mutate.

Explicit prohibitions:

- no `git tag`, `git push`, branch change, commit, or remote call;
- no `goreleaser`, release script, version bump, package publish, installer, PATH, alias, or executable copy;
- no reading or logging tokens/secrets;
- no rewriting `.goreleaser.yml` or upstream release ownership.

## T015 — Canonical Identity Guide

Create `engdocs/FORK_RELEASE_IDENTITY.md` for Ruslan and future release operators.

Document:

- canonical identity shape and worked examples;
- how upstream version, fork sequence, fork SHA, and upstream SHA relate;
- why validation is separate from publication;
- required preceding sync-manifest and Windows/Linux canary evidence;
- exact prohibited side effects in this mission;
- future release gate inputs and decisions, without prescribing or running publication;
- collision/rollback rules if upstream publishes a newer version while a candidate is open.

Link back to `engdocs/FORK_MAINTENANCE.md` and the mission schema. Do not duplicate the entire upstream `RELEASING.md`.

## T016 — Quality Gates

Run:

```text
go test ./scripts -run TestForkReleaseIdentity -count=1
make ci-pr-lint
make test
git diff --check
```

If full `make test` is too slow or blocked by a documented environment prerequisite, preserve the targeted pass and report the exact full-suite limitation. Do not install `bd` to satisfy a test.

Review the aggregate mission diff for edits to prohibited release/install surfaces. There should be none.

## Definition of Done

- [ ] Test-only commit precedes implementation.
- [ ] Meaningful mutation proves identity tests constrain behavior.
- [ ] Valid fork candidates emit schema-conformant immutable provenance.
- [ ] Upstream-indistinguishable and incomplete identities fail.
- [ ] Documentation makes the future release gate explicit.
- [ ] No tag, release, package, installer, version-source, PATH, alias, remote, or active binary change occurred.

## Risks and Stop Conditions

Stop if the chosen identity is incompatible with repository SemVer parsing, if existing upstream contributor work overlaps, or if validation cannot avoid invoking release tooling. A future publisher design is out of scope and must become a separately approved mission.

## Reviewer Guidance

Test confusing and adversarial identities. Verify exact SHAs, base version equality, and fork marker. Search the diff and command transcript for any publication/install side effect. Ensure documentation never implies that successful identity validation is release approval.

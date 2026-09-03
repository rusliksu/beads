# Ruslan Beads Fork Maintenance

This is the operator entry point for maintaining Ruslan's Beads distribution.
It supplements, but does not replace, the Beads [project charter](PROJECT_CHARTER.md),
[contributor guidance](../CONTRIBUTING.md), or
[maintainer PR policy](../PR_MAINTAINER_GUIDELINES.md).

## Repository authority

- `rusliksu/beads` is the **fork authority**. Its `main` branch is the stable
  state of Ruslan's distribution.
- `gastownhall/beads` is the **upstream provenance** repository. Upstream refs
  are evidence inputs; they are not an implicit write source for the fork.
- Every fork change and upstream adoption uses a task branch and a reviewed
  pull request into `rusliksu/beads:main`. Direct pushes and auto-merge are not
  allowed.
- This fork starts behavior-identical to upstream. Intentional product
  divergence requires a separate approved Mission; it must not be hidden in a
  synchronization pull request.

## Configure and verify remotes

Use credential-free HTTPS URLs in recorded commands and evidence. Never copy a
remote URL containing embedded credentials into logs or manifests.

```powershell
git remote get-url origin
git remote get-url upstream
```

Expected values:

```text
origin   https://github.com/rusliksu/beads.git
upstream https://github.com/gastownhall/beads.git
```

For a fresh clone, add and verify upstream without changing the current branch:

```powershell
git remote add upstream https://github.com/gastownhall/beads.git
git remote get-url origin
git remote get-url upstream
git status --short --branch
```

If `upstream` already exists with the wrong credential-free URL, correct it
explicitly and verify again:

```powershell
git remote set-url upstream https://github.com/gastownhall/beads.git
git remote get-url upstream
```

A credential-bearing remote is a stop condition: sanitize the configuration
before collecting or sharing evidence.

## Sync candidate lifecycle

A **sync candidate** is an unmerged proposal pinned to immutable fork and
upstream commits. Prepare it in a task-owned worktree; never in the stable
primary checkout.

1. Read the [Mission specification](../kitty-specs/ruslan-beads-fork-authority-01M1HSDR/spec.md)
   and current context before any high-risk Git action.
2. Read `CONTRIBUTING.md` and `PR_MAINTAINER_GUIDELINES.md`, then run the
   repository's read-only contributor preflight. If it cannot execute, report
   the exact environmental failure and complete an equivalent read-only search;
   do not call the preflight green.
3. Fetch fork and upstream refs explicitly. Record the resolved `origin/main`
   and `upstream/main` commit IDs; branch names alone are not evidence.
4. Run the read-only preflight helper with pinned local refs:

   ```powershell
   pwsh -NoProfile -File scripts/fork-sync-preflight.ps1 `
     -ForkRef refs/remotes/origin/main `
     -UpstreamRef refs/remotes/upstream/main `
     -OutputPath "$env:TEMP/beads-fork-sync-manifest.json" `
     -NoFetch
   ```

5. Inspect the complete upstream range, fork-only commits, merge forecast,
   conflicts, behavior-equivalence verdict, and blockers in the generated
   `fork-sync-manifest/v1` record.
6. Only a candidate with `candidate_status=ready_for_pr`, no conflicts, and
   `behavior_equivalence=unchanged` may proceed to a task branch and pull
   request. The candidate branch and PR must preserve the pinned range.
7. Review the aggregate diff and contributor prior art before readiness. A
   force-pushed source ref or stale recorded SHA invalidates the evidence;
   regenerate it instead of silently expanding the candidate.
8. Require executed Windows and Linux canary jobs for the exact PR commit.
   Each result records the candidate SHA, build tag, smoke outcome, and artifact
   checksum. A skipped, canceled, billing-blocked, timed-out-before-test, or
   otherwise unexecuted job is `unexecuted`, not passed.
9. Merge remains an explicit operator action after review. A green check does
   not authorize auto-merge.

The evidence shapes and invariants are defined in the Mission
[data model](../kitty-specs/ruslan-beads-fork-authority-01M1HSDR/data-model.md),
[`fork-sync-manifest` contract](../kitty-specs/ruslan-beads-fork-authority-01M1HSDR/contracts/fork-sync-manifest.schema.json),
and [implementation plan](../kitty-specs/ruslan-beads-fork-authority-01M1HSDR/plan.md).

## Blocking rules

Do not mark a candidate ready, merge it, or hand it to release work when any
of these conditions exists:

- the fork or upstream commit cannot be resolved to an immutable SHA;
- the recorded ref changed after evidence was generated;
- the merge forecast reports a conflict;
- fork-only product changes make behavior equivalence unknown or divergent;
- contributor prior art has not been reviewed and attributed;
- the aggregate diff contains unrelated changes, `.beads/` data, credentials,
  host paths, or generated machine-local state;
- required CI failed, is pending, or did not execute;
- Windows and Linux results refer to different commits;
- the candidate identity is indistinguishable from upstream.

A blocked candidate leaves fork `main` unchanged. Resolve the cause in the
task branch or prepare a new pinned candidate; do not weaken the evidence
contract.

## Bootstrap state classification

The repository tracks only stable project bootstrap produced by the canonical
Spec Kitty CLI:

| Category | Tracked surfaces | Rule |
| --- | --- | --- |
| Project identity and configuration | `.kittify/config.yaml`, `.kittify/metadata.yaml` | Preserve the generated project UUID, slug, schema metadata, and version; do not normalize generated values by hand. |
| Canonical event and routing manifests | `.kittify/canonical-events.jsonl`, `.kittify/agent_profiles_manifest.json`, `.kittify/command-skills-manifest.json` | Track verbatim canonical output when doctor reports no drift. |
| Merge and orientation contract | `.gitattributes`, bounded `AGENTS.md` orientation block, `.claudeignore` | Preserve Spec Kitty merge drivers, generated-file classification, and command routing without rewriting Beads product instructions. |
| Host-local runtime, caches, and workspaces | `.kittify/runtime/`, `.kittify/workspaces/`, `.kittify/events/`, `.kittify/logs/`, `.kittify/derived/`, `.kittify/dossiers/`, `.kittify/migrations/`, `.kittify/skills-manifest.json`, `.kittify/sync-state.json`, `.worktrees/` | Ignore; never stage. |
| Credential and local database material | `.beads-credential-key`, `.beads/proxieddb/`, `.beads/` local Dolt state | Ignore; never inspect credential contents or stage local databases. |
| Unrelated upstream changes | Any surface outside the Mission/WP ownership map | Exclude from this change. |

Generated stable files must contain no absolute user paths, credential values,
tokens, or local database material. If a future canonical file contains
volatile host identity that should not be tracked, exclude and document the
whole generated file; do not hand-edit a canonical manifest to conceal drift.

## Closed delivery gates

Fork maintenance evidence does **not** authorize any of the following:

- direct push to or automatic merge into `main`;
- Git tag or GitHub release creation;
- package, installer, or distribution-channel publication;
- changes to upstream release workflows or installer ownership;
- modification, aliasing, shadowing, or replacement of the active `bd`.

After an accepted synchronization PR, hand release-candidate work to the
separate release-identity gate described by the Mission
[`fork-release-identity` contract](../kitty-specs/ruslan-beads-fork-authority-01M1HSDR/contracts/fork-release-identity.schema.json).
Publishing a release still requires a later explicitly authorized Mission.
Installing or activating that release requires another explicit environment
gate and must verify the selected artifact independently.

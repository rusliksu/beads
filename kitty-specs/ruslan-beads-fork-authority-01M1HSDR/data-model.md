# Data Model: Fork Maintenance Evidence

This mission adds no Beads product entities or database schema. The following are standalone evidence records used by repository tooling and CI.

## SyncManifest

| Field | Type | Rule |
|-------|------|------|
| `schema` | string | Exactly `fork-sync-manifest/v1` |
| `generated_at` | UTC timestamp | Evidence creation time; never used as identity |
| `fork_repository` | string | Exactly `rusliksu/beads` for this fork |
| `fork_base_sha` | 40-character Git SHA | Must resolve locally and remain immutable |
| `upstream_repository` | string | Exactly `gastownhall/beads` |
| `upstream_head_sha` | 40-character Git SHA | Must resolve locally and remain immutable |
| `merge_base_sha` | 40-character Git SHA | Common base for the evaluated pair |
| `fork_only_commits` | non-negative integer | Commits reachable only from fork base |
| `upstream_only_commits` | non-negative integer | Commits reachable only from upstream head |
| `conflicts` | array of paths | Empty only when the forecast finds no conflict |
| `behavior_equivalence` | enum | `unchanged`, `unknown`, or `divergent`; only `unchanged` can support behavior-identical readiness |
| `candidate_status` | enum | `ready_for_pr`, `blocked`, or `already_identical` |
| `blockers` | array of strings | Non-empty whenever status is `blocked` |

### Invariants

- The three SHA fields refer to immutable commits, not branch-name strings.
- `ready_for_pr` requires no conflicts, `behavior_equivalence=unchanged`, and a non-zero upstream range.
- Any fork-only product change makes equivalence `unknown` or `divergent` until reviewed; the helper cannot infer behavior identity from a clean merge alone.
- The manifest contains no credentials, tokens, environment values, local home paths, or remote URLs with user info.

## CanaryEvidence

| Field | Type | Rule |
|-------|------|------|
| `candidate_sha` | 40-character Git SHA | Same value for all required platform legs |
| `platform` | enum | `windows` or `linux` |
| `runner` | string | Hosted runner label used for the check |
| `build_tags` | string | Must include the repository-required `gms_pure_go` tag |
| `build_result` | enum | `passed`, `failed`, or `unexecuted` |
| `smoke_result` | enum | `passed`, `failed`, or `unexecuted` |
| `artifact_sha256` | checksum or null | Required only when a binary was built |

### Invariants

- Cross-platform readiness requires two `passed` records for the same `candidate_sha`, one per platform.
- Skipped, canceled, billing-blocked, timed-out-before-execution, or missing jobs are `unexecuted`, not `passed`.
- Artifacts are ephemeral canary outputs and are not installed or published as releases.

## ForkReleaseIdentity

| Field | Type | Rule |
|-------|------|------|
| `schema` | string | Exactly `fork-release-identity/v1` |
| `candidate_version` | string | `<upstream-version>-ruslan.<sequence>+upstream.<short-sha>` |
| `upstream_version` | semantic version | Stable upstream base version without fork marker |
| `sequence` | positive integer | Fork candidate sequence for that upstream version |
| `fork_sha` | 40-character Git SHA | Exact candidate commit |
| `upstream_sha` | 40-character Git SHA | Exact provenance commit |
| `published` | boolean | Must be `false` in this mission |
| `installed` | boolean | Must be `false` in this mission |

### Invariants

- `candidate_version` must be visibly fork-specific and cannot equal `upstream_version`.
- The build-metadata short SHA must prefix `upstream_sha`.
- Validation emits a record only; it does not create tags, releases, packages, aliases, or installations.

## Lifecycle

```text
local refs
  -> SyncManifest (already_identical | blocked | ready_for_pr)
  -> candidate PR commit
  -> Windows CanaryEvidence + Linux CanaryEvidence
  -> reviewed candidate (operator may merge later)
  -> ForkReleaseIdentity proposal (future release gate remains closed)
```

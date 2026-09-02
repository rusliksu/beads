# Research: Ruslan Beads Fork Authority

## Decision 1: Keep Fork Governance Outside Beads Runtime

- **Decision**: Implement the fork boundary through repository documentation, read-only helper scripts, validation tests, and fork-scoped CI.
- **Rationale**: `engdocs/PROJECT_CHARTER.md` limits Beads core to issue-tracking primitives and excludes orchestration policy. Fork maintenance does not require a new command, field, schema, or storage capability.
- **Alternatives considered**: Add `bd fork-sync` or first-class provenance fields. Rejected because they create upstream product divergence for an operator-specific workflow.

## Decision 2: Emit a Versioned Sync Manifest

- **Decision**: Make the preflight output conform to `fork-sync-manifest/v1` and include exact fork, upstream, and merge-base SHAs plus divergence/conflict state.
- **Rationale**: A versioned machine-readable artifact is reviewable, testable, and reusable by CI without treating console prose as a contract.
- **Alternatives considered**: Document manual Git commands only. Rejected because it cannot reliably prove which commits were compared or prevent stale-ref interpretation.

## Decision 3: Use PowerShell 7 Across Windows and Linux

- **Decision**: Author one `pwsh` helper surface and exercise it on both platforms.
- **Rationale**: Windows is the primary maintenance environment, `pwsh` is available on GitHub-hosted Windows and Ubuntu runners, and a single implementation avoids cross-shell drift.
- **Alternatives considered**: Bash-only helper, duplicate Bash/PowerShell helpers, or a new Go maintenance binary. Rejected respectively for Windows friction, duplicated contracts, and excessive product-like surface.

## Decision 4: Add a Dedicated Fork Canary

- **Decision**: Add a workflow guarded to `rusliksu/beads` that builds and smokes the exact PR commit on `windows-latest` and `ubuntu-latest`.
- **Rationale**: Existing upstream PR CI has extensive Linux coverage and focused Windows tests, while the general Windows build smoke is push-to-main only. The fork requires pre-merge evidence for both platforms on the same candidate.
- **Alternatives considered**: Rely only on upstream CI or on the release workflow. Rejected because neither gives the required fork PR candidate evidence without a release-side effect.

## Decision 5: Validate Identity Without Publishing

- **Decision**: Define fork candidates as `<upstream-version>-ruslan.<sequence>+upstream.<short-sha>` and validate the identity/provenance record locally.
- **Rationale**: A visible fork marker prevents confusion with upstream while the build metadata preserves upstream ancestry. Validation can be proven without tags, releases, installers, or channel changes.
- **Alternatives considered**: Reuse the upstream version verbatim or immediately modify GoReleaser publishing ownership. Rejected because the former is ambiguous and the latter crosses the explicit release gate.

## Decision 6: Preserve Generated Spec Kitty State Narrowly

- **Decision**: Keep repository-local configuration, canonical event metadata, agent/profile command manifests, merge attributes, and orientation; ignore machine-local runtime, cache, workspace, and derived files.
- **Rationale**: Future missions need reproducible project identity and command routing, but generated host state must not pollute the fork.
- **Alternatives considered**: Commit every generated file or discard Spec Kitty initialization. Rejected because the first is noisy and host-coupled, while the second makes this mission non-reproducible.

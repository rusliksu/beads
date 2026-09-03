# Fork Release Identity

This document defines the unpublished identity record for a candidate from
Ruslan's Beads fork. It supplements the
[fork-maintenance guide](FORK_MAINTENANCE.md) and does not replace the upstream
[release process](../RELEASING.md).

Identity validation proves that a proposed name and its immutable provenance
fit the fork contract. It does not approve, build, publish, or install a
release.

## Canonical identity

The candidate version has this shape:

```text
<upstream-version>-ruslan.<sequence>+upstream.<short-upstream-sha>
```

For example:

```text
1.2.3-ruslan.1+upstream.abcdef0
```

The fields have one relationship, not four independent meanings:

- `upstream_version` is the stable SemVer base, such as `1.2.3`.
- `sequence` is a positive fork-candidate number for that upstream base. It is
  not an upstream patch number.
- `fork_sha` is the exact 40-character lowercase commit for the candidate.
- `upstream_sha` is the exact 40-character lowercase provenance commit. The
  7-12 character SHA in version build metadata must prefix this value.

Thus the example above is valid with an upstream SHA beginning `abcdef0` and
an exact fork SHA. Reusing `1.2.3`, changing `ruslan` to another owner,
providing a branch name, or providing a non-matching upstream prefix is invalid.
The machine-readable shape is authoritative in the Mission
[`fork-release-identity/v1` schema](../kitty-specs/ruslan-beads-fork-authority-01M1HSDR/contracts/fork-release-identity.schema.json).

## Validation-only helper

Run the helper from a task-owned checkout and write evidence to a disposable or
review-owned path:

```powershell
pwsh -NoProfile -File scripts/fork-release-identity.ps1 `
  -CandidateVersion '1.2.3-ruslan.1+upstream.abcdef0' `
  -UpstreamVersion '1.2.3' `
  -ForkSha '0123456789abcdef0123456789abcdef01234567' `
  -UpstreamSha 'abcdef0123456789abcdef0123456789abcdef01' `
  -OutputPath "$env:TEMP/beads-fork-release-identity.json"
```

A successful run atomically writes exactly one
`fork-release-identity/v1` JSON record. Its `published` and `installed` fields
are always `false`. Repeating the same inputs produces the same record.
Validation treats SHAs as immutable values; it does not resolve branch names,
fetch remotes, or infer readiness from repository state.

## Evidence required before a future release gate

An identity proposal is incomplete unless review can inspect all of the
following for the same exact candidate commit:

1. A `fork-sync-manifest/v1` record with pinned fork and upstream SHAs, complete
   range and blocker evidence, and an accepted behavior-equivalence decision.
2. Executed Windows and Linux canary records for the candidate SHA. Both build
   and smoke results must pass; skipped, canceled, billing-blocked, missing, or
   infrastructure-only results are unexecuted, not green.
3. The `fork-release-identity/v1` record with the candidate version, positive
   sequence, exact fork SHA, exact upstream SHA, and matching metadata prefix.
4. Review confirmation that the aggregate diff contains no unrelated product
   divergence and that contributor prior art and attribution are handled.

These inputs make the proposal reviewable. They do not themselves open the
publication or installation gates.

## Side effects prohibited in this Mission

Neither successful validation nor completed canary evidence authorizes:

- creating, changing, or deleting a Git tag, branch, commit, or remote;
- pushing, merging, opening a GitHub release, or invoking GoReleaser;
- changing version sources, upstream release workflows, or release ownership;
- building or publishing a package, installer, or distribution-channel entry;
- changing `PATH`, aliases, executable locations, or the active `bd`;
- copying a candidate executable over any installed binary;
- reading or logging tokens, credentials, or secret-bearing remote URLs.

Publication requires a separately approved future Mission. Activating or
installing any resulting artifact is a later, independent environment gate.

## Future gate decisions

A future release operator must make explicit decisions about the identity,
candidate commit, upstream provenance, evidence completeness, artifact/channel
ownership, publication target, rollback owner, and whether installation should
remain closed. Publication tooling and commands belong in that later reviewed
scope; they are deliberately absent here.

If upstream publishes a newer version while a candidate is open, do not rename,
rebase, or silently expand the pinned candidate:

- keep the existing record pinned and mark it stale or blocked if the newer
  upstream state must be adopted;
- prepare a new sync candidate and new evidence from the selected newer
  upstream commit;
- never overwrite an existing identity record or reuse a sequence that could
  collide; select a new sequence through review;
- if the older upstream base remains intentionally selected, preserve its exact
  provenance and document that decision rather than implying latest-upstream
  status.

Because identities produced by this Mission are unpublished and uninstalled,
rollback means abandoning the proposal and its ephemeral evidence. There is no
tag, channel, package, alias, or active binary to roll back. Any rollback after
publication belongs to the separately authorized release runbook and gate.

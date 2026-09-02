# Quickstart: Validate the Fork-Authority Design

This quickstart is a no-publish, no-install validation path. It does not create a release, merge a pull request, or replace the active `bd`.

## 1. Confirm Repository Roles

From a disposable clone or task-owned checkout:

```powershell
git remote -v
git status --short --branch
```

Expected roles:

- `origin` points to `rusliksu/beads`.
- `upstream` points to `gastownhall/beads`.
- Work is on a task branch, never directly on fork `main`.

Do not print credential-bearing URLs. If a remote URL contains embedded user information, fix the remote configuration before collecting evidence.

## 2. Run the Sync Preflight Without Fetching

After implementation, the bounded validation command will be:

```powershell
pwsh -NoProfile -File scripts/fork-sync-preflight.ps1 `
  -ForkRef refs/remotes/origin/main `
  -UpstreamRef refs/remotes/upstream/main `
  -OutputPath "$env:TEMP\beads-fork-sync-manifest.json" `
  -NoFetch
```

Verify that the output records exact SHAs, counts both sides of the comparison, discloses conflicts/blockers, and does not change the current branch or worktree.

## 3. Validate a Fork Release Identity Without Publishing

Use an example identity and immutable SHAs:

```powershell
pwsh -NoProfile -File scripts/fork-release-identity.ps1 `
  -CandidateVersion '1.2.3-ruslan.1+upstream.abcdef0' `
  -UpstreamVersion '1.2.3' `
  -ForkSha '0123456789abcdef0123456789abcdef01234567' `
  -UpstreamSha 'abcdef0123456789abcdef0123456789abcdef01' `
  -OutputPath "$env:TEMP\beads-fork-release-identity.json"
```

Expected: a valid `fork-release-identity/v1` record with `published=false` and `installed=false`. No Git tag, GitHub release, package, or binary installation is created.

## 4. Run Local Contract Tests

```powershell
go test ./scripts -run 'TestFork(SyncPreflight|ReleaseIdentity|CanaryWorkflow)' -count=1
```

Tests must use disposable repositories and must not initialize or mutate the real Beads database.

## 5. Remote Canary Gate

Only after a parent/operator separately authorizes branch push and PR creation:

- the fork-only workflow checks out the exact PR commit;
- Windows and Linux legs build with the required tag;
- each leg runs `version` and `help` smoke checks;
- each leg records a checksum and commit-bound manifest;
- both jobs must actually execute and pass.

An unexecuted, skipped, canceled, timed-out-before-test, or billing-blocked job is not success. Merge, release publication, and active installation remain closed gates.

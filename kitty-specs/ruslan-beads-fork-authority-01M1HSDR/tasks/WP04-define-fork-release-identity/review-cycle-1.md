---
affected_files: []
cycle_number: 1
mission_slug: ruslan-beads-fork-authority-01M1HSDR
reproduction_command:
reviewed_at: '2026-09-03T06:07:39Z'
reviewer_agent: user
wp_id: WP04
---

**Issue 1 — valid identity collisions overwrite existing evidence**

`scripts/fork-release-identity.ps1:88` calls
`[System.IO.File]::Move($temporaryPath, $fullOutputPath, $true)`, so a fully
valid invocation replaces an existing identity record and exits successfully.
An independent bounded Windows check pre-created the output with a sentinel,
ran the valid documented identity, and observed `exit_code=0`,
`timed_out=false`, and `sentinel_preserved=false`.

This contradicts the canonical collision rule in
`engdocs/FORK_RELEASE_IDENTITY.md:109`: "never overwrite an existing identity
record". The current adversarial test protects an existing output only when
input validation fails (`scripts/fork_release_identity_test.go:127`), so it
does not constrain the valid-collision path.

Fix the helper to fail closed when `OutputPath` already exists, without
replacing or modifying that file, while retaining atomic creation for a new
path. Add a production-path test that pre-creates a sentinel, supplies valid
inputs, requires a non-zero bounded exit, and proves the sentinel is byte-for-
byte unchanged. Also keep the valid-new-path and repeatability tests green.

The remaining reviewed surface was acceptable: test-first commit ordering,
the exact three owned product files, PowerShell parse, schema fields and
cross-field validation, fail-closed invalid-input cases, explicit unpublished
and uninstalled provenance, and absence of release/install/tag/remote actions.
Go, `make`, Linux PowerShell, `jq`, the `no_coverage` pre-review outcome, the
synthetic JUnit baseline failure, and unchecked Definition-of-Done boxes remain
explicitly unexecuted or non-green and must not be represented as coverage.

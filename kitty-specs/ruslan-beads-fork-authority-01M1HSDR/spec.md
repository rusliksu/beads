# Mission Specification: Ruslan Beads Fork Authority

**Mission Branch**: `codex/beads-fork-authority`  
**Created**: 2026-09-02  
**Status**: Draft  
**Input**: Establish `rusliksu/beads` as Ruslan's behavior-identical Beads distribution, with `gastownhall/beads` retained as upstream provenance and every adoption, release, or installation kept behind an explicit gate.

## Intent Summary

Ruslan is the primary operator and maintainer. When an upstream Beads change is worth adopting, he needs a reproducible path that starts from a known upstream commit, produces a reviewable synchronization pull request, proves the resulting fork on Windows and Linux, and leaves the stable fork branch and installed `bd` unchanged until their separate gates pass. The canonical terms are **fork authority** for `rusliksu/beads`, **upstream provenance** for `gastownhall/beads`, and **sync candidate** for an unmerged upstream adoption proposal.

## User Scenarios & Testing

### User Story 1 - Establish Fork Authority (Priority: P1)

As Ruslan, I want the fork to declare its own stable authority and its external upstream so that future work has an unambiguous source, destination, and ownership boundary.

**Why this priority**: Every later sync, build, release, and support decision depends on knowing which repository is authoritative for Ruslan's distribution.

**Independent Test**: From a clean clone, an operator can identify the fork authority, the upstream provenance, the stable branch, and the rule that upstream changes enter only through review.

**Acceptance Scenarios**:

1. **Given** a clean clone of `rusliksu/beads`, **When** an operator reads the fork-maintenance guidance, **Then** `origin` is identified as Ruslan's authority, `gastownhall/beads` as upstream provenance, and fork `main` as the stable branch.
2. **Given** a proposed direct change to fork `main`, **When** the governance rules are applied, **Then** the change is rejected in favor of a task branch and reviewed pull request.

---

### User Story 2 - Review an Upstream Sync Candidate (Priority: P1)

As Ruslan, I want each upstream update to arrive as a traceable sync candidate so that I can inspect exactly what changed before accepting it into the fork.

**Why this priority**: Independent maintenance is unsafe if upstream updates can silently rewrite the fork or mix unrelated product divergence into synchronization.

**Independent Test**: A candidate records its upstream base, fork base, included range, review status, and validation evidence; a failed or ambiguous candidate leaves fork `main` unchanged.

**Acceptance Scenarios**:

1. **Given** fork `main` and upstream `main` at known commits, **When** a sync candidate is prepared, **Then** the candidate identifies both commits and exposes the complete adoption diff for review.
2. **Given** a conflict, unexpected product divergence, or incomplete evidence, **When** the candidate is evaluated, **Then** it remains unmerged and the stable fork branch is unchanged.
3. **Given** a candidate that passed review and required validation, **When** an authorized maintainer accepts it, **Then** the fork records the accepted upstream provenance through the pull-request history.

---

### User Story 3 - Prove Cross-Platform Buildability (Priority: P1)

As Ruslan, I want every adoption or fork release candidate checked on Windows and Linux so that the distribution remains usable across the environments he controls.

**Why this priority**: Windows portability is a load-bearing reason for owning the fork, while Linux compatibility must remain intact for upstream alignment and automation.

**Independent Test**: The same candidate commit receives successful Windows and Linux build canary results, with failures blocking merge or release rather than being treated as success.

**Acceptance Scenarios**:

1. **Given** a sync or release candidate commit, **When** the build canary runs, **Then** both Windows and Linux results are attached to that exact commit.
2. **Given** either platform is not run, fails, or reports an infrastructure-only non-result, **When** readiness is assessed, **Then** the candidate is not described as cross-platform proven.

---

### User Story 4 - Identify Fork Releases Without Replacing Active bd (Priority: P2)

As Ruslan, I want fork-produced release candidates to carry an explicit fork identity and provenance so that they cannot be confused with upstream releases or the currently installed `bd`.

**Why this priority**: Clear identity enables future self-maintained distribution while preventing an experimental artifact from silently becoming the active tool.

**Independent Test**: A proposed release identity is visibly fork-specific, references both fork and upstream commits, and produces no installation or public release side effect in this mission.

**Acceptance Scenarios**:

1. **Given** a fork release candidate, **When** its identity record is inspected, **Then** it is distinguishable from an upstream release and traces to exact fork and upstream commits.
2. **Given** successful build evidence, **When** this mission completes, **Then** no release is published and the active `bd` installation remains unchanged.

### Edge Cases

- Upstream force-pushes or rewrites the expected source commit before the candidate is reviewed.
- Fork `main` contains intentional behavior divergence that cannot be separated from an upstream synchronization diff.
- One build platform reports success while the other is skipped, canceled, billing-blocked, or fails before executing tests.
- The fork candidate builds, but its version string or artifact name is indistinguishable from upstream.
- A newer upstream commit appears while a candidate is under review; the candidate remains pinned rather than silently expanding its range.
- The current machine has an installed `bd`; build validation must not overwrite or shadow that executable.

## Requirements

### Functional Requirements

| ID | Title | User Story | Priority | Status |
|----|-------|------------|----------|--------|
| FR-001 | Declare fork authority | As Ruslan, I want `rusliksu/beads` and its stable `main` branch declared as the authority for my distribution while `gastownhall/beads` remains upstream provenance. | High | Open |
| FR-002 | Preserve upstream provenance | As a maintainer, I want every sync candidate to record the exact upstream source commit and fork base commit so that adoption is traceable. | High | Open |
| FR-003 | Require reviewed sync pull requests | As Ruslan, I want upstream changes to enter fork `main` only through a task branch and reviewed pull request so that stable history cannot change silently. | High | Open |
| FR-004 | Expose candidate scope | As a reviewer, I want the complete upstream range, local-only changes, conflicts, and validation state visible before acceptance so that I can make an informed decision. | High | Open |
| FR-005 | Run two-platform build canary | As Ruslan, I want the exact candidate commit checked on Windows and Linux so that portability claims are evidence-backed. | High | Open |
| FR-006 | Block incomplete candidates | As Ruslan, I want conflicts, unreviewed divergence, failed checks, or unexecuted canaries to block acceptance so that missing evidence is never reported as success. | High | Open |
| FR-007 | Define fork release identity | As a release operator, I want a fork-specific identity that traces to fork and upstream commits so that fork artifacts cannot be mistaken for upstream artifacts. | Medium | Open |
| FR-008 | Keep active installation unchanged | As Ruslan, I want planning and candidate validation to leave the installed `bd` unchanged until a later explicit installation gate. | High | Open |

### Non-Functional Requirements

| ID | Title | Requirement | Category | Priority | Status |
|----|-------|-------------|----------|----------|--------|
| NFR-001 | Complete traceability | 100% of sync and release candidates identify an immutable fork commit, immutable upstream commit, creation time, and review or validation outcome. | Auditability | High | Open |
| NFR-002 | Cross-platform evidence | 100% of candidates described as build-ready have successful checks for the same commit on at least one supported Windows runner and one supported Linux runner. | Portability | High | Open |
| NFR-003 | Reproducible clean-clone check | A maintainer following the documented canary from a clean clone can reproduce the candidate build result without modifying the machine's active `bd`. | Reproducibility | High | Open |
| NFR-004 | Stable-branch protection | 100% of upstream adoptions and fork-specific changes reach fork `main` through reviewable pull requests; direct agent pushes to `main` are zero. | Governance | High | Open |
| NFR-005 | Behavior-equivalence visibility | Each upstream sync candidate explicitly reports whether fork behavior remains identical; any intentional difference is zero for this mission or is rejected into a separate mission. | Compatibility | High | Open |

### Constraints

| ID | Title | Constraint | Category | Priority | Status |
|----|-------|------------|----------|----------|--------|
| C-001 | Repository roles | `rusliksu/beads` is the fork authority and `gastownhall/beads` is upstream provenance; these roles must not be reversed or conflated. | Governance | High | Open |
| C-002 | Behavior-identical first | This mission introduces no Beads product behavior, storage-schema, issue-model, or orchestration-policy divergence from upstream. | Product | High | Open |
| C-003 | Pull-request-only adoption | No direct push or automatic merge to fork `main` is allowed; merge remains an explicit operator action after review. | Governance | High | Open |
| C-004 | No release | This mission may define and validate release identity but must not publish a tag, release, package, binary, or distribution channel. | Delivery | High | Open |
| C-005 | No installation replacement | This mission must not install, overwrite, alias, or activate a fork-built `bd`. | Environment | High | Open |
| C-006 | Preserve Beads product boundary | Fork maintenance must not move orchestration policy, agent routing, retry policy, or workflow semantics into Beads core. | Architecture | High | Open |
| C-007 | Contributor protection | Before implementation or PR handling, the workflow must run the repository's contributor/PR preflight and preserve relevant upstream contributor work and attribution. | Governance | High | Open |

### Key Entities

- **Fork Authority**: Ruslan-owned repository and stable branch that define the accepted distribution state.
- **Upstream Provenance**: External repository and immutable commit range from which candidate changes originate.
- **Sync Candidate**: Reviewable proposal that combines a pinned fork base, pinned upstream range, disclosed conflicts or divergence, and validation evidence.
- **Build Canary Evidence**: Platform-specific outcome tied to one exact candidate commit; an unexecuted check is not a passing result.
- **Fork Release Identity**: Distinguishable candidate identity that links a fork commit to its upstream provenance without publishing or installing it.

## Assumptions

- The initial fork is behavior-identical to the selected upstream `main` commit.
- GitHub pull requests are the review surface for fork changes and upstream sync candidates.
- The current mission prepares the governance and validation contract; publishing releases and changing the active installation are later, separately authorized missions or gates.
- The repository's existing `engdocs/PROJECT_CHARTER.md`, contributor policy, and storage boundary remain authoritative; the project-local Spec Kitty charter is not yet initialized.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A fresh maintainer can identify fork authority, upstream provenance, stable branch, and PR-only adoption rule from one canonical maintenance entry point without prior conversation.
- **SC-002**: A dry-run sync candidate records exact fork and upstream commits, shows its full scope, and leaves fork `main` unchanged.
- **SC-003**: The exact same candidate commit receives successful, independently inspectable Windows and Linux build canary outcomes before any readiness claim.
- **SC-004**: A fork release candidate identity is visibly distinct from upstream and is traceable to both fork and upstream commits, while zero releases are published.
- **SC-005**: Completion of the mission causes zero changes to the active `bd` executable, aliases, or installation paths.
- **SC-006**: Every planned implementation work package maps to at least one functional requirement, and all eight functional requirements map to at least one work package.

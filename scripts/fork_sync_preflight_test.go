package scripts_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

type forkSyncManifest struct {
	Schema              string   `json:"schema"`
	GeneratedAt         string   `json:"generated_at"`
	ForkRepository      string   `json:"fork_repository"`
	ForkBaseSHA         string   `json:"fork_base_sha"`
	UpstreamRepository  string   `json:"upstream_repository"`
	UpstreamHeadSHA     string   `json:"upstream_head_sha"`
	MergeBaseSHA        string   `json:"merge_base_sha"`
	ForkOnlyCommits     int      `json:"fork_only_commits"`
	UpstreamOnlyCommits int      `json:"upstream_only_commits"`
	Conflicts           []string `json:"conflicts"`
	BehaviorEquivalence string   `json:"behavior_equivalence"`
	CandidateStatus     string   `json:"candidate_status"`
	Blockers            []string `json:"blockers"`
}

type forkSyncFixture struct {
	dir  string
	base string
}

type forkSyncRun struct {
	manifest forkSyncManifest
	raw      map[string]json.RawMessage
	output   string
	err      error
}

func TestForkSyncPreflightClassifiesPinnedComparisons(t *testing.T) {
	t.Run("identical refs", func(t *testing.T) {
		fixture := newForkSyncFixture(t)
		run := runForkSyncPreflight(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/remotes/upstream/main")
		requireForkSyncSuccess(t, run)
		if run.manifest.CandidateStatus != "already_identical" {
			t.Fatalf("candidate_status = %q, want already_identical", run.manifest.CandidateStatus)
		}
		if run.manifest.ForkOnlyCommits != 0 || run.manifest.UpstreamOnlyCommits != 0 {
			t.Fatalf("directional counts = fork:%d upstream:%d, want 0/0", run.manifest.ForkOnlyCommits, run.manifest.UpstreamOnlyCommits)
		}
	})

	t.Run("upstream-only range with explicit equivalence", func(t *testing.T) {
		fixture := newForkSyncFixture(t)
		upstream := fixture.commitFrom(t, fixture.base, "upstream.txt", "upstream\n")
		fixture.setRef(t, "refs/remotes/upstream/main", upstream)

		run := runForkSyncPreflight(t, fixture.dir, "unchanged", "refs/remotes/origin/main", "refs/remotes/upstream/main")
		requireForkSyncSuccess(t, run)
		if run.manifest.CandidateStatus != "ready_for_pr" {
			t.Fatalf("candidate_status = %q, want ready_for_pr; blockers=%v", run.manifest.CandidateStatus, run.manifest.Blockers)
		}
		if run.manifest.ForkOnlyCommits != 0 || run.manifest.UpstreamOnlyCommits != 1 {
			t.Fatalf("directional counts = fork:%d upstream:%d, want 0/1", run.manifest.ForkOnlyCommits, run.manifest.UpstreamOnlyCommits)
		}
	})

	t.Run("fork-only range stays conservative", func(t *testing.T) {
		fixture := newForkSyncFixture(t)
		fork := fixture.commitFrom(t, fixture.base, "fork.txt", "fork\n")
		fixture.setRef(t, "refs/remotes/origin/main", fork)

		run := runForkSyncPreflight(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/remotes/upstream/main")
		requireForkSyncSuccess(t, run)
		if run.manifest.CandidateStatus != "blocked" || run.manifest.BehaviorEquivalence != "unknown" {
			t.Fatalf("fork divergence classified as status=%q equivalence=%q", run.manifest.CandidateStatus, run.manifest.BehaviorEquivalence)
		}
		requireString(t, run.manifest.Blockers, "fork_only_commits")

		explicit := runForkSyncPreflight(t, fixture.dir, "unchanged", "refs/remotes/origin/main", "refs/remotes/upstream/main")
		requireForkSyncSuccess(t, explicit)
		if explicit.manifest.CandidateStatus != "blocked" {
			t.Fatalf("explicit equivalence bypassed fork-only divergence: status=%q", explicit.manifest.CandidateStatus)
		}
		requireString(t, explicit.manifest.Blockers, "fork_only_commits")
	})

	t.Run("both sides changed without conflict stays blocked", func(t *testing.T) {
		fixture := newForkSyncFixture(t)
		fork := fixture.commitFrom(t, fixture.base, "fork.txt", "fork\n")
		upstream := fixture.commitFrom(t, fixture.base, "upstream.txt", "upstream\n")
		fixture.setRef(t, "refs/remotes/origin/main", fork)
		fixture.setRef(t, "refs/remotes/upstream/main", upstream)

		run := runForkSyncPreflight(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/remotes/upstream/main")
		requireForkSyncSuccess(t, run)
		if run.manifest.CandidateStatus != "blocked" || len(run.manifest.Conflicts) != 0 {
			t.Fatalf("clean divergence classified as status=%q conflicts=%v", run.manifest.CandidateStatus, run.manifest.Conflicts)
		}
		requireString(t, run.manifest.Blockers, "fork_only_commits")
		requireString(t, run.manifest.Blockers, "behavior_equivalence_unknown")
	})

	t.Run("content conflict carries sorted path evidence", func(t *testing.T) {
		fixture := newForkSyncFixture(t)
		fork := fixture.commitFrom(t, fixture.base, "shared.txt", "fork\n")
		upstream := fixture.commitFrom(t, fixture.base, "shared.txt", "upstream\n")
		fixture.setRef(t, "refs/remotes/origin/main", fork)
		fixture.setRef(t, "refs/remotes/upstream/main", upstream)

		run := runForkSyncPreflight(t, fixture.dir, "unchanged", "refs/remotes/origin/main", "refs/remotes/upstream/main")
		requireForkSyncSuccess(t, run)
		if run.manifest.CandidateStatus != "blocked" {
			t.Fatalf("conflict classified as %q, want blocked", run.manifest.CandidateStatus)
		}
		if !reflect.DeepEqual(run.manifest.Conflicts, []string{"shared.txt"}) {
			t.Fatalf("conflicts = %v, want [shared.txt]", run.manifest.Conflicts)
		}
		requireString(t, run.manifest.Blockers, "merge_conflicts")
	})
}

func TestForkSyncPreflightRejectsInvalidRefsDeterministically(t *testing.T) {
	fixture := newForkSyncFixture(t)

	t.Run("missing ref", func(t *testing.T) {
		first := runForkSyncPreflight(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/remotes/upstream/missing")
		second := runForkSyncPreflight(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/remotes/upstream/missing")
		requireForkSyncFailure(t, first)
		requireForkSyncFailure(t, second)
		if first.output != second.output {
			t.Fatalf("missing-ref diagnostic is not deterministic:\nfirst: %q\nsecond: %q", first.output, second.output)
		}
		if !strings.Contains(first.output, "upstream ref does not resolve to a commit") {
			t.Fatalf("missing sanitized ref diagnostic: %s", first.output)
		}
	})

	t.Run("non-commit ref", func(t *testing.T) {
		blob := strings.TrimSpace(fixture.git(t, []byte("blob\n"), "hash-object", "-w", "--stdin"))
		fixture.setRef(t, "refs/tags/not-a-commit", blob)

		run := runForkSyncPreflight(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/tags/not-a-commit")
		requireForkSyncFailure(t, run)
		if !strings.Contains(run.output, "upstream ref does not resolve to a commit") {
			t.Fatalf("non-commit ref did not produce the bounded diagnostic: %s", run.output)
		}
	})
}

func TestForkSyncPreflightDoesNotExposeCredentialBearingRemotes(t *testing.T) {
	fixture := newForkSyncFixture(t)
	fixture.git(t, nil, "remote", "add", "origin", "https://user:supersecret@example.invalid/rusliksu/beads.git")
	fixture.git(t, nil, "remote", "add", "upstream", "https://oauth2:upstreamtoken@example.invalid/gastownhall/beads.git")

	run := runForkSyncPreflight(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/remotes/upstream/main")
	requireForkSyncSuccess(t, run)
	evidence := run.output + marshalForkSyncManifest(t, run.manifest)
	for _, forbidden := range []string{"supersecret", "upstreamtoken", "user:", "oauth2:", "example.invalid"} {
		if strings.Contains(evidence, forbidden) {
			t.Fatalf("credential-bearing remote material %q leaked into evidence: %s", forbidden, evidence)
		}
	}
}

func TestForkSyncPreflightNoFetchLeavesRepositoryStateUnchanged(t *testing.T) {
	fixture := newForkSyncFixture(t)
	fixture.git(t, nil, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing-origin.git"))
	fixture.git(t, nil, "remote", "add", "upstream", filepath.Join(t.TempDir(), "missing-upstream.git"))
	before := fixture.snapshot(t)

	run := runForkSyncPreflight(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/remotes/upstream/main")
	requireForkSyncSuccess(t, run)
	withoutFlag := runForkSyncPreflightWithFlag(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/remotes/upstream/main", false)
	requireForkSyncSuccess(t, withoutFlag)
	afterBothModes := fixture.snapshot(t)
	if before != afterBothModes {
		t.Fatalf("repository state changed with explicit or default no-fetch semantics:\n--- before ---\n%s\n--- after ---\n%s", before, afterBothModes)
	}
}

func TestForkSyncPreflightManifestSchemaAndRepeatability(t *testing.T) {
	fixture := newForkSyncFixture(t)
	first := runForkSyncPreflight(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/remotes/upstream/main")
	second := runForkSyncPreflight(t, fixture.dir, "unknown", "refs/remotes/origin/main", "refs/remotes/upstream/main")
	requireForkSyncSuccess(t, first)
	requireForkSyncSuccess(t, second)

	shaPattern := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for field, value := range map[string]string{
		"fork_base_sha":      first.manifest.ForkBaseSHA,
		"upstream_head_sha": first.manifest.UpstreamHeadSHA,
		"merge_base_sha":    first.manifest.MergeBaseSHA,
	} {
		if !shaPattern.MatchString(value) {
			t.Errorf("%s = %q, want a lowercase 40-character SHA", field, value)
		}
	}
	if first.manifest.Schema != "fork-sync-manifest/v1" || first.manifest.ForkRepository != "rusliksu/beads" || first.manifest.UpstreamRepository != "gastownhall/beads" {
		t.Fatalf("manifest identity fields are wrong: %+v", first.manifest)
	}
	if _, err := time.Parse(time.RFC3339Nano, first.manifest.GeneratedAt); err != nil {
		t.Fatalf("generated_at = %q, want RFC3339 timestamp: %v", first.manifest.GeneratedAt, err)
	}
	expectedFields := []string{
		"schema",
		"generated_at",
		"fork_repository",
		"fork_base_sha",
		"upstream_repository",
		"upstream_head_sha",
		"merge_base_sha",
		"fork_only_commits",
		"upstream_only_commits",
		"conflicts",
		"behavior_equivalence",
		"candidate_status",
		"blockers",
	}
	if len(first.raw) != len(expectedFields) {
		t.Fatalf("manifest has %d fields, want %d: %v", len(first.raw), len(expectedFields), first.raw)
	}
	for _, field := range expectedFields {
		if _, ok := first.raw[field]; !ok {
			t.Errorf("manifest is missing required field %q", field)
		}
	}

	first.manifest.GeneratedAt = ""
	second.manifest.GeneratedAt = ""
	if !reflect.DeepEqual(first.manifest, second.manifest) {
		t.Fatalf("same refs produced semantically different manifests:\nfirst: %+v\nsecond: %+v", first.manifest, second.manifest)
	}
}

func newForkSyncFixture(t *testing.T) *forkSyncFixture {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, nil, "init", "--initial-branch=main")
	runGit(t, dir, nil, "config", "core.hooksPath", ".git/hooks")
	runGit(t, dir, nil, "config", "user.name", "Fork Sync Test")
	runGit(t, dir, nil, "config", "user.email", "fork-sync-test@example.invalid")
	writeForkSyncFile(t, dir, "shared.txt", "base\n")
	runGit(t, dir, nil, "add", "shared.txt")
	runGit(t, dir, nil, "commit", "-m", "base")
	base := strings.TrimSpace(runGit(t, dir, nil, "rev-parse", "HEAD"))
	fixture := &forkSyncFixture{dir: dir, base: base}
	fixture.setRef(t, "refs/remotes/origin/main", base)
	fixture.setRef(t, "refs/remotes/upstream/main", base)
	return fixture
}

func (fixture *forkSyncFixture) commitFrom(t *testing.T, parent, path, body string) string {
	t.Helper()
	fixture.git(t, nil, "checkout", "--detach", "--quiet", parent)
	writeForkSyncFile(t, fixture.dir, path, body)
	fixture.git(t, nil, "add", "--all")
	fixture.git(t, nil, "commit", "--quiet", "-m", "fixture change")
	commit := strings.TrimSpace(fixture.git(t, nil, "rev-parse", "HEAD"))
	fixture.git(t, nil, "checkout", "--quiet", "main")
	return commit
}

func (fixture *forkSyncFixture) setRef(t *testing.T, ref, commit string) {
	t.Helper()
	fixture.git(t, nil, "update-ref", ref, commit)
}

func (fixture *forkSyncFixture) git(t *testing.T, stdin []byte, args ...string) string {
	t.Helper()
	return runGit(t, fixture.dir, stdin, args...)
}

func (fixture *forkSyncFixture) snapshot(t *testing.T) string {
	t.Helper()
	var snapshot strings.Builder
	for _, command := range [][]string{
		{"status", "--porcelain=v1", "-z"},
		{"rev-parse", "HEAD"},
		{"symbolic-ref", "-q", "HEAD"},
		{"write-tree"},
		{"for-each-ref", "--format=%(refname) %(objectname)", "refs/remotes"},
	} {
		fmt.Fprintf(&snapshot, "%q\n%s\n", command, fixture.git(t, nil, command...))
	}
	return snapshot.String()
}

func runForkSyncPreflight(t *testing.T, repo, equivalence, forkRef, upstreamRef string) forkSyncRun {
	t.Helper()
	return runForkSyncPreflightWithFlag(t, repo, equivalence, forkRef, upstreamRef, true)
}

func runForkSyncPreflightWithFlag(t *testing.T, repo, equivalence, forkRef, upstreamRef string, includeNoFetch bool) forkSyncRun {
	t.Helper()
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		t.Fatalf("PowerShell 7 is required for the fork sync contract: %v", err)
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve fork sync test source path")
	}
	script := filepath.Join(filepath.Dir(sourceFile), "fork-sync-preflight.ps1")
	outputPath := filepath.Join(t.TempDir(), "fork-sync-manifest.json")
	arguments := []string{
		"-NoLogo",
		"-NoProfile",
		"-NonInteractive",
		"-File", script,
		"-ForkRef", forkRef,
		"-UpstreamRef", upstreamRef,
		"-OutputPath", outputPath,
		"-BehaviorEquivalence", equivalence,
	}
	if includeNoFetch {
		arguments = append(arguments, "-NoFetch")
	}
	cmd := exec.Command(pwsh, arguments...)
	cmd.Dir = repo
	combined, runErr := cmd.CombinedOutput()
	run := forkSyncRun{output: string(combined), err: runErr}
	if runErr != nil {
		return run
	}
	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read fork sync manifest: %v\ncommand output: %s", err, combined)
	}
	if err := json.Unmarshal(body, &run.manifest); err != nil {
		t.Fatalf("decode fork sync manifest: %v\nmanifest: %s\ncommand output: %s", err, body, combined)
	}
	if err := json.Unmarshal(body, &run.raw); err != nil {
		t.Fatalf("decode raw fork sync manifest: %v\nmanifest: %s", err, body)
	}
	return run
}

func requireForkSyncSuccess(t *testing.T, run forkSyncRun) {
	t.Helper()
	if run.err != nil {
		t.Fatalf("fork sync preflight failed: %v\n%s", run.err, run.output)
	}
}

func requireForkSyncFailure(t *testing.T, run forkSyncRun) {
	t.Helper()
	if run.err == nil {
		t.Fatalf("fork sync preflight unexpectedly succeeded: %+v\n%s", run.manifest, run.output)
	}
	var exitErr *exec.ExitError
	if !errors.As(run.err, &exitErr) || exitErr.ExitCode() == 0 {
		t.Fatalf("fork sync preflight did not return a deterministic non-zero process exit: %v\n%s", run.err, run.output)
	}
}

func requireString(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("%q not found in %v", want, values)
}

func marshalForkSyncManifest(t *testing.T, manifest forkSyncManifest) string {
	t.Helper()
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal fork sync manifest: %v", err)
	}
	return string(body)
}

func runGit(t *testing.T, dir string, stdin []byte, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(string(stdin))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func writeForkSyncFile(t *testing.T, root, relative, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture parent for %s: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", relative, err)
	}
}

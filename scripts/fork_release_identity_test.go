package scripts_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

const forkReleaseIdentityProcessTimeout = 15 * time.Second

const (
	validForkReleaseCandidate = "1.2.3-ruslan.1+upstream.abcdef0"
	validForkReleaseSHA       = "0123456789abcdef0123456789abcdef01234567"
	validUpstreamReleaseSHA   = "abcdef0123456789abcdef0123456789abcdef01"
)

type forkReleaseIdentity struct {
	Schema           string `json:"schema"`
	CandidateVersion string `json:"candidate_version"`
	UpstreamVersion  string `json:"upstream_version"`
	Sequence         int    `json:"sequence"`
	ForkSHA          string `json:"fork_sha"`
	UpstreamSHA      string `json:"upstream_sha"`
	Published        bool   `json:"published"`
	Installed        bool   `json:"installed"`
}

type forkReleaseIdentityInput struct {
	CandidateVersion string
	UpstreamVersion  string
	ForkSHA          string
	UpstreamSHA      string
}

type forkReleaseIdentityRun struct {
	manifest forkReleaseIdentity
	raw      map[string]json.RawMessage
	output   string
	err      error
	exitCode int
	timedOut bool
}

func TestForkReleaseIdentityAcceptsSchemaConformantCandidate(t *testing.T) {
	input := validForkReleaseIdentityInput()
	first := runForkReleaseIdentity(t, input, filepath.Join(t.TempDir(), "identity.json"))
	second := runForkReleaseIdentity(t, input, filepath.Join(t.TempDir(), "identity.json"))
	requireForkReleaseIdentitySuccess(t, first)
	requireForkReleaseIdentitySuccess(t, second)

	want := forkReleaseIdentity{
		Schema:           "fork-release-identity/v1",
		CandidateVersion: validForkReleaseCandidate,
		UpstreamVersion:  "1.2.3",
		Sequence:         1,
		ForkSHA:          validForkReleaseSHA,
		UpstreamSHA:      validUpstreamReleaseSHA,
		Published:        false,
		Installed:        false,
	}
	if !reflect.DeepEqual(first.manifest, want) {
		t.Fatalf("identity = %+v, want %+v", first.manifest, want)
	}
	if !reflect.DeepEqual(first.manifest, second.manifest) {
		t.Fatalf("repeat runs differ:\nfirst: %+v\nsecond: %+v", first.manifest, second.manifest)
	}

	expectedFields := []string{
		"schema",
		"candidate_version",
		"upstream_version",
		"sequence",
		"fork_sha",
		"upstream_sha",
		"published",
		"installed",
	}
	if len(first.raw) != len(expectedFields) {
		t.Fatalf("identity has %d fields, want %d: %v", len(first.raw), len(expectedFields), first.raw)
	}
	for _, field := range expectedFields {
		if _, ok := first.raw[field]; !ok {
			t.Errorf("identity is missing required field %q", field)
		}
	}
}

func TestForkReleaseIdentityRejectsExistingOutputWithoutModification(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "identity.json")
	sentinel := []byte("existing identity evidence must remain byte-for-byte unchanged\n")
	if err := os.WriteFile(outputPath, sentinel, 0o600); err != nil {
		t.Fatalf("write sentinel output: %v", err)
	}

	run := runForkReleaseIdentity(t, validForkReleaseIdentityInput(), outputPath)
	requireForkReleaseIdentityFailure(t, run)
	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read sentinel output: %v", err)
	}
	if !bytes.Equal(body, sentinel) {
		t.Fatalf("valid collision modified existing evidence:\n got: %q\nwant: %q", body, sentinel)
	}
}

func TestForkReleaseIdentityRejectsAmbiguousOrIncompleteInputs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*forkReleaseIdentityInput)
	}{
		{name: "upstream-identical version", mutate: func(input *forkReleaseIdentityInput) { input.CandidateVersion = "1.2.3" }},
		{name: "zero sequence", mutate: func(input *forkReleaseIdentityInput) { input.CandidateVersion = "1.2.3-ruslan.0+upstream.abcdef0" }},
		{name: "wrong owner marker", mutate: func(input *forkReleaseIdentityInput) { input.CandidateVersion = "1.2.3-other.1+upstream.abcdef0" }},
		{name: "missing build metadata", mutate: func(input *forkReleaseIdentityInput) { input.CandidateVersion = "1.2.3-ruslan.1" }},
		{name: "malformed candidate semver", mutate: func(input *forkReleaseIdentityInput) { input.CandidateVersion = "01.2.3-ruslan.1+upstream.abcdef0" }},
		{name: "malformed upstream semver", mutate: func(input *forkReleaseIdentityInput) { input.UpstreamVersion = "1.2" }},
		{name: "candidate base mismatch", mutate: func(input *forkReleaseIdentityInput) { input.UpstreamVersion = "1.2.4" }},
		{name: "uppercase fork sha", mutate: func(input *forkReleaseIdentityInput) { input.ForkSHA = strings.ToUpper(input.ForkSHA) }},
		{name: "uppercase upstream sha", mutate: func(input *forkReleaseIdentityInput) { input.UpstreamSHA = strings.ToUpper(input.UpstreamSHA) }},
		{name: "mutable fork ref", mutate: func(input *forkReleaseIdentityInput) { input.ForkSHA = "refs/heads/main" }},
		{name: "mutable upstream ref", mutate: func(input *forkReleaseIdentityInput) { input.UpstreamSHA = "refs/remotes/upstream/main" }},
		{name: "short upstream sha", mutate: func(input *forkReleaseIdentityInput) { input.UpstreamSHA = "abcdef0" }},
		{name: "metadata prefix mismatch", mutate: func(input *forkReleaseIdentityInput) { input.CandidateVersion = "1.2.3-ruslan.1+upstream.7654321" }},
		{name: "uppercase metadata sha", mutate: func(input *forkReleaseIdentityInput) { input.CandidateVersion = "1.2.3-ruslan.1+upstream.ABCDEF0" }},
		{name: "metadata sha too short", mutate: func(input *forkReleaseIdentityInput) { input.CandidateVersion = "1.2.3-ruslan.1+upstream.abcdef" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validForkReleaseIdentityInput()
			test.mutate(&input)
			outputPath := filepath.Join(t.TempDir(), "identity.json")
			const sentinel = "existing evidence must survive invalid input\n"
			if err := os.WriteFile(outputPath, []byte(sentinel), 0o600); err != nil {
				t.Fatalf("write sentinel output: %v", err)
			}

			run := runForkReleaseIdentity(t, input, outputPath)
			requireForkReleaseIdentityFailure(t, run)
			body, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("read sentinel output: %v", err)
			}
			if string(body) != sentinel {
				t.Fatalf("invalid input replaced existing evidence: %q", body)
			}
		})
	}
}

func TestForkReleaseIdentityLeavesRepositoryAndEnvironmentUnchanged(t *testing.T) {
	before := snapshotForkReleaseIdentityState(t)
	run := runForkReleaseIdentity(t, validForkReleaseIdentityInput(), filepath.Join(t.TempDir(), "identity.json"))
	requireForkReleaseIdentitySuccess(t, run)
	after := snapshotForkReleaseIdentityState(t)

	if before != after {
		t.Fatalf("identity validation changed repository or environment state:\n--- before ---\n%s\n--- after ---\n%s", before, after)
	}
}

func validForkReleaseIdentityInput() forkReleaseIdentityInput {
	return forkReleaseIdentityInput{
		CandidateVersion: validForkReleaseCandidate,
		UpstreamVersion:  "1.2.3",
		ForkSHA:          validForkReleaseSHA,
		UpstreamSHA:      validUpstreamReleaseSHA,
	}
}

func runForkReleaseIdentity(t *testing.T, input forkReleaseIdentityInput, outputPath string) forkReleaseIdentityRun {
	t.Helper()
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		t.Fatalf("PowerShell 7 is required for the fork release identity contract: %v", err)
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve fork release identity test source path")
	}
	script := filepath.Join(filepath.Dir(sourceFile), "fork-release-identity.ps1")
	arguments := []string{
		"-NoLogo",
		"-NoProfile",
		"-NonInteractive",
		"-File", script,
		"-CandidateVersion", input.CandidateVersion,
		"-UpstreamVersion", input.UpstreamVersion,
		"-ForkSha", input.ForkSHA,
		"-UpstreamSha", input.UpstreamSHA,
		"-OutputPath", outputPath,
	}
	ctx, cancel := context.WithTimeout(context.Background(), forkReleaseIdentityProcessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, pwsh, arguments...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	run := forkReleaseIdentityRun{
		output: stdout.String() + stderr.String(),
		err:    runErr,
	}
	if ctx.Err() == context.DeadlineExceeded {
		run.timedOut = true
		run.err = fmt.Errorf("fork release identity exceeded %s: %w", forkReleaseIdentityProcessTimeout, ctx.Err())
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			run.exitCode = exitErr.ExitCode()
		} else {
			run.exitCode = -1
		}
		return run
	}

	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read fork release identity: %v\noutput: %s", err, run.output)
	}
	if err := json.Unmarshal(body, &run.manifest); err != nil {
		t.Fatalf("decode fork release identity: %v\nidentity: %s\noutput: %s", err, body, run.output)
	}
	if err := json.Unmarshal(body, &run.raw); err != nil {
		t.Fatalf("decode raw fork release identity: %v\nidentity: %s", err, body)
	}
	return run
}

func requireForkReleaseIdentitySuccess(t *testing.T, run forkReleaseIdentityRun) {
	t.Helper()
	if run.timedOut {
		t.Fatalf("fork release identity timed out: %s", run.output)
	}
	if run.err != nil {
		t.Fatalf("fork release identity failed: %v\n%s", run.err, run.output)
	}
	if run.exitCode != 0 {
		t.Fatalf("successful identity run recorded exit code %d", run.exitCode)
	}
}

func requireForkReleaseIdentityFailure(t *testing.T, run forkReleaseIdentityRun) {
	t.Helper()
	if run.timedOut {
		t.Fatalf("fork release identity timed out instead of returning a bounded failure: %s", run.output)
	}
	if run.err == nil {
		t.Fatalf("fork release identity unexpectedly succeeded: %+v\n%s", run.manifest, run.output)
	}
	var exitErr *exec.ExitError
	if !errors.As(run.err, &exitErr) || exitErr.ExitCode() == 0 {
		t.Fatalf("fork release identity did not return a non-zero process exit: %v\n%s", run.err, run.output)
	}
}

func snapshotForkReleaseIdentityState(t *testing.T) string {
	t.Helper()
	var snapshot strings.Builder
	fmt.Fprintf(&snapshot, "PATH=%s\n", os.Getenv("PATH"))
	for _, args := range [][]string{
		{"rev-parse", "HEAD"},
		{"status", "--porcelain=v1", "-z"},
		{"for-each-ref", "--format=%(refname) %(objectname)"},
		{"tag", "--list"},
		{"remote", "-v"},
	} {
		fmt.Fprintf(&snapshot, "git %q\n%s\n", args, runForkReleaseSnapshotCommand(t, "git", args...))
	}
	aliasCommand := "Get-Alias | Sort-Object Name | ForEach-Object { '{0}|{1}' -f $_.Name, $_.Definition }"
	fmt.Fprintf(&snapshot, "aliases\n%s\n", runForkReleaseSnapshotCommand(t, "pwsh", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", aliasCommand))
	fmt.Fprintf(&snapshot, "installed-bd\n%s\n", installedBDMetadata(t))
	return snapshot.String()
}

func runForkReleaseSnapshotCommand(t *testing.T, name string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), forkReleaseIdentityProcessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("snapshot command %s %s timed out after %s", name, strings.Join(args, " "), forkReleaseIdentityProcessTimeout)
	}
	if err != nil {
		t.Fatalf("snapshot command %s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return string(output)
}

func installedBDMetadata(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("bd")
	if errors.Is(err, exec.ErrNotFound) {
		return "not-found"
	}
	if err != nil {
		t.Fatalf("resolve installed bd: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat installed bd: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("hash installed bd: %v", err)
	}
	sum := sha256.Sum256(body)
	return fmt.Sprintf("path=%s size=%d mode=%s modtime=%s sha256=%s", path, info.Size(), info.Mode(), info.ModTime().UTC().Format(time.RFC3339Nano), hex.EncodeToString(sum[:]))
}

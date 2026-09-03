package scripts_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const forkCanaryWorkflowPath = "../.github/workflows/fork-canary.yml"

func TestForkCanaryWorkflow(t *testing.T) {
	body, err := os.ReadFile(filepath.FromSlash(forkCanaryWorkflowPath))
	if err != nil {
		t.Fatalf("read fork canary workflow: %v (the workflow must exist before this contract can pass)", err)
	}

	if problems := validateForkCanaryWorkflow(string(body)); len(problems) != 0 {
		t.Fatalf("fork canary workflow contract violations:\n- %s", strings.Join(problems, "\n- "))
	}
}

func TestForkCanaryWorkflowRejectsMeaningfulMutations(t *testing.T) {
	body, err := os.ReadFile(filepath.FromSlash(forkCanaryWorkflowPath))
	if err != nil {
		t.Fatalf("read fork canary workflow: %v", err)
	}
	original := string(body)

	tests := []struct {
		name       string
		mutate     func(string) string
		wantNeedle string
	}{
		{
			name: "repository guard removed",
			mutate: func(in string) string {
				return strings.Replace(in, "if: github.repository == 'rusliksu/beads'", "if: success()", 1)
			},
			wantNeedle: "repository guard",
		},
		{
			name: "Windows leg removed",
			mutate: func(in string) string {
				return strings.Replace(in, "windows-latest", "macos-latest", 1)
			},
			wantNeedle: "windows-latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutated := tt.mutate(original)
			if mutated == original {
				t.Fatalf("mutation did not change workflow; fixture contract drifted")
			}
			problems := validateForkCanaryWorkflow(mutated)
			if !containsProblem(problems, tt.wantNeedle) {
				t.Fatalf("mutation was not rejected for %q; violations: %v", tt.wantNeedle, problems)
			}
		})
	}
}

func validateForkCanaryWorkflow(body string) []string {
	var problems []string
	require := func(ok bool, message string) {
		if !ok {
			problems = append(problems, message)
		}
	}

	require(regexp.MustCompile(`(?m)^on:\s*$`).MatchString(body), "top-level on trigger is required")
	require(regexp.MustCompile(`(?m)^\s{2}pull_request:\s*$`).MatchString(body), "pull_request trigger is required")
	require(regexp.MustCompile(`(?m)^\s{4}branches:\s*\[\s*main\s*\]\s*$`).MatchString(body), "pull_request must target fork main")
	require(regexp.MustCompile(`(?m)^\s{2}workflow_dispatch:\s*$`).MatchString(body), "manual dispatch must remain canary-only")
	require(strings.Contains(body, "if: github.repository == 'rusliksu/beads'"), "exact fork repository guard is required")
	require(strings.Contains(body, "windows-latest"), "windows-latest matrix leg is required")
	require(strings.Contains(body, "ubuntu-latest"), "ubuntu-latest matrix leg is required")

	candidateExpression := "${{ github.event.pull_request.head.sha || github.sha }}"
	require(strings.Count(body, candidateExpression) >= 2, "checkout and evidence must resolve the exact PR head/candidate SHA")
	require(regexp.MustCompile(`(?m)^\s+ref:\s*\$\{\{ github\.event\.pull_request\.head\.sha \|\| github\.sha \}\}\s*$`).MatchString(body), "checkout ref must be the exact PR head/candidate SHA")
	require(regexp.MustCompile(`actions/checkout@[0-9a-f]{40}(?:\s|$)`).MatchString(body), "actions/checkout must be pinned to a full commit SHA")
	require(regexp.MustCompile(`actions/setup-go@[0-9a-f]{40}(?:\s|$)`).MatchString(body), "actions/setup-go must be pinned to a full commit SHA")
	require(regexp.MustCompile(`actions/upload-artifact@[0-9a-f]{40}(?:\s|$)`).MatchString(body), "actions/upload-artifact must be pinned to a full commit SHA")
	require(regexp.MustCompile(`(?m)^\s+go-version-file:\s*['"]?go\.mod['"]?\s*$`).MatchString(body), "setup-go must read go.mod")

	timeoutMatch := regexp.MustCompile(`(?m)^\s+timeout-minutes:\s*([0-9]+)\s*$`).FindStringSubmatch(body)
	require(timeoutMatch != nil, "per-job timeout-minutes is required")
	if timeoutMatch != nil {
		minutes, err := strconv.Atoi(timeoutMatch[1])
		require(err == nil && minutes <= 15, "per-job timeout must be 15 minutes or tighter")
	}

	require(strings.Contains(body, "BD_DISABLE_METRICS: \"1\""), "metrics must be disabled")
	require(strings.Contains(body, "BD_DISABLE_EVENT_FLUSH: \"1\""), "event flush must be disabled")
	require(strings.Contains(body, "gms_pure_go"), "build must use the gms_pure_go tag")
	require(strings.Contains(body, "RUNNER_TEMP"), "binary and evidence must use the runner-temporary directory")
	require(regexp.MustCompile(`go build[^\n]*-tags[^\n]*gms_pure_go[^\n]*-o[^\n]*\./cmd/bd`).MatchString(body), "cmd/bd must build with gms_pure_go into an explicit output path")
	require(strings.Contains(body, "& $binary version"), "version smoke must execute the just-built binary by explicit path")
	require(strings.Contains(body, "& $binary help"), "help smoke must execute the just-built binary by explicit path")
	require(strings.Contains(body, "git rev-parse HEAD"), "checked-out HEAD must be verified against the candidate SHA")

	for _, field := range []string{"candidate_sha", "platform", "runner", "go_version", "build_tags", "build_result", "smoke_result", "artifact_sha256", "workflow_run_id"} {
		require(strings.Contains(body, field), fmt.Sprintf("evidence manifest must contain %s", field))
	}
	require(strings.Contains(body, "if-no-files-found: error"), "artifact upload must fail when evidence is missing")
	retentionMatch := regexp.MustCompile(`(?m)^\s+retention-days:\s*([0-9]+)\s*$`).FindStringSubmatch(body)
	require(retentionMatch != nil, "artifact retention must be explicit")
	if retentionMatch != nil {
		days, err := strconv.Atoi(retentionMatch[1])
		require(err == nil && days <= 7, "artifact retention must be seven days or shorter")
	}
	require(regexp.MustCompile(`(?m)^\s+name:\s*fork-canary-\$\{\{ matrix\.platform \}\}-\$\{\{ env\.CANDIDATE_SHA \}\}\s*$`).MatchString(body), "artifact name must include platform and candidate SHA")

	for _, forbidden := range []struct {
		pattern string
		label   string
	}{
		{`(?im)^\s*(?:run|uses):[^\n]*(?:gh release|goreleaser|publish|go install|package manager|git tag|git push|setx?\s+PATH|production sign)`, "release, publish, install, tag, push, PATH replacement, and production signing steps are forbidden"},
		{`(?m)^\s+secrets:`, "production or repository secrets are forbidden"},
	} {
		require(!regexp.MustCompile(forbidden.pattern).MatchString(body), forbidden.label)
	}

	return problems
}

func containsProblem(problems []string, needle string) bool {
	for _, problem := range problems {
		if strings.Contains(problem, needle) {
			return true
		}
	}
	return false
}

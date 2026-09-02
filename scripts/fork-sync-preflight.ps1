[CmdletBinding()]
param(
    [Parameter()]
    [string]$ForkRef = 'refs/remotes/origin/main',

    [Parameter()]
    [string]$UpstreamRef = 'refs/remotes/upstream/main',

    [Parameter(Mandatory)]
    [string]$OutputPath,

    [Parameter()]
    [ValidateSet('unchanged', 'unknown', 'divergent')]
    [string]$BehaviorEquivalence = 'unknown',

    [Parameter()]
    [switch]$NoFetch
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$GitCommandTimeoutMilliseconds = 15000

function Invoke-GitCapture {
    param(
        [Parameter(Mandatory)]
        [string[]]$Arguments,

        [Parameter()]
        [int[]]$AllowedExitCodes = @(0)
    )

    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = 'git'
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    foreach ($argument in $Arguments) {
        $startInfo.ArgumentList.Add($argument)
    }

    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    try {
        $started = $process.Start()
    }
    catch {
        $process.Dispose()
        throw 'git is unavailable'
    }
    if (-not $started) {
        $process.Dispose()
        throw 'git is unavailable'
    }

    $standardOutput = $process.StandardOutput.ReadToEndAsync()
    $standardError = $process.StandardError.ReadToEndAsync()
    if (-not $process.WaitForExit($GitCommandTimeoutMilliseconds)) {
        try {
            $process.Kill($true)
        }
        catch {
            # Preserve the deterministic timeout classification even when the
            # platform reports that the process exited during cancellation.
        }
        $process.WaitForExit()
        $null = $standardOutput.GetAwaiter().GetResult()
        $null = $standardError.GetAwaiter().GetResult()
        $process.Dispose()
        throw 'git command timed out'
    }

    $stdout = $standardOutput.GetAwaiter().GetResult()
    $null = $standardError.GetAwaiter().GetResult()
    $exitCode = $process.ExitCode
    $process.Dispose()

    if ($AllowedExitCodes -notcontains $exitCode) {
        throw 'git command failed'
    }

    [pscustomobject]@{
        ExitCode = $exitCode
        StdOut  = $stdout
    }
}

function Assert-SafeRevisionInput {
    param(
        [Parameter(Mandatory)]
        [string]$Value,

        [Parameter(Mandatory)]
        [string]$Label
    )

    $isSafe =
        $Value.Length -ge 1 -and
        $Value.Length -le 256 -and
        $Value -cmatch '^[0-9A-Za-z][0-9A-Za-z._/-]*$' -and
        -not $Value.Contains('..') -and
        -not $Value.Contains('//') -and
        -not $Value.EndsWith('/') -and
        -not $Value.EndsWith('.') -and
        -not $Value.EndsWith('.lock', [System.StringComparison]::OrdinalIgnoreCase)

    if (-not $isSafe) {
        throw "$Label ref is invalid"
    }
}

function Resolve-Commit {
    param(
        [Parameter(Mandatory)]
        [string]$Revision,

        [Parameter(Mandatory)]
        [string]$Label
    )

    try {
        $result = Invoke-GitCapture -Arguments @(
            'rev-parse',
            '--verify',
            '--quiet',
            '--end-of-options',
            "$Revision^{commit}"
        )
    }
    catch {
        if ($_.Exception.Message -ceq 'git command timed out') {
            throw "$Label ref resolution timed out"
        }
        throw "$Label ref does not resolve to a commit"
    }

    $commit = $result.StdOut.Trim()
    if ($commit -cnotmatch '^[0-9a-f]{40}$') {
        throw "$Label ref does not resolve to a commit"
    }
    $commit
}

function Get-NonNegativeCount {
    param(
        [Parameter(Mandatory)]
        [string]$Range,

        [Parameter(Mandatory)]
        [string]$Label
    )

    try {
        $result = Invoke-GitCapture -Arguments @('rev-list', '--count', $Range)
        $count = 0
        if (-not [int]::TryParse(
                $result.StdOut.Trim(),
                [System.Globalization.NumberStyles]::None,
                [System.Globalization.CultureInfo]::InvariantCulture,
                [ref]$count
            ) -or $count -lt 0) {
            throw 'invalid count'
        }
        $count
    }
    catch {
        if ($_.Exception.Message -ceq 'git command timed out') {
            throw "$Label commit count timed out"
        }
        throw "cannot compute $Label commit count"
    }
}

function Get-SortedUniqueStrings {
    param(
        [Parameter()]
        [AllowEmptyCollection()]
        [string[]]$Values = @()
    )

    $set = [System.Collections.Generic.SortedSet[string]]::new(
        [System.StringComparer]::Ordinal
    )
    foreach ($value in $Values) {
        if (-not [string]::IsNullOrEmpty($value)) {
            $null = $set.Add($value)
        }
    }
    [string[]]$set
}

function Get-ConflictForecast {
    param(
        [Parameter(Mandatory)]
        [string]$ForkCommit,

        [Parameter(Mandatory)]
        [string]$UpstreamCommit,

        [Parameter(Mandatory)]
        [string]$MergeBase
    )

    try {
        $result = Invoke-GitCapture -Arguments @(
            'merge-tree',
            '--write-tree',
            '--name-only',
            '--no-messages',
            '-z',
            '--merge-base',
            $MergeBase,
            $ForkCommit,
            $UpstreamCommit
        ) -AllowedExitCodes @(0, 1)
    }
    catch {
        if ($_.Exception.Message -ceq 'git command timed out') {
            throw 'merge forecast timed out'
        }
        throw 'merge forecast failed'
    }

    $segments = @($result.StdOut.Split(
            [char]0,
            [System.StringSplitOptions]::RemoveEmptyEntries
        ))
    if ($segments.Count -lt 1 -or $segments[0] -cnotmatch '^[0-9a-f]{40}$') {
        throw 'merge forecast produced invalid evidence'
    }

    if ($result.ExitCode -eq 0) {
        if ($segments.Count -ne 1) {
            throw 'merge forecast produced ambiguous evidence'
        }
        return [string[]]@()
    }

    if ($segments.Count -lt 2) {
        throw 'merge forecast did not identify conflict paths'
    }
    Get-SortedUniqueStrings -Values $segments[1..($segments.Count - 1)]
}

function Write-ManifestAtomically {
    param(
        [Parameter(Mandatory)]
        [string]$Path,

        [Parameter(Mandatory)]
        [string]$Json
    )

    try {
        $fullPath = [System.IO.Path]::GetFullPath($Path)
        $directory = [System.IO.Path]::GetDirectoryName($fullPath)
        if ([string]::IsNullOrEmpty($directory) -or -not [System.IO.Directory]::Exists($directory)) {
            throw 'missing output directory'
        }

        $temporaryName = '.{0}.{1}.tmp' -f @(
            [System.IO.Path]::GetFileName($fullPath),
            [guid]::NewGuid().ToString('N')
        )
        $temporaryPath = [System.IO.Path]::Combine($directory, $temporaryName)
        $encoding = [System.Text.UTF8Encoding]::new($false)
        [System.IO.File]::WriteAllText($temporaryPath, $Json, $encoding)
        try {
            [System.IO.File]::Move($temporaryPath, $fullPath, $true)
        }
        finally {
            if ([System.IO.File]::Exists($temporaryPath)) {
                [System.IO.File]::Delete($temporaryPath)
            }
        }
    }
    catch {
        throw 'cannot write output manifest'
    }
}

try {
    Assert-SafeRevisionInput -Value $ForkRef -Label 'fork'
    Assert-SafeRevisionInput -Value $UpstreamRef -Label 'upstream'

    try {
        $insideWorkTree = Invoke-GitCapture -Arguments @('rev-parse', '--is-inside-work-tree')
    }
    catch {
        if ($_.Exception.Message -ceq 'git command timed out') {
            throw 'repository verification timed out'
        }
        throw 'current directory is not a Git worktree'
    }
    if ($insideWorkTree.StdOut.Trim() -cne 'true') {
        throw 'current directory is not a Git worktree'
    }

    # There is intentionally no fetch path. -NoFetch is an explicit operator
    # marker; omitting it still cannot trigger network or ref mutation.
    $null = $NoFetch

    $forkCommit = Resolve-Commit -Revision $ForkRef -Label 'fork'
    $upstreamCommit = Resolve-Commit -Revision $UpstreamRef -Label 'upstream'

    try {
        $mergeBaseResult = Invoke-GitCapture -Arguments @('merge-base', $forkCommit, $upstreamCommit)
    }
    catch {
        if ($_.Exception.Message -ceq 'git command timed out') {
            throw 'merge-base calculation timed out'
        }
        throw 'fork and upstream commits do not share a merge base'
    }
    $mergeBase = $mergeBaseResult.StdOut.Trim()
    if ($mergeBase -cnotmatch '^[0-9a-f]{40}$') {
        throw 'fork and upstream commits do not share a merge base'
    }

    $forkOnlyCommits = Get-NonNegativeCount -Range "$upstreamCommit..$forkCommit" -Label 'fork-only'
    $upstreamOnlyCommits = Get-NonNegativeCount -Range "$forkCommit..$upstreamCommit" -Label 'upstream-only'
    $conflicts = @(Get-ConflictForecast `
            -ForkCommit $forkCommit `
            -UpstreamCommit $upstreamCommit `
            -MergeBase $mergeBase)

    $blockers = [System.Collections.Generic.List[string]]::new()
    $candidateStatus = 'blocked'
    if ($forkOnlyCommits -eq 0 -and $upstreamOnlyCommits -eq 0) {
        $candidateStatus = 'already_identical'
    }
    else {
        if ($conflicts.Count -gt 0) {
            $blockers.Add('merge_conflicts')
        }
        if ($forkOnlyCommits -gt 0) {
            $blockers.Add('fork_only_commits')
        }
        if ($upstreamOnlyCommits -eq 0) {
            $blockers.Add('no_upstream_commits')
        }
        if ($BehaviorEquivalence -eq 'unknown') {
            $blockers.Add('behavior_equivalence_unknown')
        }
        elseif ($BehaviorEquivalence -eq 'divergent') {
            $blockers.Add('behavior_equivalence_divergent')
        }

        if (
            $upstreamOnlyCommits -gt 0 -and
            $forkOnlyCommits -eq 0 -and
            $conflicts.Count -eq 0 -and
            $BehaviorEquivalence -ceq 'unchanged'
        ) {
            $candidateStatus = 'ready_for_pr'
            $blockers.Clear()
        }
    }
    $sortedBlockers = @(Get-SortedUniqueStrings -Values $blockers.ToArray())

    if ($candidateStatus -ceq 'blocked' -and $sortedBlockers.Count -eq 0) {
        throw 'candidate classification is ambiguous'
    }

    $manifest = [ordered]@{
        schema               = 'fork-sync-manifest/v1'
        generated_at         = [datetime]::UtcNow.ToString(
            'o',
            [System.Globalization.CultureInfo]::InvariantCulture
        )
        fork_repository      = 'rusliksu/beads'
        fork_base_sha        = $forkCommit
        upstream_repository  = 'gastownhall/beads'
        upstream_head_sha    = $upstreamCommit
        merge_base_sha       = $mergeBase
        fork_only_commits    = $forkOnlyCommits
        upstream_only_commits = $upstreamOnlyCommits
        conflicts            = [string[]]$conflicts
        behavior_equivalence = $BehaviorEquivalence
        candidate_status     = $candidateStatus
        blockers             = [string[]]$sortedBlockers
    }

    $json = $manifest | ConvertTo-Json -Depth 4
    Write-ManifestAtomically -Path $OutputPath -Json ($json + [Environment]::NewLine)
    Write-Output (
        'fork-sync-manifest/v1 status={0} fork={1} upstream={2}' -f
        $candidateStatus,
        $forkCommit,
        $upstreamCommit
    )
}
catch {
    [Console]::Error.WriteLine("fork-sync-preflight: {0}" -f $_.Exception.Message)
    exit 2
}

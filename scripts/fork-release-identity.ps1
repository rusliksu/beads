[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$CandidateVersion,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$UpstreamVersion,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$ForkSha,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$UpstreamSha,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$OutputPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$stableVersionPattern = '(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)'
$candidatePattern = "^(?<base>$stableVersionPattern)-ruslan\.(?<sequence>[1-9][0-9]*)\+upstream\.(?<upstreamPrefix>[0-9a-f]{7,12})$"
$candidateMatch = [regex]::Match($CandidateVersion, $candidatePattern)

if (-not $candidateMatch.Success) {
    throw 'CandidateVersion must match <upstream-version>-ruslan.<positive-sequence>+upstream.<7-12 lowercase SHA characters>.'
}

if ($UpstreamVersion -notmatch "^$stableVersionPattern$") {
    throw 'UpstreamVersion must be a stable semantic version with no prerelease, build metadata, or leading zeroes.'
}

if ($candidateMatch.Groups['base'].Value -cne $UpstreamVersion) {
    throw 'CandidateVersion base must exactly equal UpstreamVersion.'
}

$sequence = 0L
if (-not [long]::TryParse($candidateMatch.Groups['sequence'].Value, [ref]$sequence) -or $sequence -lt 1) {
    throw 'CandidateVersion sequence must be a positive 64-bit integer.'
}

$fullShaPattern = '^[0-9a-f]{40}$'
if ($ForkSha -cnotmatch $fullShaPattern) {
    throw 'ForkSha must be an exact 40-character lowercase hexadecimal commit SHA, not a mutable ref.'
}
if ($UpstreamSha -cnotmatch $fullShaPattern) {
    throw 'UpstreamSha must be an exact 40-character lowercase hexadecimal commit SHA, not a mutable ref.'
}

$upstreamPrefix = $candidateMatch.Groups['upstreamPrefix'].Value
if (-not $UpstreamSha.StartsWith($upstreamPrefix, [System.StringComparison]::Ordinal)) {
    throw 'CandidateVersion upstream metadata must prefix UpstreamSha.'
}

$identity = [ordered]@{
    schema            = 'fork-release-identity/v1'
    candidate_version = $CandidateVersion
    upstream_version  = $UpstreamVersion
    sequence          = $sequence
    fork_sha          = $ForkSha
    upstream_sha      = $UpstreamSha
    published         = $false
    installed         = $false
}

$fullOutputPath = [System.IO.Path]::GetFullPath($OutputPath)
$outputDirectory = [System.IO.Path]::GetDirectoryName($fullOutputPath)
if ([string]::IsNullOrWhiteSpace($outputDirectory) -or -not [System.IO.Directory]::Exists($outputDirectory)) {
    throw 'OutputPath parent directory must already exist.'
}

$outputName = [System.IO.Path]::GetFileName($fullOutputPath)
$temporaryPath = [System.IO.Path]::Combine(
    $outputDirectory,
    ".${outputName}.$([System.Guid]::NewGuid().ToString('N')).tmp"
)

try {
    $json = $identity | ConvertTo-Json -Depth 3
    $utf8WithoutBom = [System.Text.UTF8Encoding]::new($false)
    [System.IO.File]::WriteAllText($temporaryPath, "$json$([System.Environment]::NewLine)", $utf8WithoutBom)
    [System.IO.File]::Move($temporaryPath, $fullOutputPath)
}
finally {
    if ([System.IO.File]::Exists($temporaryPath)) {
        [System.IO.File]::Delete($temporaryPath)
    }
}

Write-Output $fullOutputPath

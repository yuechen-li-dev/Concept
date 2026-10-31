param(
    [ValidateSet('Quick', 'Runtime', 'Compiler', 'Qualification')]
    [string]$Lane = 'Quick',
    [string]$Filter,
    [string]$Path = 'tests/runtime',
    [switch]$Verify,
    [string]$ProfileDirectory
)

$ErrorActionPreference = 'Stop'
if ($ProfileDirectory -and $Lane -ne 'Qualification') {
    throw '-ProfileDirectory profiles the complete Qualification lane.'
}
if ($Lane -ne 'Runtime' -and ($Filter -or $Verify)) {
    throw '-Filter and -Verify select Concept facts in the Runtime lane.'
}
$timer = [Diagnostics.Stopwatch]::StartNew()
$priorQualificationProfile = $env:CONCEPT_QUALIFICATION_PROFILE
$priorEVT2hPhaseQualification = $env:CONCEPT_EVT2H_PHASE_QUALIFICATION
Push-Location -LiteralPath $PSScriptRoot
try {
    switch ($Lane) {
        'Quick' {
            # Concept owns behavior expectations. Go only dispatches the real
            # framework and checks that this lane cannot bootstrap Cathedral.
            & go test ./internal/concept -short -run '^(TestRuntimeConceptSpecifications|TestBackendConceptSpecifications|TestRuntimeWorkflowPolicy)$' -count=1 -timeout=2m
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
            & go test ./cmd/concept -short -count=1 -timeout=2m
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        }
        'Compiler' {
            # The broader Go compiler/corpus/determinism sweep is explicit.
            & go test -short ./... -count=1 -parallel=4 -timeout=10m
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        }
        'Runtime' {
            $conceptArgs = @('run', './cmd/concept', 'test', $Path, '--module-root', 'libraries', '--verbose')
            if ($Filter) { $conceptArgs += @('--filter', $Filter) }
            if ($Verify) { $conceptArgs += '--verify' }
            & go @conceptArgs
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        }
        'Qualification' {
            # New ordinary milestones sample by default; the phase checkpoint
            # schedules their retained 100 fresh generation/emission trials.
            $env:CONCEPT_EVT2H_PHASE_QUALIFICATION = '1'
            # Includes the fresh CMIRAMD9 CFG/native 100-run gates. Keep the
            # full lane bounded without truncating those qualification runs.
            $goTimer = [Diagnostics.Stopwatch]::StartNew()
            if ($ProfileDirectory) {
                $profileRoot = [IO.Path]::GetFullPath($ProfileDirectory)
                [IO.Directory]::CreateDirectory($profileRoot) | Out-Null
                $env:CONCEPT_QUALIFICATION_PROFILE = Join-Path $profileRoot 'commands.jsonl'
                [IO.File]::WriteAllText($env:CONCEPT_QUALIFICATION_PROFILE, '')
                Write-Host "Collecting uncached qualification evidence in $profileRoot"
                & go test ./... -json -count=1 -parallel=4 -timeout=45m > (Join-Path $profileRoot 'tests.jsonl')
            } else {
                & go test ./... -count=1 -parallel=4 -timeout=45m
            }
            $goExit = $LASTEXITCODE
            $goTimer.Stop()
            if ($ProfileDirectory) {
                @{ go_seconds = $goTimer.Elapsed.TotalSeconds; go_exit = $goExit } | ConvertTo-Json | Set-Content (Join-Path $profileRoot 'timing.json')
            }
            if ($goExit -ne 0) { exit $goExit }
            $vetTimer = [Diagnostics.Stopwatch]::StartNew()
            & go vet ./...
            $vetExit = $LASTEXITCODE
            $vetTimer.Stop()
            if ($ProfileDirectory) {
                @{ go_seconds = $goTimer.Elapsed.TotalSeconds; go_exit = $goExit; vet_seconds = $vetTimer.Elapsed.TotalSeconds; vet_exit = $vetExit; wall_seconds = $timer.Elapsed.TotalSeconds } | ConvertTo-Json | Set-Content (Join-Path $profileRoot 'timing.json')
                & (Join-Path $PSScriptRoot 'tools/qualification/report.ps1') -Directory $profileRoot
            }
            if ($vetExit -ne 0) { exit $vetExit }
        }
    }
    $timer.Stop()
    Write-Host "$Lane passed in $([math]::Round($timer.Elapsed.TotalSeconds, 2)) seconds."
} finally {
    $env:CONCEPT_EVT2H_PHASE_QUALIFICATION = $priorEVT2hPhaseQualification
    $env:CONCEPT_QUALIFICATION_PROFILE = $priorQualificationProfile
    Pop-Location
}

param(
    [ValidateSet('Quick', 'Runtime', 'Compiler', 'Qualification')]
    [string]$Lane = 'Quick',
    [string]$Filter,
    [string]$Path = 'tests/runtime',
    [switch]$Verify
)

$ErrorActionPreference = 'Stop'
if ($Lane -ne 'Runtime' -and ($Filter -or $Verify)) {
    throw '-Filter and -Verify select Concept facts in the Runtime lane.'
}
$timer = [Diagnostics.Stopwatch]::StartNew()
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
            & go test ./... -count=1 -parallel=4 -timeout=35m
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
            & go vet ./...
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        }
    }
    $timer.Stop()
    Write-Host "$Lane passed in $([math]::Round($timer.Elapsed.TotalSeconds, 2)) seconds."
} finally {
    Pop-Location
}

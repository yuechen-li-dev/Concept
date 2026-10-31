param([Parameter(Mandatory)][string]$Directory)
$ErrorActionPreference = 'Stop'
$tests = foreach ($line in [IO.File]::ReadLines((Join-Path $Directory 'tests.jsonl'))) { $line | ConvertFrom-Json }
$events = foreach ($line in [IO.File]::ReadLines((Join-Path $Directory 'commands.jsonl'))) { $line | ConvertFrom-Json }
$completed = @($tests | Where-Object { $_.Action -in @('pass', 'fail', 'skip') -and $_.Test -and $_.Test -notmatch '/' })
$commands = @($events | Where-Object { $_.kind -eq 'command-cleanup' -and $_.fields.started })
$objects = @($events | Where-Object kind -eq 'native-object')
$summary = [ordered]@{
    timing = Get-Content (Join-Path $Directory 'timing.json') -Raw | ConvertFrom-Json
    top_level_tests = $completed.Count
    passed = @($completed | Where-Object Action -eq 'pass').Count
    failed = @($completed | Where-Object Action -eq 'fail').Count
    skipped = @($completed | Where-Object Action -eq 'skip').Count
    packages = @($tests | Where-Object { $_.Action -in @('pass','fail') -and -not $_.Test } | Select-Object Package, Action, Elapsed)
    largest_families = @($completed | Sort-Object Elapsed -Descending | Select-Object -First 30 Test, Action, Elapsed)
    cathedral_generation = @($events | Where-Object kind -eq 'cathedral-generation')
    cold_native_objects = @($objects | Where-Object { $_.fields.cold }).Count
    reused_native_objects = @($objects | Where-Object { -not $_.fields.cold }).Count
    native_command_launches = $commands.Count
    native_command_cpu_seconds = ($commands | ForEach-Object { $_.fields.user_seconds + $_.fields.system_seconds } | Measure-Object -Sum).Sum
    cold_objects = @($objects | Where-Object { $_.fields.cold } | Select-Object test, seconds, fields)
    scope = 'Native command counts and CPU cover the Go nativeCommand boundary, not descendant processes or all compiler invocations. Cleanup elapsed is not process wall time. Go JSON records test-family wall time; native-object and cathedral-generation record actual stage wall time. No qualification result or fresh artifact is cached.'
}
$summary | ConvertTo-Json -Depth 12 | Set-Content (Join-Path $Directory 'summary.json')
Write-Host "Qualification: $($summary.passed) passed families, $($summary.failed) failed; $($summary.cold_native_objects) cold / $($summary.reused_native_objects) reused C objects."

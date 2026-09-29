# VK8 device run. From the repository root:
#   powershell -ExecutionPolicy Bypass -File tools\vk8\run_vk8.ps1
# Everything is logged to tools\vk8\out\vk8.log (read back by Claude); the
# last lines summarize each step as PASS/FAIL.
$ErrorActionPreference = "Continue"
$root = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$out = Join-Path $PSScriptRoot "out"
New-Item -ItemType Directory -Force -Path $out | Out-Null
$log = Join-Path $out "vk8.log"
"VK8 device run $(Get-Date -Format o)" | Out-File -Encoding utf8 $log
$summary = @()

function Log([string]$text) { $text | Out-File -Encoding utf8 -Append $log; Write-Host $text }
function Step([string]$name, [scriptblock]$body) {
    Log ""
    Log "==== $name"
    $global:LASTEXITCODE = 0
    try {
        $output = & $body 2>&1 | Out-String
        Log $output
        $ok = ($LASTEXITCODE -eq 0)
    } catch {
        Log ($_ | Out-String)
        $ok = $false
    }
    $status = if ($ok) { "PASS" } else { "FAIL" }
    Log "---- $name : $status (exit $LASTEXITCODE)"
    $script:summary += "$status  $name"
}

# Toolchain discovery.
if (-not $env:VULKAN_SDK) {
    $sdk = Get-ChildItem "C:\VulkanSDK" -Directory -ErrorAction SilentlyContinue | Sort-Object Name -Descending | Select-Object -First 1
    if ($sdk) { $env:VULKAN_SDK = $sdk.FullName }
}
$mingw = Join-Path $HOME "mingw64\bin"
if (Test-Path $mingw) { $env:PATH = "$mingw;$env:PATH" }
$goRoot = "C:\Program Files\Go\bin"
if (-not (Get-Command go -ErrorAction SilentlyContinue) -and (Test-Path $goRoot)) { $env:PATH = "$goRoot;$env:PATH" }
if ($env:VULKAN_SDK) { $env:PATH = "$(Join-Path $env:VULKAN_SDK 'Bin');$env:PATH" }
Log "VULKAN_SDK=$env:VULKAN_SDK"
Log "root=$root"

Set-Location $root
Step "toolchain" {
    go version
    gcc --version | Select-Object -First 1
    g++ --version | Select-Object -First 1
    glslc --version | Select-Object -First 1
}
Step "vulkaninfo" { vulkaninfo --summary }
Step "compile double.comp" {
    glslc examples\vulkan\kernels\double.comp -o examples\vulkan\kernels\double.spv
}
$inc = Join-Path $env:VULKAN_SDK "Include"
$lib = Join-Path $env:VULKAN_SDK "Lib"
Step "reference C program" {
    gcc -std=c11 -Wall -Wextra -I"$inc" examples\vulkan\reference\compute_dispatch.c -L"$lib" -lvulkan-1 -o "$out\compute_dispatch_c.exe"
    if ($LASTEXITCODE -eq 0) { & "$out\compute_dispatch_c.exe" examples\vulkan\kernels\double.spv }
}
Step "build concept" { go build -o "$out\concept.exe" .\cmd\concept }
Step "Vulkan library tests (test device)" { & "$out\concept.exe" test libraries\Vulkan --verify }
Step "examples on the test device" { & "$out\concept.exe" test examples\vulkan --verify }
$env:CONCEPT_VULKAN_RUNTIME = "device"
Step "examples on the GPU (Normal)" { & "$out\concept.exe" test examples\vulkan --verbose }
Step "examples on the GPU (Verify)" { & "$out\concept.exe" test examples\vulkan --verify --verbose }
Remove-Item Env:\CONCEPT_VULKAN_RUNTIME

Log ""
Log "==== summary"
$summary | ForEach-Object { Log $_ }

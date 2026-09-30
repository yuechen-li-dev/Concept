# Vulkan library GPU verification. From the repository root:
#   powershell -ExecutionPolicy Bypass -File tools\vulkan\run_gpu.ps1
# Logs to tools\vulkan\out\gpu.log; the last lines summarize each step.
$ErrorActionPreference = "Continue"
$root = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$out = Join-Path $PSScriptRoot "out"
New-Item -ItemType Directory -Force -Path $out | Out-Null
$log = Join-Path $out "gpu.log"
"Vulkan GPU run $(Get-Date -Format o)" | Out-File -Encoding utf8 $log
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
$k = "examples\vulkan\kernels"
$concept = Join-Path $out "concept.exe"

Step "toolchain" {
    go version
    (& gcc --version)[0]
    (& glslc --version)[0]
    (& dxc --version)[0]
}
Step "vulkaninfo" { vulkaninfo --summary }
Step "compile kernels (glslc, dxc)" {
    glslc --target-env=vulkan1.3 "$k\double.comp" -o "$k\double.spv"
    if ($LASTEXITCODE -eq 0) { glslc --target-env=vulkan1.3 "$k\scale.comp" -o "$k\scale.spv" }
    if ($LASTEXITCODE -eq 0) { dxc -spirv -fspv-target-env=vulkan1.3 -T cs_6_0 -E main "$k\scale.hlsl" -Fo "$k\scale_hlsl.spv" }
}
Step "build concept" { go build -o $concept .\cmd\concept }
Step "kernel bindings are current" {
    & $concept vulkan-bind "$k\double.spv" -o examples\vulkan\DoubleKernel.concept --check
    if ($LASTEXITCODE -eq 0) { & $concept vulkan-bind "$k\scale.spv" -o examples\vulkan\ScaleKernel.concept --check }
}
Step "HLSL and GLSL reflect the same interface" {
    $glsl = & $concept vulkan-bind "$k\scale.spv" --describe
    $hlsl = & $concept vulkan-bind "$k\scale_hlsl.spv" --describe
    "GLSL:"; $glsl; "HLSL (DXC):"; $hlsl
    $a = ($glsl | Select-String "^fingerprint").Line
    $b = ($hlsl | Select-String "^fingerprint").Line
    if ($a -ne $b) { "fingerprints differ"; $global:LASTEXITCODE = 1 }
}
$inc = Join-Path $env:VULKAN_SDK "Include"
$lib = Join-Path $env:VULKAN_SDK "Lib"
Step "reference C program" {
    gcc -std=c11 -Wall -Wextra -I"$inc" examples\vulkan\reference\compute_dispatch.c -L"$lib" -lvulkan-1 -o "$out\compute_dispatch_c.exe"
    if ($LASTEXITCODE -eq 0) { & "$out\compute_dispatch_c.exe" "$k\double.spv" }
}
Step "library tests (test device)" { & $concept test libraries\Vulkan --verify }
Step "examples (test device)" { & $concept test examples\vulkan --verify }

$env:CONCEPT_VULKAN_RUNTIME = "device"
Step "library tests (GPU, default device)" { & $concept test libraries\Vulkan --verbose }
Step "examples (GPU, default device, Normal)" { & $concept test examples\vulkan --verbose }
Step "examples (GPU, default device, Verify)" { & $concept test examples\vulkan --verify --verbose }
$env:CONCEPT_VULKAN_DEVICE = "AMD"
Step "examples (GPU, CONCEPT_VULKAN_DEVICE=AMD)" { & $concept test examples\vulkan --verbose }
Remove-Item Env:\CONCEPT_VULKAN_DEVICE
# Khronos validation with synchronization validation: the derived barriers
# are checked by the real validator. Any VUID or SYNC-HAZARD message fails.
$env:CONCEPT_VULKAN_VALIDATION = "1"
$env:VK_LAYER_ENABLES = "VK_VALIDATION_FEATURE_ENABLE_SYNCHRONIZATION_VALIDATION_EXT"
Step "examples + library under validation and sync validation" {
    $text = (& $concept test examples\vulkan --verbose 2>&1 | Out-String) + (& $concept test libraries\Vulkan --verbose 2>&1 | Out-String)
    $text
    if ($text -match "VUID-|SYNC-HAZARD|Validation Error") { "validation reported problems"; $global:LASTEXITCODE = 1 }
}
Remove-Item Env:\CONCEPT_VULKAN_VALIDATION
Remove-Item Env:\VK_LAYER_ENABLES
Remove-Item Env:\CONCEPT_VULKAN_RUNTIME

Log ""
Log "==== summary"
$summary | ForEach-Object { Log $_ }

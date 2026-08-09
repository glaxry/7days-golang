$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Push-Location (Join-Path $repoRoot "tools\docsite")
try {
    & go run . -root $repoRoot
    if ($LASTEXITCODE -ne 0) {
        throw "HTML documentation build failed"
    }
}
finally {
    Pop-Location
}

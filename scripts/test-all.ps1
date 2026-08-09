$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$modules = Get-ChildItem -LiteralPath $repoRoot -Recurse -Filter go.mod -File |
    Sort-Object FullName

foreach ($module in $modules) {
    $relative = $module.Directory.FullName.Substring($repoRoot.Length + 1)
    Write-Host "==> go test $relative"
    Push-Location $module.Directory.FullName
    try {
        & go test -timeout 2m ./...
        if ($LASTEXITCODE -ne 0) {
            throw "Tests failed in $relative"
        }
    }
    finally {
        Pop-Location
    }
}

Write-Host "==> verify generated HTML documentation"
Push-Location (Join-Path $repoRoot "tools\docsite")
try {
    & go run . -root $repoRoot -check
    if ($LASTEXITCODE -ne 0) {
        throw "HTML documentation is out of date"
    }
}
finally {
    Pop-Location
}

$wasmOutput = Join-Path ([System.IO.Path]::GetTempPath()) ("7days-golang-wasm-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $wasmOutput | Out-Null
try {
    foreach ($demo in Get-ChildItem -LiteralPath (Join-Path $repoRoot "demo-wasm") -Directory) {
        Write-Host "==> GOOS=js GOARCH=wasm go build $($demo.Name)"
        $env:GOOS = "js"
        $env:GOARCH = "wasm"
        & go build -o (Join-Path $wasmOutput ($demo.Name + ".wasm")) (Join-Path $demo.FullName "main.go")
        if ($LASTEXITCODE -ne 0) {
            throw "WebAssembly build failed in $($demo.FullName)"
        }
    }
}
finally {
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    if (Test-Path -LiteralPath $wasmOutput) {
        Remove-Item -LiteralPath $wasmOutput -Recurse -Force
    }
}

Write-Host "All $($modules.Count) modules and WebAssembly demos passed."

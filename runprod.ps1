$ErrorActionPreference = "Stop"

$projectDir = $PSScriptRoot
$executable = Join-Path $projectDir "photoseparator.exe"

Push-Location $projectDir
try {
    go build -o $executable .
    if ($LASTEXITCODE -ne 0) {
        throw "Go build failed with exit code $LASTEXITCODE."
    }

    @(
        "D:\Data\PicturesStage"
        "D:\Data\NetPictures"
        "D:\Data\NetVideo"
    ) | & $executable

    if ($LASTEXITCODE -ne 0) {
        throw "Photo Separator failed with exit code $LASTEXITCODE."
    }
}
finally {
    Pop-Location
}

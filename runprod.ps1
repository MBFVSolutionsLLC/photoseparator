$ErrorActionPreference = "Stop"

$projectDir = $PSScriptRoot
$executable = Join-Path $projectDir "photoseparator.exe"

Push-Location $projectDir
try {
    go build -o $executable .
    if ($LASTEXITCODE -ne 0) {
        throw "Go build failed with exit code $LASTEXITCODE."
    }

    & $executable -root "D:\Data\PicturesStage" -photo-out "D:\Data\NetPictures" -video-out "D:\Data\NetVideo"

    if ($LASTEXITCODE -ne 0) {
        throw "Photo Separator failed with exit code $LASTEXITCODE."
    }
}
finally {
    Pop-Location
}

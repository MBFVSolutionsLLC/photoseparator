# Photo Separator

Photo Separator is a Go command-line application that scans a root folder for photos
and videos, computes a perceptual thumbprint for each item, infers the best available
capture date, and organizes the files into dated photo and video output folders.

## What it does

- Prompts for:
  - the root directory to scan
  - the photo output directory
  - the video output directory
- Recursively finds common photo and video files.
- Skips the photo and video output directories if they are inside the root directory.
- Requires output directories to differ from the root directory.
- Computes a perceptual thumbprint so resized copies of the same picture usually
  produce the same value.
- For videos, extracts a frame near the middle using `ffmpeg` when available and
  thumbprints that frame.
- Infers dates from JPEG EXIF, filename patterns, parent folders, and file timestamps.
- Organizes files as:

```text
output/
  2024/
    07/
      13/
        20240713_153044.a1b2c3d4e5f60718.jpg
```

- Places duplicates into a `duplicates` folder under the matching day. When two
  files have the same date and thumbprint, the smaller file is treated as the
  duplicate.

## Supported formats

Native image decoding supports `.jpg`, `.jpeg`, `.png`, `.gif`, and
uncompressed 24-bit or 32-bit `.bmp`.

If `ffmpeg` is installed and available on `PATH`, videos are processed by
extracting a middle frame when `ffprobe` can read the duration; otherwise the
first frame is thumbprinted. `ffprobe` is also used, when available, to read
video creation timestamps. Supported video extensions include `.mp4`, `.mov`,
`.m4v`, `.avi`, `.mkv`, `.webm`, `.wmv`, `.3gp`, `.mts`, and `.m2ts`.

Files such as HEIC, TIFF, and WEBP are detected as photos. They can be
thumbprinted when either `ffmpeg` or ImageMagick's `magick` command can convert
them to a temporary JPEG frame.

## Run

```powershell
go run .
```

To build and run with the production directories configured in `runprod.ps1`:

```powershell
.\runprod.ps1
```

The script builds `photoseparator.exe` in the repository directory and runs it.
Like the application default, it moves processed files from the source directory.

Optional flags:

```powershell
go run . -workers 12 -copy
```

- `-workers`: number of concurrent file processors. Defaults to CPU count.
- `-copy`: copy files instead of moving them.
- `-dry-run`: print planned actions without moving or copying files.

If any files are skipped or fail during processing, the application reports the
failure and exits with a non-zero status.

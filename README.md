# giffer

Turn a **series of photos** (archive or folder) into an animated GIF.

**Release:** v1.1.0 — Windows, Linux, and macOS.

## Download

Prebuilt binaries live under [`release/`](https://github.com/alienfacepalm/giffer/tree/master/release) (one directory per platform) and on [GitHub Releases](https://github.com/alienfacepalm/giffer/releases) (includes `SHA256SUMS`):

| Platform | Path |
|----------|------|
| Windows (x64) | [`windows-amd64/giffer.exe`](release/windows-amd64/giffer.exe) |
| Windows (x86) | [`windows-386/giffer.exe`](release/windows-386/giffer.exe) |
| Linux (x64) | [`linux-amd64/giffer`](release/linux-amd64/giffer) |
| Linux (ARM64) | [`linux-arm64/giffer`](release/linux-arm64/giffer) |
| macOS (Intel) | [`darwin-amd64/giffer`](release/darwin-amd64/giffer) |
| macOS (Apple Silicon) | [`darwin-arm64/giffer`](release/darwin-arm64/giffer) |

Open your platform folder, download the binary, then:

```bash
# Linux / macOS
chmod +x giffer
./giffer
```

```powershell
# Windows
.\giffer.exe
```

Put photo archives or folders in an `upload/` directory next to the binary (or pass `--input`).

## Usage

Place photo archives and/or photo directories under `upload/`, then:

```bash
./giffer
```

On a terminal with no flags, that opens a short wizard (batch vs single, delay, width, loop). In scripts or when any flag is set without `--input`, it batch-converts `upload/` (skips sources that already have a matching `.gif`). Within each conversion, frame decode and encode also use a CPU worker pool.

**Supported archives:** `.zip`, `.tar`, `.tar.gz` / `.tgz`, `.tar.bz2` / `.tbz2`, `.tar.xz` / `.txz`, `.7z`

Single conversion:

```bash
./giffer --input upload/photos.zip
./giffer --input upload/photos.tar.gz
./giffer --input upload/vacation/
```

| Flag | Default | Meaning |
|------|---------|---------|
| `--input` | (wizard / batch `upload/`) | photo archive or directory |
| `--output` | beside input | destination `.gif` |
| `--delay-ms` | `100` | frame delay (GIF-safe default) |
| `--max-width` | `0` | max frame width; `0` = first photo width |
| `--loop` | `0` | `0` = loop forever |

## Local UI

```bash
./giffer ui
```

Opens a local page at `http://127.0.0.1:8765` — always that port, never a random one. If something is already listening there, `giffer ui` kills it and takes over. Override with `--addr`.

Works the same on Windows, Linux, and macOS: all UI assets (including fonts and Three.js) are embedded in the binary, so the page loads fully offline. Drop a photo archive (zip, tar.gz, 7z, and other supported formats). Uploaded archives and GIF output land in `upload/` next to the binary (override with `--upload-dir`). **Reset** clears the form, preview, and in-flight convert.

| Flag | Default | Meaning |
|------|---------|---------|
| `--addr` | `127.0.0.1:8765` | Listen address (fixed; never remapped) |
| `--upload-dir` | beside the binary | Uploaded archives and GIF output |

## Build

```bash
make build          # local binary → bin/giffer
make release        # all platforms → release/
```

Tagged releases (`v*`) are built and uploaded by GitHub Actions — see `.github/workflows/release.yml`.

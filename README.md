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

Double-click the binary to open the convert UI in a native window. From a terminal, `./giffer` opens the interactive wizard; `./giffer ui` always opens the UI window.

```bash
# Linux / macOS
chmod +x giffer
./giffer          # wizard on a TTY; double-click opens the UI window
./giffer ui       # always open the UI window
```

```powershell
# Windows
.\giffer.exe      # wizard in a terminal; double-click opens the UI window
.\giffer.exe ui   # always open the UI window
```

Put photo archives or folders in an `upload/` directory next to the binary (or pass `--input`).

## Usage

On a terminal with no flags, `./giffer` opens a short wizard (batch vs single, delay, width, loop). Double-clicking the release binary opens the UI window instead. In scripts or when any flag is set without `--input`, it batch-converts `upload/` (skips sources that already have a matching `.gif`).

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

Double-click the release binary, or run `./giffer ui`.

Opens the convert UI in a native window (embedded webview) — not an external browser tab. The UI listens on `http://127.0.0.1:8765` internally; always that port, never a random one. If something is already listening there, giffer kills it and takes over.

All UI assets (including fonts and Three.js) are embedded in the binary. Drop a photo archive (zip, tar.gz, 7z, and other supported formats). Uploaded archives and GIF output land in `upload/` next to the binary (override with `--upload-dir`). **Reset** clears the form, preview, and in-flight convert.

| Flag | Default | Meaning |
|------|---------|---------|
| `--addr` | `127.0.0.1:8765` | Listen address (fixed; never remapped) |
| `--upload-dir` | beside the binary | Uploaded archives and GIF output |

### Platform notes (embedded webview)

| OS | Requirement |
|----|-------------|
| Windows | WebView2 Runtime (included on Windows 10/11) |
| Linux | `libwebkit2gtk-4.1-0` (or 4.0) at runtime |
| macOS | System WebKit (no extra install) |

## Build

```bash
make build          # local binary → bin/giffer (desktop webview)
make release        # all platforms → release/
```

Release builds use the `desktop` tag. Windows uses go-webview2 (no CGO). Linux and macOS use webview_go (CGO; Linux build needs `libwebkit2gtk-4.1-dev`).

Tagged releases (`v*`) are built and uploaded by GitHub Actions — see `.github/workflows/release.yml`.

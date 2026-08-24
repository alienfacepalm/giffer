# giffer

Turn a **series of photos** (zip or folder) into an animated GIF.

**Latest release: [v1.1.1](https://github.com/alienfacepalm/giffer/releases/latest)** — Windows, Mac, and Linux. Download, double-click, convert.

## Quick start

### Windows
1. Download **[giffer.exe](https://github.com/alienfacepalm/giffer/releases/latest/download/giffer-windows-amd64.exe)**
2. **Double-click** it
3. Drop your photo zip in the window → **Convert**

### Mac
1. Download **[Giffer.app.zip](https://github.com/alienfacepalm/giffer/releases/latest/download/Giffer-darwin-arm64.app.zip)** ([Intel Mac](https://github.com/alienfacepalm/giffer/releases/latest/download/Giffer-darwin-amd64.app.zip))
2. Unzip → **double-click** `Giffer.app`
3. Drop your photo zip → **Convert**

> Blocked by Gatekeeper? Right-click `Giffer.app` → **Open** → **Open** again.

### Linux
1. Download **[giffer](https://github.com/alienfacepalm/giffer/releases/latest/download/giffer-linux-amd64)** ([ARM64](https://github.com/alienfacepalm/giffer/releases/latest/download/giffer-linux-arm64))
2. `chmod +x giffer` then `./giffer` (or double-click after allowing execute)
3. Drop your photo zip → **Convert**

> Window won't open? Run once: `sudo apt install libwebkit2gtk-4.1-0`

GIF output lands in an `upload/` folder next to the app. Supported inputs: `.zip`, `.tar`, `.tar.gz` / `.tgz`, `.tar.bz2` / `.tbz2` / `.tbz`, `.tar.xz` / `.txz`, `.7z`, `.rar`, and image folders.

---

## Download all platforms

Prebuilt apps live in [`release/`](release/) and on [GitHub Releases](https://github.com/alienfacepalm/giffer/releases/latest) (includes `SHA256SUMS`):

| Platform | File | How to run |
|----------|------|------------|
| Windows (64-bit) | [`giffer.exe`](release/windows-amd64/giffer.exe) | Double-click |
| Windows (32-bit) | [`giffer.exe`](release/windows-386/giffer.exe) | Double-click |
| Linux (64-bit) | [`giffer`](release/linux-amd64/giffer) | `chmod +x giffer` then double-click or `./giffer` |
| Linux (ARM64) | [`giffer`](release/linux-arm64/giffer) | same as above |
| macOS (Intel) | [`Giffer.app`](release/darwin-amd64/Giffer.app) | Double-click |
| macOS (Apple Silicon) | [`Giffer.app`](release/darwin-arm64/Giffer.app) | Double-click |

Each build is a **native app** with an embedded window — not a browser tab, no Go install needed.

---

## Usage (terminal)

Double-click opens the UI. From a terminal:

```powershell
# Windows
.\giffer.exe ui          # open UI
.\giffer.exe --input upload\photos.zip
```

```bash
# Mac / Linux
./giffer ui              # open UI
./giffer                 # interactive wizard (terminal only)
./giffer --input upload/photos.zip
```

With **no flags** in a terminal, `./giffer` opens the wizard. **Double-click** (or `./giffer ui`) opens the UI window. With flags but no `--input`, it batch-converts everything in `upload/`.

| Flag | Default | Meaning |
|------|---------|---------|
| `--input` | (wizard / batch) | photo archive or directory |
| `--output` | beside input | destination `.gif` |
| `--delay-ms` | `100` | milliseconds per frame |
| `--max-width` | `0` | max width; `0` = first photo width |
| `--loop` | `0` | `0` = loop forever |

**Supported archives:** `.zip`, `.tar`, `.tar.gz` / `.tgz`, `.tar.bz2` / `.tbz2` / `.tbz`, `.tar.xz` / `.txz`, `.7z`, `.rar`

---

## Local UI

The UI runs in a native window on `http://127.0.0.1:8765` internally. Drop a photo archive, set delay/width/loop, click **Convert**. **Reset** clears the form and preview.

| Flag | Default | Meaning |
|------|---------|---------|
| `--addr` | `127.0.0.1:8765` | listen address (loopback only unless `--allow-remote`) |
| `--upload-dir` | beside the binary | uploads and GIF output |
| `--allow-remote` | off | allow non-loopback `--addr` (exposes the convert API) |

| OS | One-time requirement |
|----|---------------------|
| Windows | WebView2 (built into Windows 10/11) |
| Linux | `libwebkit2gtk-4.1-0` |
| macOS | none (uses system WebKit) |

---

## Build

```bash
make build          # dev binary → bin/giffer
make build-gui      # release-style GUI binary (double-click test)
make release        # all platforms → release/
```

Tagged releases (`v*`) are built by GitHub Actions — see `.github/workflows/release.yml`.

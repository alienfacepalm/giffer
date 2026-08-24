# Giffer

## Overview

Giffer turns a **series of photos** — from a supported archive or a directory — into a single animated GIF. Delivery is phased: **Phase 1 is the CLI** (required). **Phase 2 is a local desktop UI** (embedded webview) that reuses the Phase 1 conversion core and the same parameters.

## Phases

| Phase | Scope | Status |
|-------|--------|--------|
| **1 — CLI** | Archive/dir path in → GIF on disk; batch mode for `upload/`; interactive wizard; all conversion behavior and parameters | Required — shipped |
| **2 — UI** | Thin local upload UI in a native window wrapping the same converter and parameters (`giffer ui` or double-click) | Optional — shipped |

Phase 1 is complete when the CLI implements this spec end to end. Phase 2 is optional but is implemented and must not fork conversion logic.

---

## Distribution

**Current release: [v1.1.1](https://github.com/alienfacepalm/giffer/releases/latest)** (Windows, Linux, macOS).

### Quick start (users)

| OS | Steps |
|----|-------|
| **Windows** | Download `giffer-windows-amd64.exe` → double-click → drop photo zip → Convert |
| **macOS** | Download `Giffer-darwin-*.app.zip` → unzip → double-click `Giffer.app` → drop photo zip → Convert |
| **Linux** | Download `giffer-linux-*` → `chmod +x giffer` → run or double-click → drop photo zip → Convert |

Full download links and troubleshooting: [`release/README.md`](release/README.md) and [GitHub Releases](https://github.com/alienfacepalm/giffer/releases/latest).

### Release layout

Prebuilt native apps are committed under `release/<platform>/` and published on GitHub Releases:

| Platform | Path | Run |
|----------|------|-----|
| Windows (x64) | `release/windows-amd64/giffer.exe` | Double-click |
| Windows (x86) | `release/windows-386/giffer.exe` | Double-click |
| Linux (x64) | `release/linux-amd64/giffer` | `chmod +x`, then double-click or `./giffer` |
| Linux (ARM64) | `release/linux-arm64/giffer` | same |
| macOS (Intel) | `release/darwin-amd64/Giffer.app` | Double-click |
| macOS (Apple Silicon) | `release/darwin-arm64/Giffer.app` | Double-click |

GitHub Releases include flat download names (`giffer-windows-amd64.exe`, …), `Giffer-darwin-*.app.zip`, and `SHA256SUMS`. Pushing a `v*` tag runs CI and uploads assets. Rebuild committed copies with `make release`.

Double-clicking opens the Phase 2 UI in a native window. Terminal `./giffer` with no flags on a TTY runs the Phase 1 wizard. `giffer ui` always opens the UI window.

---

## Phase 1 — CLI

### Pipeline

```text
Archive or directory input → Extract/collect images → Sort by filename → Resize to max width → Encode GIF → GIF output
```

Frame decode (read/decompress) and frame encode (resize + palette) run on a worker pool sized to `GOMAXPROCS` (capped). Frame order stays filename-sorted; only independent per-frame work is parallel.

### Inputs

- Place input photo archives and/or photo directories in the project `upload/` directory.
- Supported sources: a photo archive, or a directory of images (path may be absolute or relative; examples use `upload/<name>.zip`, `upload/<name>.tar.gz`, or `upload/<name>/`).
- Supported archive formats: `.zip`, `.tar`, `.tar.gz` / `.tgz`, `.tar.bz2` / `.tbz2` / `.tbz`, `.tar.xz` / `.txz`, `.7z`, `.rar`.
- Supported image types: `jpg` / `jpeg`, `png`, `webp`, and still `gif` frames treated as single images.
- JPEG (and other EXIF-bearing) frames are auto-rotated using the EXIF Orientation tag so phone photos are right-side up in the GIF.
- Non-image files and junk paths (for example `__MACOSX` and `.DS_Store`) are ignored.
- Nested folders are allowed; only the file basename is used for sorting and type detection.
- If the archive/directory is invalid, unreadable, empty of supported images, or contains no usable frames after filtering, the run fails with a clear error.

### Frame order

Frames are ordered by ascending case-insensitive filename (basename only). This is a fixed rule, not a user parameter. EXIF date sorting is out of scope.

### Modes

#### Interactive wizard (no flags, terminal)

```bash
giffer
```

When stdin is a terminal and no flags are passed, giffer runs a short CLI wizard that asks for mode (batch vs single), then `delay-ms`, `max-width`, `loop`, and (for single mode) `input` / `output`. Confirm before converting.

The wizard uses a decorative banner, emoji prompts, and (on a TTY) ANSI colors. During conversion it may show a live progress line on stderr (`reading` → `encoding` → `writing`).

#### Batch mode (no `--input`)

```bash
giffer
giffer --delay-ms 100
```

- In a **non-interactive** context (piped stdin, CI, scripts), bare `giffer` runs batch mode.
- With any tunable flag set and no `--input`, batch mode always runs.
- Scans top-level entries under `upload/`.
- Each supported archive → `upload/<stem>.gif` (compound suffixes like `.tar.gz` strip correctly: `photos.tar.gz` → `photos.gif`).
- Each directory that contains at least one supported image → `upload/<dirname>.gif`.
- If the target `.gif` already exists, skip that source (stdout line contains `skip <path>`; may include emoji).
- Sources without images (empty dirs, unsupported files, existing `.gif` files) are ignored.
- If two sources would write the same output path (for example `photos.zip` and `photos/`), fail with an output collision error.
- Conversions run in parallel (worker pool sized to `GOMAXPROCS` / CPU count).
- Tunable flags (`--delay-ms`, `--max-width`, `--loop`) still apply to every job.
- If `upload/` is missing → exit `1`. If nothing to convert (empty or all skipped) → exit `0`.
- On a TTY, stderr may also show a short batch header/summary; this is presentation only.

#### Single-input mode

```bash
giffer --input upload/photos.zip --output out.gif --delay-ms 100 --max-width 0 --loop 0
giffer --input upload/vacation/
```

### Parameters

These are the only user-facing conversion settings.

| Parameter   | Purpose                                                      | Default                                      | Validation                          |
|-------------|--------------------------------------------------------------|----------------------------------------------|-------------------------------------|
| `input`     | Path to photo archive or directory under `upload/`           | omit for batch / wizard                      | supported archive or existing directory |
| `output`    | Destination `.gif` path                                      | same basename as the input, beside the input | must end in `.gif`                  |
| `delay-ms`  | Milliseconds each frame is shown. Default `100` matches common GIF practice (encoded in 10ms units; very short delays are unreliable across browsers). | `100` | integer `> 0` |
| `max-width` | Max frame width in px; height scales to preserve aspect ratio. `0` means use the first sorted photo's native width as the baseline. | `0` | integer `≥ 0` |
| `loop`      | GIF loop count; `0` means loop forever                       | `0`                                          | integer `≥ 0`                       |

Images already narrower than `max-width` are left at their native width. When `max-width` is `0`, the converter reads the first usable frame (after filename sort) and uses that frame's width for every frame.

### Output

- Output is a single animated GIF per source.
- **Single-input mode:** if the destination file already exists, overwrite it and print a short warning to stderr (message contains `warning: overwriting`).
- **Batch mode:** if the destination file already exists, skip conversion for that source.
- Aspect ratio is preserved when resizing; frames are not cropped.

### CLI presentation

- Interactive / TTY runs may use emoji, ANSI color, banners, and progress bars for clarity.
- Non-TTY / scripted runs stay machine-friendly:
  - Success: GIF path alone on stdout (no required prefix).
  - Batch skip: stdout line contains `skip <path>`.
  - Errors: clear text on stderr (may include an emoji prefix); exit codes unchanged.
- Decoration must not change exit codes or the parseable substrings above.

### CLI sketch

```bash
giffer
giffer --input upload/photos.zip --output out.gif --delay-ms 100 --max-width 0 --loop 0
giffer ui
```

| Flag / command | Maps to     |
|----------------|-------------|
| `--input`      | `input`     |
| `--output`     | `output`    |
| `--delay-ms`   | `delay-ms`  |
| `--max-width`  | `max-width` |
| `--loop`       | `loop`      |
| `ui`           | Phase 2 local UI |

With **no flags** on a terminal, giffer opens the interactive wizard. Omitting `--input` in a non-interactive run (or with other flags set) runs batch mode on `upload/`. Omitting `--output` in single-input mode writes `<input-basename>.gif` next to the input (typically under `upload/`). Omitting the tunable flags uses the defaults above.

### Success and failure (CLI)

| Outcome              | Behavior                                  |
|----------------------|-------------------------------------------|
| Success              | Exit `0`; GIF written; path(s) on stdout  |
| Batch skip           | Exit `0` if nothing failed; `skip` lines on stdout |
| Bad / unreadable archive | Exit `1`; message on stderr           |
| No supported images  | Exit `1`; message on stderr               |
| Output collision     | Exit `1`; message on stderr               |
| Invalid parameters   | Exit `2`; message on stderr               |
| Write failure        | Exit `1`; message on stderr               |
| Batch partial failure| Exit `1`; per-job errors on stderr        |

### Non-goals (Phase 1)

- Replacing the CLI with a remote or cloud service
- Video input
- Cloud upload or remote processing
- Image editing (crop, filters, text overlays)
- Advanced GIF optimization / palette / quality controls
- FPS as a separate control (use `delay-ms` only)
- Fit/crop modes, reverse playback, frame ranges, per-frame delays
- EXIF-based or custom sort orders

---

## Phase 2 — UI (optional, shipped)

Must not change Phase 1 conversion behavior; call the same conversion core and parameters.

### Launch

- **Double-click** the release binary (no attached console) → native UI window.
- **`giffer ui`** → same native window.
- Terminal **`giffer`** with no flags on a TTY → Phase 1 wizard (unchanged).

### Command

```bash
giffer ui
giffer ui --addr 127.0.0.1:8765 --upload-dir upload
```

| Flag | Default | Meaning |
|------|---------|---------|
| `--addr` | `127.0.0.1:8765` | Listen address; **never remapped** to another port; loopback only unless `--allow-remote` |
| `--upload-dir` | beside the binary (`<exe-dir>/upload`) | Directory for uploaded archives and GIF output; during `go run` / `go test` falls back to `upload/` in the working directory |
| `--allow-remote` | off | Allow non-loopback `--addr` (exposes the convert API on the network) |

- The UI is fully self-contained in the release binary (HTML/CSS/JS, Three.js, fonts). No CDN or network is required. The UI runs in an **embedded OS webview** (native window) — not an external browser tab.
- A local HTTP server on `--addr` serves the embedded assets and conversion API to the webview.
- If the configured address is already in use by another giffer UI, giffer **reclaims** that port (stops the prior giffer listener) and binds the same address — it does not pick a free alternate port, and it will not kill unrelated processes.

### Platform requirements (embedded webview)

| OS | Runtime | Build (release) |
|----|---------|-----------------|
| Windows | WebView2 Runtime (Windows 10/11) | go-webview2, no CGO |
| Linux | `libwebkit2gtk-4.1-0` (or 4.0) | `libwebkit2gtk-4.0-dev`, CGO |
| macOS | System WebKit | Xcode CLT, CGO |

### UI sketch

- Photo-archive drop zone or file picker (`input`); copy explains a **series of photos in one archive**; supported formats shown as compact chips (zip, tar.gz, tar.bz2, tar.xz, tar, 7z, rar). Chosen sources are treated like files under `upload/`.
- Fields for `delay-ms`, `max-width`, and `loop` (pre-filled with Phase 1 defaults).
- Output as download and/or path chooser (`output`).
- One primary **Convert** action and a **Reset** control that restores defaults (file cleared, `delay-ms` / `max-width` / `loop` to Phase 1 defaults, progress/status/preview cleared; aborts an in-flight convert).
- Realtime progress while converting (reading → encoding → writing); clear error text on failure; success state with the resulting GIF available.
- Progress events always include numeric `done`, `total`, and `percent` (zero is sent, not omitted). User-visible copy must never show `undefined`, `null`, or `NaN`.
- Branding and contrast: UI keeps brand text readable against the page background (including light/dark contrast schemes as needed).

### Success and failure (UI)

| Outcome              | Behavior                         |
|----------------------|----------------------------------|
| Success              | Success status + GIF ready       |
| Bad / unreadable archive | Error message                |
| No supported images  | Error message                    |
| Invalid parameters   | Inline validation errors         |
| Write failure        | Error message                    |

### Non-goals (Phase 2)

- Replacing or forking the CLI converter
- Cloud upload
- Editing tools or advanced GIF optimization UI
- Remapping to a different listen port when the configured address is busy

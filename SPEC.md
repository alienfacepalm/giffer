# Giffer

## Overview

Giffer turns a zip of photos (or a directory of photos) into a single animated GIF. Delivery is phased: **Phase 1 is the CLI** (required). **Phase 2 is an optional local UI** that may never ship. If Phase 2 is built, it reuses the Phase 1 conversion core and the same parameters.

## Phases

| Phase | Scope | Status |
|-------|--------|--------|
| **1 — CLI** | Zip/dir path in → GIF on disk; batch mode for `upload/`; all conversion behavior and parameters | Required |
| **2 — UI** | Thin local upload UI wrapping the same converter and parameters | Optional |

Phase 1 is complete when the CLI implements this spec end to end. Phase 2 is not required for a finished product.

---

## Phase 1 — CLI

### Pipeline

```text
Zip or directory input → Extract/collect images → Sort by filename → Resize to max width → Encode GIF → GIF output
```

### Inputs

- Place input `.zip` archives and/or photo directories in the project `upload/` directory.
- Supported sources: a `.zip` archive, or a directory of images (path may be absolute or relative; examples use `upload/<name>.zip` or `upload/<name>/`).
- Supported image types: `jpg` / `jpeg`, `png`, `webp`, and still `gif` frames treated as single images.
- Non-image files and junk paths (for example `__MACOSX` and `.DS_Store`) are ignored.
- Nested folders are allowed; only the file basename is used for sorting and type detection.
- If the zip/directory is invalid, unreadable, empty of supported images, or contains no usable frames after filtering, the run fails with a clear error.

### Frame order

Frames are ordered by ascending case-insensitive filename (basename only). This is a fixed rule, not a user parameter. EXIF date sorting is out of scope.

### Modes

#### Batch mode (no `--input`)

```bash
giffer
```

- Scans top-level entries under `upload/`.
- Each `.zip` → `upload/<basename>.gif`.
- Each directory that contains at least one supported image → `upload/<dirname>.gif`.
- If the target `.gif` already exists, skip that source (print `skip <path>`).
- Sources without images (empty dirs, non-zip files, existing `.gif` files) are ignored.
- If two sources would write the same output path (for example `photos.zip` and `photos/`), fail with an output collision error.
- Conversions run in parallel (worker pool sized to `GOMAXPROCS` / CPU count).
- Tunable flags (`--delay-ms`, `--max-width`, `--loop`) still apply to every job.
- If `upload/` is missing → exit `1`. If nothing to convert (empty or all skipped) → exit `0`.

#### Single-input mode

```bash
giffer --input upload/photos.zip --output out.gif --delay-ms 500 --max-width 800 --loop 0
giffer --input upload/vacation/
```

### Parameters

These are the only user-facing settings.

| Parameter   | Purpose                                                      | Default                                      | Validation                          |
|-------------|--------------------------------------------------------------|----------------------------------------------|-------------------------------------|
| `input`     | Path to `.zip` or photo directory under `upload/`            | omit for batch mode over `upload/`           | `.zip` or existing directory        |
| `output`    | Destination `.gif` path                                      | same basename as the input, beside the input | must end in `.gif`                  |
| `delay-ms`  | Milliseconds each frame is shown                             | `500`                                        | integer `> 0`                       |
| `max-width` | Max frame width in px; height scales to preserve aspect ratio | `800`                                       | integer `≥ 1`                       |
| `loop`      | GIF loop count; `0` means loop forever                       | `0`                                          | integer `≥ 0`                       |

Images already narrower than `max-width` are left at their native width.

### Output

- Output is a single animated GIF per source.
- **Single-input mode:** if the destination file already exists, overwrite it and print a short warning to stderr.
- **Batch mode:** if the destination file already exists, skip conversion for that source.
- Aspect ratio is preserved when resizing; frames are not cropped.

### CLI sketch

```bash
giffer
giffer --input upload/photos.zip --output out.gif --delay-ms 500 --max-width 800 --loop 0
```

| Flag          | Maps to     |
|---------------|-------------|
| `--input`     | `input`     |
| `--output`    | `output`    |
| `--delay-ms`  | `delay-ms`  |
| `--max-width` | `max-width` |
| `--loop`      | `loop`      |

Omitting `--input` runs batch mode on `upload/`. Omitting `--output` in single-input mode writes `<input-basename>.gif` next to the input (typically under `upload/`). Omitting the tunable flags uses the defaults above.

### Success and failure (CLI)

| Outcome              | Behavior                                  |
|----------------------|-------------------------------------------|
| Success              | Exit `0`; GIF written; path(s) on stdout  |
| Batch skip           | Exit `0` if nothing failed; `skip` lines on stdout |
| Bad / unreadable zip | Exit `1`; message on stderr               |
| No supported images  | Exit `1`; message on stderr               |
| Output collision     | Exit `1`; message on stderr               |
| Invalid parameters   | Exit `2`; message on stderr               |
| Write failure        | Exit `1`; message on stderr               |
| Batch partial failure| Exit `1`; per-job errors on stderr        |

### Non-goals (Phase 1)

- UI of any kind
- Video input
- Cloud upload or remote processing
- Image editing (crop, filters, text overlays)
- Advanced GIF optimization / palette / quality controls
- FPS as a separate control (use `delay-ms` only)
- Fit/crop modes, reverse playback, frame ranges, per-frame delays
- EXIF-based or custom sort orders

---

## Phase 2 — UI (optional)

Build only if needed. Must not change Phase 1 behavior; call the same conversion core and parameters.

### UI sketch

- Zip/directory drop zone or file picker (`input`); chosen sources are treated like files under `upload/`.
- Fields for `delay-ms`, `max-width`, and `loop` (pre-filled with Phase 1 defaults).
- Output as download and/or path chooser (`output`).
- One primary **Convert** action.
- Progress while converting; clear error text on failure; success state with the resulting GIF available.

### Success and failure (UI)

| Outcome              | Behavior                         |
|----------------------|----------------------------------|
| Success              | Success status + GIF ready       |
| Bad / unreadable zip | Error message                    |
| No supported images  | Error message                    |
| Invalid parameters   | Inline validation errors         |
| Write failure        | Error message                    |

### Non-goals (Phase 2)

- Replacing or forking the CLI converter
- Cloud upload
- Editing tools or advanced GIF optimization UI

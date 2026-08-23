# Giffer — SPEC

> **TL;DR** — CLI turns zips / images / folders into one GIF. Optional UI later must call the same core.
> 📦 + 🖼️ + 📁 → 🔤 → ↔️ → 🎞️ · Phase 1 required · Phase 2 optional

| Phase | Scope | Status |
|-------|--------|:------:|
| **1 — CLI** | Inputs → GIF on disk; all conversion + params | ✅ Required |
| **2 — UI** | Thin local UI over the same converter | ⬜ Optional |

Phase 1 done = CLI matches this spec end-to-end. Phase 2 is **not** required for a finished product.

User-facing how-to: [README.md](README.md)

---

## Phase 1 — CLI

### 🔄 Pipeline

**TL;DR:** collect → sort by name → resize → encode GIF.

```text
Inputs (zip / files / dirs) → Collect images → Sort by filename → Resize to max width → Encode GIF → GIF output
```

### 📥 Inputs

**TL;DR:** Prefer `upload/`. Accept zip, image, or directory (mix OK). Merge all frames into one GIF. Ignore junk. Fail clearly if nothing usable.

<details>
<summary>📋 Full rules</summary>

- Prefer placing inputs under the project `upload/` directory.
- Each input path may be:
  - a `.zip` archive of images
  - an individual image file (`jpg` / `jpeg`, `png`, `webp`, still `gif`)
  - a directory of images (searched recursively)
- Multiple inputs are allowed (repeated `--input` and/or positional path arguments). All collected images are merged into one GIF.
- Non-image files and junk paths (for example `__MACOSX` and `.DS_Store`) are ignored.
- Nested folders are allowed; only the file basename is used for sorting and type detection.
- If an input is invalid/unreadable, empty of supported images, or yields no usable frames after filtering, the run fails with a clear error.

</details>

### 🔤 Frame order

**TL;DR:** A→Z by basename (case-insensitive). Ties → full path. Fixed rule — no EXIF / custom sort.

<details>
<summary>📋 Full rules</summary>

Frames are ordered by ascending case-insensitive filename (basename only). Ties break on full source path. This is a fixed rule, not a user parameter. EXIF date sorting is out of scope.

</details>

### 🎛️ Parameters

**TL;DR:** Only these five knobs. Defaults: delay `500`, width `800`, loop forever.

| Parameter   | Purpose                                                       | Default                                                             | Validation                                      |
|-------------|---------------------------------------------------------------|---------------------------------------------------------------------|-------------------------------------------------|
| `input`     | One or more paths: zip, image file, or directory              | required (at least one)                                             | each path must be zip, supported image, or dir  |
| `output`    | Destination `.gif` path                                       | single input: `<basename>.gif` beside it; multiple: `animation.gif` | must end in `.gif`                              |
| `delay-ms`  | Milliseconds each frame is shown                              | `500`                                                               | integer `> 0`                                   |
| `max-width` | Max frame width in px; height scales to preserve aspect ratio | `800`                                                               | integer `≥ 1`                                   |
| `loop`      | GIF loop count; `0` means loop forever                        | `0`                                                                 | integer `≥ 0`                                   |

Images already narrower than `max-width` are left at their native width.

### 💾 Output

**TL;DR:** One animated GIF. Overwrite if exists (warn). Never crop — aspect preserved.

<details>
<summary>📋 Full rules</summary>

- Output is a single animated GIF.
- If the destination file already exists, overwrite it and print a short warning to stderr.
- Aspect ratio is preserved when resizing; frames are not cropped.

</details>

### 💻 CLI sketch

```bash
giffer --input upload/photos.zip --output out.gif --delay-ms 500 --max-width 800 --loop 0
giffer --input upload/album/
giffer --input upload/a.jpg --input upload/b.png --output upload/out.gif
giffer upload/img1.jpg upload/img2.jpg --output upload/out.gif
```

| Flag / args     | Maps to                  |
|-----------------|--------------------------|
| `--input`       | `input` (repeatable)     |
| path arguments  | additional `input` paths |
| `--output`      | `output`                 |
| `--delay-ms`    | `delay-ms`               |
| `--max-width`   | `max-width`              |
| `--loop`        | `loop`                   |

**Defaults when flags omitted:** one input → `<basename>.gif` beside it (dirs use the directory name). Multiple inputs → `animation.gif` in the cwd. Tunables fall back to the table above.

### ✅ / ❌ Success and failure (CLI)

| Outcome                    | Behavior                              |
|----------------------------|---------------------------------------|
| ✅ Success                 | Exit `0`; GIF written; path on stdout |
| 💥 Bad / unreadable input  | Exit `1`; message on stderr           |
| 📭 No supported images     | Exit `1`; message on stderr           |
| 🚫 Invalid parameters      | Exit `2`; message on stderr           |
| 💾 Write failure           | Exit `1`; message on stderr           |

### 🚫 Non-goals (Phase 1)

**TL;DR:** No UI, video, cloud, editing, advanced GIF knobs, FPS, crop modes, or custom sort.

<details>
<summary>📋 Full list</summary>

- UI of any kind
- Video input
- Cloud upload or remote processing
- Image editing (crop, filters, text overlays)
- Advanced GIF optimization / palette / quality controls
- FPS as a separate control (use `delay-ms` only)
- Fit/crop modes, reverse playback, frame ranges, per-frame delays
- EXIF-based or custom sort orders

</details>

---

## Phase 2 — UI (optional)

**TL;DR:** Build only if needed. Same converter + params as Phase 1. Drop zone → knobs → Convert → GIF.

### 🖼️ UI sketch

- Drop zone or file/folder picker for one or more inputs (zips, images, directories).
- Fields for `delay-ms`, `max-width`, and `loop` (pre-filled with Phase 1 defaults).
- Output as download and/or path chooser (`output`).
- One primary **Convert** action.
- Progress while converting; clear error text on failure; success state with the resulting GIF available.

### ✅ / ❌ Success and failure (UI)

| Outcome                    | Behavior                     |
|----------------------------|------------------------------|
| ✅ Success                 | Success status + GIF ready   |
| 💥 Bad / unreadable input  | Error message                |
| 📭 No supported images     | Error message                |
| 🚫 Invalid parameters      | Inline validation errors     |
| 💾 Write failure           | Error message                |

### 🚫 Non-goals (Phase 2)

**TL;DR:** No forked converter, cloud, or editing / advanced optimization UI.

<details>
<summary>📋 Full list</summary>

- Replacing or forking the CLI converter
- Cloud upload
- Editing tools or advanced GIF optimization UI

</details>

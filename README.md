# giffer

> **TL;DR** — Drop photos under `upload/`, run `giffer`, get one animated GIF.
> Pipeline: 📦/🖼️/📁 → 🔤 sort → ↔️ resize → 🎞️ GIF

| | |
|---|---|
| **What** | Photos → animated GIF (CLI) |
| **Inputs** | `.zip`, image files, or folders (mix OK) |
| **Defaults** | `500ms` / frame · max width `800px` · loop forever |
| **Status** | Phase 1 CLI ✅ · Phase 2 UI optional |

Full product rules: [SPEC.md](SPEC.md)

---

## ⚡ Quick start

```bash
# build
make build          # → bin/giffer
# or
go build -o bin/giffer ./cmd/giffer

# convert (put media in upload/)
./bin/giffer --input upload/photos.zip
./bin/giffer --input upload/album/
./bin/giffer upload/a.jpg upload/b.png --output upload/out.gif
```

**TL;DR:** one input → GIF beside it; many inputs → `animation.gif` in the cwd unless `--output` is set.

---

## 🎛️ Flags

| Flag | Icon | Default | Notes |
|------|:----:|---------|-------|
| `--input` / paths | 📥 | *required* | Repeatable; zip, image, or dir |
| `--output` | 💾 | beside input / `animation.gif` | Must end in `.gif` |
| `--delay-ms` | ⏱️ | `500` | Frame duration (`> 0`) |
| `--max-width` | ↔️ | `800` | Height scales; narrower images untouched |
| `--loop` | 🔁 | `0` | `0` = forever |

```bash
giffer --input upload/photos.zip --output out.gif --delay-ms 500 --max-width 800 --loop 0
```

---

## 📥 Inputs

**TL;DR:** Anything under `upload/` that is a zip, `jpg`/`png`/`webp`/`gif`, or a folder of those. Junk (`__MACOSX`, `.DS_Store`) is skipped. Frames sort by **filename** (A→Z, case-insensitive).

<details>
<summary>📋 Details</summary>

- Prefer `upload/` for zips, loose images, and photo dirs (keep the repo root clean).
- Multiple `--input` flags and/or positional paths merge into **one** GIF.
- Nested folders OK; only the **basename** matters for sort + type.
- Empty / unreadable / no usable images → clear error, exit `1`.

</details>

---

## ✅ / ❌ Outcomes

| Result | Exit | Where |
|--------|:----:|-------|
| ✅ Success | `0` | GIF path on stdout |
| 💥 Bad input / no images / write fail | `1` | stderr |
| 🚫 Invalid flags | `2` | stderr |

Existing output files are **overwritten** (short warning on stderr).

---

## 🧪 Dev

```bash
make check    # tidy · fmt · vet · test
make test
make vet
```

---

## 🗺️ Roadmap

| Phase | Scope | |
|-------|--------|:-:|
| **1 — CLI** | Convert + all parameters | ✅ |
| **2 — UI** | Optional local wrapper of the same core | ⬜ |

Not in scope: video, cloud, editing tools, EXIF sort, advanced GIF tuning. See [SPEC.md](SPEC.md).

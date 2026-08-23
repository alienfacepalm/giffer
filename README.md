# giffer

Turn zips or directories of photos into animated GIFs.

## Usage

Place photo zips and/or photo directories under `upload/`, then:

```bash
./giffer
```

That scans `upload/`, skips any source that already has a matching `.gif`, and converts the rest in parallel.

Single conversion:

```bash
./giffer --input upload/photos.zip
./giffer --input upload/vacation/
```

| Flag | Default | Meaning |
|------|---------|---------|
| `--input` | (batch `upload/`) | `.zip` or photo directory |
| `--output` | beside input | destination `.gif` |
| `--delay-ms` | `500` | frame delay |
| `--max-width` | `800` | max frame width |
| `--loop` | `0` | `0` = loop forever |

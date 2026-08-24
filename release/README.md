# giffer downloads

Statically linked binaries (no CGO, no Go install). Pick your platform folder, download the binary inside, and run it.

| Folder | OS / CPU | Binary |
|--------|----------|--------|
| [`windows-amd64/`](windows-amd64/) | Windows x64 | `giffer.exe` |
| [`linux-amd64/`](linux-amd64/) | Linux x64 | `giffer` |
| [`linux-arm64/`](linux-arm64/) | Linux ARM64 | `giffer` |
| [`darwin-amd64/`](darwin-amd64/) | macOS Intel | `giffer` |
| [`darwin-arm64/`](darwin-arm64/) | macOS Apple Silicon | `giffer` |

## Quick start

**Linux / macOS**

```bash
chmod +x giffer
./giffer
```

**Windows (PowerShell)**

```powershell
.\giffer.exe
```

Put photo archives or folders in an `upload/` directory next to the binary (or pass `--input`). Local UI: `./giffer ui` (or `giffer.exe ui` on Windows).

Tagged [GitHub Releases](https://github.com/alienfacepalm/giffer/releases) mirror these builds and include `SHA256SUMS`.

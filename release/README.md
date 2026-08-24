# giffer downloads

Binaries with an embedded webview for the convert UI. Pick your platform folder, download the binary, and run it.

| Folder | OS / CPU | Binary |
|--------|----------|--------|
| [`windows-amd64/`](windows-amd64/) | Windows x64 | `giffer.exe` |
| [`windows-386/`](windows-386/) | Windows x86 (32-bit) | `giffer.exe` |
| [`linux-amd64/`](linux-amd64/) | Linux x64 | `giffer` |
| [`linux-arm64/`](linux-arm64/) | Linux ARM64 | `giffer` |
| [`darwin-amd64/`](darwin-amd64/) | macOS Intel | `giffer` |
| [`darwin-arm64/`](darwin-arm64/) | macOS Apple Silicon | `giffer` |

## Quick start

**Double-click** `giffer` / `giffer.exe` to open the convert UI in a native window.

**Linux / macOS (terminal)**

```bash
chmod +x giffer
./giffer          # wizard on a TTY
./giffer ui       # UI window
```

**Windows (PowerShell)**

```powershell
.\giffer.exe      # wizard in a terminal
.\giffer.exe ui   # UI window
```

Put photo archives or folders in an `upload/` directory next to the binary (or pass `--input`).

| OS | Notes |
|----|-------|
| Windows | WebView2 Runtime (included on Windows 10/11) |
| Linux | Install `libwebkit2gtk-4.1-0` (or 4.0) if missing |
| macOS | Uses system WebKit |

Tagged [GitHub Releases](https://github.com/alienfacepalm/giffer/releases) mirror these builds and include `SHA256SUMS`.

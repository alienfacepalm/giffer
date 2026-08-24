# giffer downloads

Native GUI apps with an embedded webview for the convert UI. Pick your platform folder, download, and double-click to run.

| Folder | OS / CPU | Double-click | Terminal |
|--------|----------|--------------|----------|
| [`windows-amd64/`](windows-amd64/) | Windows x64 | `giffer.exe` | `giffer.exe ui` |
| [`windows-386/`](windows-386/) | Windows x86 (32-bit) | `giffer.exe` | `giffer.exe ui` |
| [`linux-amd64/`](linux-amd64/) | Linux x64 | `giffer` | `./giffer ui` |
| [`linux-arm64/`](linux-arm64/) | Linux ARM64 | `giffer` | `./giffer ui` |
| [`darwin-amd64/`](darwin-amd64/) | macOS Intel | `Giffer.app` | `./giffer ui` |
| [`darwin-arm64/`](darwin-arm64/) | macOS Apple Silicon | `Giffer.app` | `./giffer ui` |

## Quick start

**Windows:** double-click `giffer.exe`.

**macOS:** double-click `Giffer.app` (bare `giffer` binary also included for terminal use).

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

# Download giffer

**Latest release: [v1.1.1](https://github.com/alienfacepalm/giffer/releases/latest)**

Turn a folder or zip of photos into an animated GIF. No install, no terminal required — just download and double-click.

---

## Windows

1. **Download** [`giffer.exe`](https://github.com/alienfacepalm/giffer/releases/latest/download/giffer-windows-amd64.exe) (64-bit) or [`giffer.exe` 32-bit](https://github.com/alienfacepalm/giffer/releases/latest/download/giffer-windows-386.exe)
2. **Double-click** `giffer.exe`
3. **Drop** your photo zip (or pick a file) in the window → click **Convert**

Your GIF saves in an `upload/` folder next to the exe.

---

## Mac

1. **Download** [`Giffer.app.zip`](https://github.com/alienfacepalm/giffer/releases/latest/download/Giffer-darwin-arm64.app.zip) (Apple Silicon) or [Intel version](https://github.com/alienfacepalm/giffer/releases/latest/download/Giffer-darwin-amd64.app.zip)
2. **Unzip**, then **double-click** `Giffer.app`
3. **Drop** your photo zip in the window → click **Convert**

First launch: if macOS blocks the app, right-click `Giffer.app` → **Open** → **Open** again.

Your GIF saves in an `upload/` folder next to the app.

---

## Linux

1. **Download** [`giffer`](https://github.com/alienfacepalm/giffer/releases/latest/download/giffer-linux-amd64) (64-bit) or [ARM64 version](https://github.com/alienfacepalm/giffer/releases/latest/download/giffer-linux-arm64)
2. **Make it executable**, then run it:
   ```bash
   chmod +x giffer
   ./giffer
   ```
   Or double-click in your file manager after allowing execute permission.
3. **Drop** your photo zip in the window → click **Convert**

If the window does not open, install WebKit once: `sudo apt install libwebkit2gtk-4.1-0` (Ubuntu/Debian).

Your GIF saves in an `upload/` folder next to the binary.

---

## Supported photo inputs

`.zip`, `.tar`, `.tar.gz`, `.7z`, `.rar`, and folders of `.jpg` / `.png` / `.webp` images.

---

## All platforms (browse files)

| Your computer | Download from repo |
|---------------|-------------------|
| Windows 64-bit | [`windows-amd64/giffer.exe`](windows-amd64/giffer.exe) |
| Windows 32-bit | [`windows-386/giffer.exe`](windows-386/giffer.exe) |
| Linux 64-bit | [`linux-amd64/giffer`](linux-amd64/giffer) |
| Linux ARM | [`linux-arm64/giffer`](linux-arm64/giffer) |
| Mac Intel | [`darwin-amd64/Giffer.app`](darwin-amd64/Giffer.app) |
| Mac Apple Silicon | [`darwin-arm64/Giffer.app`](darwin-arm64/Giffer.app) |

Checksums: [`SHA256SUMS`](SHA256SUMS)

---

## Terminal / CLI (optional)

```bash
./giffer ui              # open the UI window
./giffer --input photos.zip   # convert from the command line
```

See the [main README](../README.md) for all flags.

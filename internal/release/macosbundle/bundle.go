package macosbundle

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// Bundle wraps binPath as a double-clickable macOS .app at appPath.
func Bundle(binPath, appPath, version string) error {
	macosDir := filepath.Join(appPath, "Contents", "MacOS")
	if err := os.MkdirAll(macosDir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(macosDir, "giffer")
	if err := copyFile(binPath, dest); err != nil {
		return err
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return err
	}

	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>giffer</string>
	<key>CFBundleIdentifier</key>
	<string>com.alienfacepalm.giffer</string>
	<key>CFBundleName</key>
	<string>Giffer</string>
	<key>CFBundleDisplayName</key>
	<string>Giffer</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>%s</string>
	<key>CFBundleVersion</key>
	<string>%s</string>
	<key>LSMinimumSystemVersion</key>
	<string>10.13</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>LSUIElement</key>
	<false/>
</dict>
</plist>
`, version, version)
	return os.WriteFile(plistPath, []byte(plist), 0o644)
}

// ReadVersion parses **Release:** vX.Y.Z from readmePath.
func ReadVersion(readmePath string) string {
	data, err := os.ReadFile(readmePath)
	if err != nil {
		return "1.0.0"
	}
	re := regexp.MustCompile(`\*\*Release:\*\* v(\d+\.\d+\.\d+)`)
	if m := re.FindSubmatch(data); len(m) == 2 {
		return string(m[1])
	}
	return "1.0.0"
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

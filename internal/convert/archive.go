package convert

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
	"github.com/ulikunitz/xz"
)

// Default decompression limits for archive extraction and image decode.
const (
	DefaultMaxExtractBytes = 2 << 30   // 2 GiB total
	DefaultMaxEntryBytes   = 256 << 20 // 256 MiB per file
)

// MaxExtractBytes / MaxEntryBytes cap archive extraction and decode reads.
// Tests may lower these; leave defaults for production.
var (
	MaxExtractBytes int64 = DefaultMaxExtractBytes
	MaxEntryBytes   int64 = DefaultMaxEntryBytes
)

// ArchiveKinds lists supported photo-archive extensions for docs and UI.
var ArchiveKinds = []string{
	".zip",
	".tar",
	".tar.gz",
	".tgz",
	".tar.bz2",
	".tbz2",
	".tbz",
	".tar.xz",
	".txz",
	".7z",
	".rar",
}

// IsArchive reports whether name looks like a supported photo archive.
func IsArchive(name string) bool {
	return archiveKind(name) != ""
}

// ArchiveStem returns the basename without the archive suffix
// (photos.tar.gz → photos).
func ArchiveStem(name string) string {
	base := filepath.Base(name)
	lower := strings.ToLower(base)
	for _, suf := range []string{
		".tar.gz", ".tar.bz2", ".tar.xz",
		".tgz", ".tbz2", ".tbz", ".txz",
		".tar", ".zip", ".7z", ".rar",
	} {
		if strings.HasSuffix(lower, suf) {
			return base[:len(base)-len(suf)]
		}
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func archiveKind(name string) string {
	lower := strings.ToLower(filepath.Base(name))
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return "tar.gz"
	case strings.HasSuffix(lower, ".tar.bz2"), strings.HasSuffix(lower, ".tbz2"), strings.HasSuffix(lower, ".tbz"):
		return "tar.bz2"
	case strings.HasSuffix(lower, ".tar.xz"), strings.HasSuffix(lower, ".txz"):
		return "tar.xz"
	case strings.HasSuffix(lower, ".tar"):
		return "tar"
	case strings.HasSuffix(lower, ".zip"):
		return "zip"
	case strings.HasSuffix(lower, ".7z"):
		return "7z"
	case strings.HasSuffix(lower, ".rar"):
		return "rar"
	default:
		return ""
	}
}

// ArchiveToGIF reads images from a supported archive and writes an animated GIF.
// Zip uses the streaming path; other formats extract to a temp directory first.
func ArchiveToGIF(opts Options) error {
	kind := archiveKind(opts.Input)
	if kind == "" {
		return fmt.Errorf("unsupported archive type %q (want %s)",
			filepath.Base(opts.Input), strings.Join(ArchiveKinds, ", "))
	}
	if kind == "zip" {
		return ZipToGIF(opts)
	}

	tmp, err := os.MkdirTemp("", "giffer-archive-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	if err := extractArchive(opts.Input, kind, tmp); err != nil {
		return err
	}

	st, err := summarizeDir(tmp)
	if err != nil {
		return err
	}
	if st.images == 0 {
		return noImagesError("archive", st)
	}

	dirOpts := opts
	dirOpts.Input = tmp
	return DirToGIF(dirOpts)
}

func extractArchive(src, kind, dest string) error {
	lim := newExtractLimiter()
	switch kind {
	case "zip":
		return extractZip(src, dest, lim)
	case "tar":
		return extractTarFile(src, dest, identityReader, lim)
	case "tar.gz":
		return extractTarFile(src, dest, gzipReader, lim)
	case "tar.bz2":
		return extractTarFile(src, dest, bzip2Reader, lim)
	case "tar.xz":
		return extractTarFile(src, dest, xzReader, lim)
	case "7z":
		return extract7z(src, dest, lim)
	case "rar":
		return extractRar(src, dest, lim)
	default:
		return fmt.Errorf("unsupported archive kind %q", kind)
	}
}

type openCompressed func(io.Reader) (io.Reader, error)

func identityReader(r io.Reader) (io.Reader, error) { return r, nil }

func gzipReader(r io.Reader) (io.Reader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	return zr, nil
}

func bzip2Reader(r io.Reader) (io.Reader, error) {
	return bzip2.NewReader(r), nil
}

func xzReader(r io.Reader) (io.Reader, error) {
	xr, err := xz.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("xz: %w", err)
	}
	return xr, nil
}

type extractLimiter struct {
	total    int64
	maxTotal int64
	maxEntry int64
}

func newExtractLimiter() *extractLimiter {
	return &extractLimiter{
		maxTotal: MaxExtractBytes,
		maxEntry: MaxEntryBytes,
	}
}

func (l *extractLimiter) copyFile(target string, r io.Reader) error {
	w, err := os.Create(target)
	if err != nil {
		return err
	}
	limited := io.LimitReader(r, l.maxEntry+1)
	n, copyErr := io.Copy(w, limited)
	closeErr := w.Close()
	if copyErr != nil {
		_ = os.Remove(target)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(target)
		return closeErr
	}
	if n > l.maxEntry {
		_ = os.Remove(target)
		return fmt.Errorf("archive entry exceeds size limit")
	}
	l.total += n
	if l.total > l.maxTotal {
		_ = os.Remove(target)
		return fmt.Errorf("archive extraction exceeds size limit")
	}
	return nil
}

func extractZip(src, dest string, lim *extractLimiter) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("unreadable zip: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if err := writeZipEntry(f, dest, lim); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, dest string, lim *extractLimiter) error {
	name := filepath.ToSlash(f.Name)
	base := path.Base(name)
	if f.FileInfo().IsDir() || shouldSkipPath(name, base) {
		return nil
	}
	target, err := safeExtractPath(dest, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	return lim.copyFile(target, rc)
}

func extractTarFile(src, dest string, open openCompressed, lim *extractLimiter) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("unreadable archive: %w", err)
	}
	defer f.Close()

	r, err := open(f)
	if err != nil {
		return fmt.Errorf("unreadable archive: %w", err)
	}
	if closer, ok := r.(io.Closer); ok {
		defer closer.Close()
	}
	return extractTar(r, dest, lim)
}

func extractTar(r io.Reader, dest string, lim *extractLimiter) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("unreadable tar: %w", err)
		}
		name := filepath.ToSlash(hdr.Name)
		base := path.Base(name)
		if hdr.Typeflag == tar.TypeDir || shouldSkipPath(name, base) {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		target, err := safeExtractPath(dest, name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := lim.copyFile(target, tr); err != nil {
			return err
		}
	}
}

func extract7z(src, dest string, lim *extractLimiter) error {
	r, err := sevenzip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("unreadable 7z: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		base := path.Base(name)
		if f.FileInfo().IsDir() || shouldSkipPath(name, base) {
			continue
		}
		target, err := safeExtractPath(dest, name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = lim.copyFile(target, rc)
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractRar(src, dest string, lim *extractLimiter) error {
	r, err := rardecode.OpenReader(src)
	if err != nil {
		return fmt.Errorf("unreadable rar: %w", err)
	}
	defer r.Close()
	for {
		hdr, err := r.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("unreadable rar: %w", err)
		}
		name := filepath.ToSlash(hdr.Name)
		base := path.Base(name)
		if hdr.IsDir || shouldSkipPath(name, base) {
			continue
		}
		target, err := safeExtractPath(dest, name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := lim.copyFile(target, r); err != nil {
			return err
		}
	}
}

func safeExtractPath(dest, entry string) (string, error) {
	slashEntry := strings.ReplaceAll(entry, "\\", "/")
	for _, part := range strings.Split(slashEntry, "/") {
		if part == ".." {
			return "", fmt.Errorf("refusing unsafe archive path %q", entry)
		}
	}
	clean := path.Clean("/" + slashEntry)
	rel := strings.TrimPrefix(clean, "/")
	if rel == "" || rel == "." {
		return "", fmt.Errorf("invalid archive entry %q", entry)
	}
	target := filepath.Join(dest, filepath.FromSlash(rel))
	relOut, err := filepath.Rel(dest, target)
	if err != nil || strings.HasPrefix(relOut, "..") || filepath.IsAbs(relOut) {
		return "", fmt.Errorf("refusing unsafe archive path %q", entry)
	}
	// filepath.Join drops dest when FromSlash(rel) is absolute (e.g. Windows drive).
	if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) && target != filepath.Clean(dest) {
		return "", fmt.Errorf("refusing unsafe archive path %q", entry)
	}
	return target, nil
}

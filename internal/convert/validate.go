package convert

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ImageKinds lists photo extensions giffer can turn into GIF frames.
var ImageKinds = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

// ImageKindsHelp is the human-readable list of photo types for error copy.
const ImageKindsHelp = "JPEG, PNG, WebP, or GIF"

const imageKindsHelp = ImageKindsHelp

type contentStats struct {
	images   int
	audio    int
	video    int
	docs     int
	other    int
	examples []string // up to a few non-image basenames
}

// CheckPhotoSource verifies path looks like a usable photo archive or folder
// before conversion. Zip is inspected in place; other archives are left to Convert.
func CheckPhotoSource(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if IsArchive(path) {
			return fmt.Errorf("unreadable archive: %w - pick a real photo archive and try again", err)
		}
		return fmt.Errorf("unreadable input: %w - pick a photo folder or archive and try again", err)
	}
	if info.IsDir() {
		st, err := summarizeDir(path)
		if err != nil {
			return err
		}
		if st.images == 0 {
			return noImagesError("folder", st)
		}
		return nil
	}
	if !IsArchive(path) {
		return fmt.Errorf(
			"unsupported file %q - upload a photo archive (%s) filled with %s images",
			filepath.Base(path), strings.Join(ArchiveKinds, ", "), imageKindsHelp)
	}
	if archiveKind(path) == "zip" {
		return checkZipHasPhotos(path)
	}
	return nil
}

func checkZipHasPhotos(zipPath string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("unreadable zip: %w - re-zip your photos as a standard .zip and try again", err)
	}
	defer zr.Close()
	st := summarizeZipNames(zr.File)
	if st.images == 0 {
		return noImagesError("zip", st)
	}
	return nil
}

func summarizeZipNames(files []*zip.File) contentStats {
	var names []string
	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}
		base := path.Base(f.Name)
		if shouldSkipPath(f.Name, base) {
			continue
		}
		names = append(names, base)
	}
	return summarizeNames(names)
}

func summarizeDir(root string) (contentStats, error) {
	var names []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		base := filepath.Base(p)
		if d.IsDir() {
			if p == root {
				return nil
			}
			if shouldSkipPath(relSlash, base) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipPath(relSlash, base) {
			return nil
		}
		names = append(names, base)
		return nil
	})
	if err != nil {
		return contentStats{}, fmt.Errorf("read directory: %w", err)
	}
	return summarizeNames(names), nil
}

func summarizeNames(names []string) contentStats {
	var st contentStats
	for _, base := range names {
		switch fileKind(base) {
		case "image":
			st.images++
		case "audio":
			st.audio++
			st.addExample(base)
		case "video":
			st.video++
			st.addExample(base)
		case "doc":
			st.docs++
			st.addExample(base)
		default:
			st.other++
			st.addExample(base)
		}
	}
	return st
}

func (st *contentStats) addExample(base string) {
	if len(st.examples) >= 3 {
		return
	}
	for _, e := range st.examples {
		if strings.EqualFold(e, base) {
			return
		}
	}
	st.examples = append(st.examples, base)
}

func fileKind(base string) string {
	ext := strings.ToLower(path.Ext(base))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return "image"
	case ".mp3", ".m4a", ".aac", ".wav", ".flac", ".ogg", ".opus", ".aiff", ".wma", ".oga":
		return "audio"
	case ".mp4", ".mov", ".mkv", ".webm", ".avi", ".m4v", ".wmv", ".mpeg", ".mpg":
		return "video"
	case ".pdf", ".txt", ".doc", ".docx", ".rtf", ".md", ".csv", ".xls", ".xlsx":
		return "doc"
	default:
		return "other"
	}
}

func noImagesError(source string, st contentStats) error {
	what := "this " + source
	switch source {
	case "zip":
		what = "this zip"
	case "archive":
		what = "this archive"
	case "folder", "directory":
		what = "this folder"
	}

	var found string
	switch {
	case st.audio > 0 && st.video == 0 && st.docs == 0 && st.other == 0:
		found = fmt.Sprintf("found %s", countPhrase(st.audio, "audio file", "audio files"))
	case st.video > 0 && st.audio == 0 && st.docs == 0 && st.other == 0:
		found = fmt.Sprintf("found %s", countPhrase(st.video, "video file", "video files"))
	case st.docs > 0 && st.audio == 0 && st.video == 0 && st.other == 0:
		found = fmt.Sprintf("found %s", countPhrase(st.docs, "document", "documents"))
	case st.audio+st.video+st.docs+st.other == 0:
		found = "it has no files giffer can use"
	default:
		parts := make([]string, 0, 4)
		if st.audio > 0 {
			parts = append(parts, countPhrase(st.audio, "audio", "audio"))
		}
		if st.video > 0 {
			parts = append(parts, countPhrase(st.video, "video", "video"))
		}
		if st.docs > 0 {
			parts = append(parts, countPhrase(st.docs, "document", "documents"))
		}
		if st.other > 0 {
			parts = append(parts, countPhrase(st.other, "other file", "other files"))
		}
		found = "found " + strings.Join(parts, ", ")
	}
	if len(st.examples) > 0 {
		found += " (e.g. " + strings.Join(st.examples, ", ") + ")"
	}

	return fmt.Errorf(
		"no supported images found in %s - %s. Zip only photos (%s) in the order you want them shown, then try again",
		what, found, imageKindsHelp)
}

func countPhrase(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// UserMessage returns a clear, corrective message for UI/CLI display.
func UserMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no supported images"):
		return msg
	case strings.Contains(msg, "no usable image frames"):
		return "Photos in the archive could not be decoded. Use real " + imageKindsHelp +
			" files (not renamed audio/video), then try again"
	case strings.Contains(msg, "out of bounds"), strings.Contains(msg, "do not fit one canvas"):
		return "Could not encode GIF: photo sizes do not fit one canvas. Set --max-width to a shared size (e.g. 800) and try again"
	case strings.Contains(msg, "too large for GIF"), strings.Contains(msg, "image is too large"):
		return "Photos are too large for GIF (each side must be under 65536px). Lower --max-width and try again"
	case strings.Contains(msg, "could not encode GIF"):
		return msg
	case strings.Contains(msg, "unreadable zip"), strings.Contains(msg, "unreadable archive"):
		if strings.Contains(msg, "try again") {
			return msg
		}
		return msg + " - re-export a standard photo archive and try again"
	case strings.Contains(msg, "unsupported input"), strings.Contains(msg, "unsupported archive"), strings.Contains(msg, "unsupported file"):
		return msg
	case strings.Contains(msg, "encode gif"):
		// Legacy / wrapped stdlib wording — always give a next step.
		return "Could not encode GIF from these photos. Set --max-width (e.g. 800) or use similarly sized photos, then try again"
	default:
		return msg
	}
}

package tools

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/nextlevelbuilder/goclaw/internal/providers"
)

// MediaNameMapFromRefs returns a map from each ref's absolute path to a clean
// display filename. The document path is no longer carried in the <media:document>
// tag (it would invite reading the file on arrival), so the name is recovered
// from the persisted path's basename, dropping the "-<8hex>" upload suffix
// persistMedia adds. Used so team-workspace attachments keep readable names
// instead of the on-disk UUID-suffixed ones.
func MediaNameMapFromRefs(refs []providers.MediaRef) map[string]string {
	nameMap := make(map[string]string)
	for _, ref := range refs {
		if ref.Kind != "document" || ref.Path == "" {
			continue
		}
		nameMap[ref.Path] = stripUploadShortID(filepath.Base(ref.Path))
	}
	return nameMap
}

// copyMediaToWorkspace copies media files to the team workspace attachments dir.
// Uses original filenames from nameMap when available; falls back to UUID filename.
// Appends -1, -2 suffix on collision. Skips files that already exist with same size.
// Returns absolute paths of copied files.
func copyMediaToWorkspace(mediaPaths []string, wsDir string, nameMap map[string]string) []string {
	if len(mediaPaths) == 0 || wsDir == "" {
		return nil
	}

	attachDir := filepath.Join(wsDir, "attachments")
	if err := os.MkdirAll(attachDir, 0750); err != nil {
		slog.Warn("workspace_media: failed to create attachments dir", "dir", attachDir, "error", err)
		return nil
	}

	var copied []string
	for _, src := range mediaPaths {
		srcInfo, err := os.Lstat(src)
		if err != nil {
			slog.Debug("workspace_media: source file not found", "path", src, "error", err)
			continue
		}
		// Skip symlinks to prevent following to unexpected locations.
		if srcInfo.Mode()&os.ModeSymlink != 0 {
			slog.Debug("workspace_media: skipping symlink", "path", src)
			continue
		}

		// Use original filename if available, otherwise fall back to UUID name.
		baseName := filepath.Base(src)
		if origName, ok := nameMap[src]; ok && origName != "" {
			// Strip directory components to prevent path traversal (e.g. "../../etc/crontab").
			baseName = filepath.Base(origName)
		}
		if baseName == "." || baseName == ".." || baseName == "" {
			slog.Debug("workspace_media: skipping invalid filename", "path", src)
			continue
		}

		dst := filepath.Join(attachDir, baseName)

		// Check if already copied (same name + same size = skip).
		if dstInfo, err := os.Stat(dst); err == nil {
			if dstInfo.Size() == srcInfo.Size() {
				copied = append(copied, dst)
				continue
			}
			// Name collision with different file — add suffix.
			dst = deduplicatePath(attachDir, baseName)
		}

		// Always copy (not hard link) to maintain isolation — members modifying
		// workspace files must not affect the original media store.
		if err := copyFile(src, dst); err != nil {
			slog.Warn("workspace_media: failed to copy file", "src", src, "dst", dst, "error", err)
			continue
		}

		slog.Info("workspace_media: copied to team workspace",
			"original", baseName, "src", filepath.Base(src), "dst", dst)
		copied = append(copied, dst)
	}

	return copied
}

// deduplicatePath appends -1, -2, etc. to the filename until it's unique.
func deduplicatePath(dir, name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 100; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return filepath.Join(dir, name) // fallback
}

// copyFile copies src to dst using io.Copy.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

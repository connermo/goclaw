package media

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

// BuildMediaTags generates content tags for media items (matching TS media placeholder format).
// For audio/voice items that have been transcribed, the transcript is embedded in a <transcript> block.
// Items with FromReply=true are annotated with "(from replied message)" so the LLM can distinguish
// media from the current message vs media from the message being replied to.
func BuildMediaTags(mediaList []MediaInfo) string {
	var tags []string
	for _, m := range mediaList {
		var tag string
		switch m.Type {
		case TypeImage:
			if m.SourceURL != "" {
				tag = fmt.Sprintf("<media:image url=%q>", m.SourceURL)
			} else {
				tag = "<media:image>"
			}
		case TypeVideo, TypeAnimation:
			tag = "<media:video>"
		case TypeAudio:
			if m.Transcript != "" {
				tag = fmt.Sprintf("<media:audio>\n<transcript>%s</transcript>", html.EscapeString(m.Transcript))
			} else {
				tag = "<media:audio>"
			}
		case TypeVoice:
			if m.Transcript != "" {
				tag = fmt.Sprintf("<media:voice>\n<transcript>%s</transcript>", html.EscapeString(m.Transcript))
			} else {
				tag = "<media:voice>"
			}
		case TypeDocument:
			if m.FileName != "" {
				tag = fmt.Sprintf("<media:document name=%q>", m.FileName)
			} else {
				tag = "<media:document>"
			}
		}
		if tag != "" {
			if m.FromReply {
				tag += " (from replied message)"
			}
			tags = append(tags, tag)
		}
	}
	return strings.Join(tags, "\n")
}

// textExtensions maps file extensions to MIME types for text files we can extract.
var textExtensions = map[string]string{
	".txt":  "text/plain",
	".md":   "text/markdown",
	".csv":  "text/csv",
	".tsv":  "text/tab-separated-values",
	".json": "application/json",
	".yaml": "text/yaml",
	".yml":  "text/yaml",
	".xml":  "text/xml",
	".log":  "text/plain",
	".ini":  "text/plain",
	".cfg":  "text/plain",
	".env":  "text/plain",
	".sh":   "text/x-shellscript",
	".py":   "text/x-python",
	".go":   "text/x-go",
	".js":   "text/javascript",
	".ts":   "text/typescript",
	".html": "text/html",
	".css":  "text/css",
	".sql":  "text/x-sql",
	".rs":   "text/x-rust",
	".java": "text/x-java",
	".c":    "text/x-c",
	".cpp":  "text/x-c++",
	".h":    "text/x-c",
	".rb":   "text/x-ruby",
	".php":  "text/x-php",
	".toml": "text/x-toml",
}

// DescribeDocument returns a one-line notice announcing an uploaded document:
// what it is, how big it is, and which tool opens it.
//
// It deliberately does not read the file. Uploading a document is not the same
// as asking for it to be analyzed — someone may attach a log and then ask an
// unrelated question, or send a file simply to store it. Inlining the content
// answered that question on the model's behalf, and spent up to 200K chars of
// context doing so. The model can see what arrived and decide for itself.
//
// Text files used to be inlined here while binary files already got a hint;
// both now take the same path, so "was it parsed?" no longer depends on the
// file extension.
func DescribeDocument(filePath, fileName string) (string, error) {
	if filePath == "" {
		return fmt.Sprintf("[File: %s — download failed]", fileName), nil
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	kind := "binary"
	if mime, isText := textExtensions[ext]; isText {
		kind = mime
	}

	var size string
	if fi, err := os.Stat(filePath); err == nil {
		size = ", " + humanSize(fi.Size())
	}

	if isArchiveFileName(fileName) {
		return fmt.Sprintf("[Archive received: %s (archive%s). Not extracted. Use exec to inspect or extract it if the request calls for that.]",
			fileName, size), nil
	}
	return fmt.Sprintf("[File received: %s (%s%s). Not parsed. Use read_document to read it if the request calls for that.]",
		fileName, kind, size), nil
}

// humanSize renders a byte count for the notice line.
func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func isArchiveFileName(fileName string) bool {
	lower := strings.ToLower(fileName)
	return strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".tar") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".tar.bz2") ||
		strings.HasSuffix(lower, ".tbz2") ||
		strings.HasSuffix(lower, ".tar.xz") ||
		strings.HasSuffix(lower, ".txz") ||
		strings.HasSuffix(lower, ".gz")
}

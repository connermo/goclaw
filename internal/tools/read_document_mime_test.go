package tools

import (
	"strings"
	"testing"
)

// Text and source extensions must resolve to a text/* (or JSON/JS) MIME so
// read_document takes its direct-read fast path instead of shipping the bytes
// to a vision model as octet-stream. This is what makes an uploaded .log or
// .py actually readable after the intake stopped inlining it.
func TestMimeFromDocExt_TextExtensionsAreTextReadable(t *testing.T) {
	textExts := []string{
		".txt", ".log", ".md", ".csv", ".tsv", ".json", ".yaml", ".yml",
		".xml", ".html", ".css", ".js", ".ts", ".py", ".go", ".rs",
		".java", ".c", ".cpp", ".h", ".rb", ".php", ".sql", ".sh",
		".ini", ".cfg", ".env",
	}
	for _, ext := range textExts {
		mime := mimeFromDocExt(ext)
		if !textReadableMIMEs[mime] && !strings.HasPrefix(mime, "text/") {
			t.Errorf("%s -> %q: not on read_document fast path", ext, mime)
		}
	}
}

// Binary and archive extensions must NOT masquerade as text — they need the
// vision chain or exec, and a wrong text mime would route them to a raw byte
// dump.
func TestMimeFromDocExt_BinaryStaysBinary(t *testing.T) {
	for _, ext := range []string{".pdf", ".docx", ".xlsx", ".pptx", ".zip", ".tar", ".gz", ".png"} {
		mime := mimeFromDocExt(ext)
		if strings.HasPrefix(mime, "text/") {
			t.Errorf("%s -> %q: binary should not resolve to text/*", ext, mime)
		}
	}
}

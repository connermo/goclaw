package media

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemp creates a file with the given content and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return p
}

// The whole point of the change: a text file's content must NOT appear in the
// notice. Uploading is not a request to analyze.
func TestDescribeDocument_TextFileContentNotInlined(t *testing.T) {
	secret := "SUPER-SECRET-LOG-LINE-42"
	path := writeTemp(t, "app.log", secret+"\nmore lines\n")

	notice, err := DescribeDocument(path, "app.log")
	if err != nil {
		t.Fatalf("DescribeDocument: %v", err)
	}
	if strings.Contains(notice, secret) {
		t.Fatalf("notice inlined file content: %q", notice)
	}
	if !strings.Contains(notice, "app.log") {
		t.Fatalf("notice missing file name: %q", notice)
	}
	if !strings.Contains(notice, "read_document") {
		t.Fatalf("notice missing tool hint: %q", notice)
	}
}

// The notice reports the type (text/plain here) and a human-readable size so
// the model has enough to decide without opening the file.
func TestDescribeDocument_ReportsTypeAndSize(t *testing.T) {
	path := writeTemp(t, "data.csv", strings.Repeat("a,b,c\n", 300)) // ~1.7KB

	notice, err := DescribeDocument(path, "data.csv")
	if err != nil {
		t.Fatalf("DescribeDocument: %v", err)
	}
	if !strings.Contains(notice, "text/csv") {
		t.Fatalf("notice missing mime: %q", notice)
	}
	if !strings.Contains(notice, "KB") {
		t.Fatalf("notice missing size: %q", notice)
	}
}

// Binary files were already hint-only; they must stay that way and read as
// "binary", not leak a wrong text mime.
func TestDescribeDocument_BinaryFile(t *testing.T) {
	path := writeTemp(t, "report.pdf", "%PDF-1.4 binary junk")

	notice, err := DescribeDocument(path, "report.pdf")
	if err != nil {
		t.Fatalf("DescribeDocument: %v", err)
	}
	if !strings.Contains(notice, "binary") {
		t.Fatalf("notice should mark binary: %q", notice)
	}
	if !strings.Contains(notice, "read_document") {
		t.Fatalf("notice missing tool hint: %q", notice)
	}
}

// Archives get their own hint pointing at exec, not read_document.
func TestDescribeDocument_Archive(t *testing.T) {
	path := writeTemp(t, "bundle.zip", "PK\x03\x04 junk")

	notice, err := DescribeDocument(path, "bundle.zip")
	if err != nil {
		t.Fatalf("DescribeDocument: %v", err)
	}
	if !strings.Contains(notice, "exec") {
		t.Fatalf("archive notice should point at exec: %q", notice)
	}
	if strings.Contains(notice, "read_document") {
		t.Fatalf("archive notice should not point at read_document: %q", notice)
	}
}

// A missing/undownloaded file must not error the whole message; it degrades to
// a download-failed notice.
func TestDescribeDocument_EmptyPath(t *testing.T) {
	notice, err := DescribeDocument("", "lost.txt")
	if err != nil {
		t.Fatalf("DescribeDocument: %v", err)
	}
	if !strings.Contains(notice, "lost.txt") || !strings.Contains(notice, "failed") {
		t.Fatalf("unexpected notice for empty path: %q", notice)
	}
}

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{
		0:              "0 B",
		512:            "512 B",
		2048:           "2.0 KB",
		5 * 1024 * 1024: "5.0 MB",
	}
	for n, want := range cases {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

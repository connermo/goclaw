package tools

import (
	"testing"

	"github.com/nextlevelbuilder/goclaw/internal/providers"
)

func TestAttachmentPathByName(t *testing.T) {
	refs := []providers.MediaRef{
		{Kind: "document", Path: "/ws/.uploads/budget-abc12345.txt"},
		// Persisted with the source's extension, which differs from the display
		// name the model was shown (report.txt).
		{Kind: "document", Path: "/ws/.uploads/report-5b1cf499.dat"},
		{Kind: "image", Path: "/ws/.uploads/photo-00000000.jpg"}, // non-document, ignored
	}
	ctx := WithMediaDocRefs(t.Context(), refs)

	cases := []struct {
		name string
		want string
		ok   bool
	}{
		{"budget.txt", "/ws/.uploads/budget-abc12345.txt", true}, // clean-name match
		{"report.txt", "/ws/.uploads/report-5b1cf499.dat", true}, // stem match across ext
		{"report-5b1cf499.dat", "/ws/.uploads/report-5b1cf499.dat", true}, // raw basename
		{"a/b/budget.txt", "/ws/.uploads/budget-abc12345.txt", true},      // basename of a path
		{"photo.jpg", "", false},   // image ref is not a document
		{"missing.txt", "", false}, // no such attachment
	}
	for _, c := range cases {
		got, ok := attachmentPathByName(ctx, c.name)
		if ok != c.ok || got != c.want {
			t.Errorf("attachmentPathByName(%q) = (%q,%v), want (%q,%v)", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestAttachmentPathByName_NoRefs(t *testing.T) {
	if _, ok := attachmentPathByName(t.Context(), "x.txt"); ok {
		t.Fatal("expected no match with no refs in context")
	}
}

func TestIsWithin(t *testing.T) {
	cases := []struct {
		path, root string
		want       bool
	}{
		{"/ws/.uploads/a.txt", "/ws", true},
		{"/ws/a.txt", "/ws", true},
		{"/other/a.txt", "/ws", false},
		{"/ws/../etc/passwd", "/ws", false},
		{"/ws/a.txt", "", false}, // empty root fails closed
	}
	for _, c := range cases {
		if got := isWithin(c.path, c.root); got != c.want {
			t.Errorf("isWithin(%q,%q) = %v, want %v", c.path, c.root, got, c.want)
		}
	}
}

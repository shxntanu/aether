package domain

import "testing"

func TestNewTagNormalizesWhitespaceAndCase(t *testing.T) {
	tag, err := NewTag(TagID("tag-1"), "  Utility   Bills  ")
	if err != nil {
		t.Fatalf("NewTag() error = %v", err)
	}

	if tag.DisplayName != "Utility Bills" {
		t.Fatalf("DisplayName = %q, want %q", tag.DisplayName, "Utility Bills")
	}
	if tag.NormalizedName != "utility bills" {
		t.Fatalf("NormalizedName = %q, want %q", tag.NormalizedName, "utility bills")
	}
}

func TestNewTagRejectsEmptyName(t *testing.T) {
	_, err := NewTag(TagID("tag-1"), " \t ")
	if err == nil {
		t.Fatal("NewTag() error = nil, want validation error")
	}
}

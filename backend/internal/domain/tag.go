package domain

import (
	"fmt"
	"strings"
)

// TagID uniquely identifies a reusable or implicit tag.
type TagID string

// Tag retains user-facing spelling while providing a case-insensitive key.
type Tag struct {
	// ID uniquely identifies the reusable tag.
	ID TagID `json:"id"`
	// DisplayName contains the current user-facing spelling.
	DisplayName string `json:"displayName"`
	// NormalizedName is the case-insensitive identity used for matching.
	NormalizedName string `json:"normalizedName"`
	// Implicit reports whether the tag is managed by the vault rather than by
	// users. Date tags are implicit and can only be changed through a document.
	Implicit bool `json:"implicit"`
}

// NewTag validates and normalizes a tag name.
func NewTag(id TagID, name string) (Tag, error) {
	displayName := strings.Join(strings.Fields(name), " ")
	if displayName == "" {
		return Tag{}, fmt.Errorf("tag name must not be empty")
	}

	return Tag{
		ID:             id,
		DisplayName:    displayName,
		NormalizedName: strings.ToLower(displayName),
	}, nil
}
